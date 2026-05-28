package jobnode

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/api/job/jobrunner"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/singleinstance"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

// Runs jobs!
// This should be compiled into a Go executable that can be run on an EC2 instance (or perhaps on a
// docker container?) that can sit listening for jobs in the DB, run them as they come in. Perhaps
// if no jobs come in for a certain time it should quit, or if it knows it'll be shut down soon it
// should not start new jobs.
// We don't want to start more jobs than we have CPU thread support for, so eg a 4 core maybe with
// hyperthreading should run up to 8 jobs? Maybe just 4...
// When a job finishes we should start a new one, keeping in mind we will have some life cutoff time

type JobNode struct {
	jobPrefix    string
	maxJobs      uint
	maxIdleSec   uint
	db           *mongo.Database
	instanceId   string
	log          logger.ILogger
	ts           timestamper.ITimeStamper
	jobContainer string // If empty string we run jobs in this process, mainly for testing. Otherwise run jobs in Docker
	jobBucket    string
	fs           fileaccess.FileAccess

	jobStartedCount uint
	//lastJobStartUnixSec uint
}

func CreateJobNode(
	jobPrefix string,
	jobContainer string,
	jobBucket string,
	maxJobs uint,
	maxIdleSec uint,
	instanceId string,
	fs fileaccess.FileAccess,
	db *mongo.Database,
	log logger.ILogger,
	ts timestamper.ITimeStamper) *JobNode {
	return &JobNode{jobPrefix, maxJobs, maxIdleSec, db, instanceId, log, ts, jobContainer, jobBucket, fs, 0}
}

func (jn *JobNode) ListenToJobQueue() bool {
	return job.ListenToJobQueue([]string{"insert"}, jn.db, jn.log, jn.onNewJobQueueItemRunOnce)
}

// func (jn *JobNode) GetLastJobStartUnixSec() uint {
// 	return jn.lastJobStartUnixSec
// }

func (jn *JobNode) CheckStartupJobs() {
	jn.log.Infof("Instance %v checking for available jobs on startup...", jn.instanceId)
	defer jn.log.Infof("Instance: %v startup check complete", jn.instanceId)

	jobCapacity, err := jn.getJobCapacity()
	if err != nil {
		jn.log.Errorf("Instance %v failed to check job capacity on startup: %v", jn.instanceId, err)
		return
	}

	jobGroups, err := job.ReadJobQueue(jn.db)

	if err != nil {
		jn.log.Errorf("Instance %v failed to query jobs on node startup: %v", jn.instanceId, err)
		return
	}

	for _, jobs := range jobGroups {
		for _, jobItem := range jobs {
			if jobItem.State == protos.JobQueueItem_UNKNOWN {
				if jobCapacity == 0 {
					return
				}

				jobCapacity = jobCapacity - 1

				jn.log.Infof("Instance %v startup running job %v (from job group %v)", jn.instanceId, jobItem.JobId, jobItem.JobGroupId)
				jn.startJob(jobItem)
			}
		}
	}
}

func (jn *JobNode) onNewJobQueueItemRunOnce(jobItem *protos.JobQueueItem) {
	if jobItem.State != protos.JobQueueItem_UNKNOWN {
		// we aren't interested, this has already started running somewhere else
		return
	}

	// In case there are multiple APIs running, we here have to decide who is going to do the check
	// so we only check jobs once (avoiding duplicate starts)
	err := singleinstance.HandleOnce(jobItem.JobId, jn.instanceId, func(sourceId string) {
		// Read all items and work out what
		jn.log.Infof("HandleOnce id %v, instance %v...", sourceId, jn.instanceId)
		jn.onNewJobQueueItem(jobItem)
	}, jn.db, jn.ts, jn.log)

	if err != nil {
		jn.log.Errorf("Failed to HandleOnce id %v, instance %v. Error: %v", jobItem.JobId, jn.instanceId, err)
	}
}

// Detect new not-yet-run jobs and claim them, run them locally if there is spare capacity in terms of docker
// containers to CPU core ratio
func (jn *JobNode) onNewJobQueueItem(jobItem *protos.JobQueueItem) {
	// Start new jobs if we have capacity
	jobCapacity, err := jn.getJobCapacity()
	if err != nil {
		jn.log.Errorf("%v", err)
		return
	}

	if jobCapacity <= 0 {
		jn.log.Infof("Instance %v has no capacity to run job %v, ignored", jn.instanceId, jobItem.JobId)
		return
	}

	// Remember when we last started a job. This is so we can check if we've been sitting idle waiting for too long
	//jn.lastJobStartUnixSec = uint(jn.ts.GetTimeNowSec())

	jn.startJob(jobItem)
}

func (jn *JobNode) getJobCapacity() (uint, error) {
	if len(jn.jobContainer) <= 0 {
		// In test mode pretend we always have capacity
		return 1, nil
	}

	instances, err := jn.GetActiveJobCount()
	if err != nil {
		return 0, err
	}

	// Our max node count should be set to how many threads we can run. We want to allow
	// running that many containers
	capacity := int(jn.maxJobs) - int(instances)
	if capacity < 0 {
		capacity = 0
	}

	return uint(capacity), nil
}

func (jn *JobNode) GetActiveJobCount() (uint, error) {
	// Get docker container count
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%v.*", jn.jobPrefix))

	out, err := cmd.CombinedOutput()
	outStr := string(out)
	if err != nil {
		if len(outStr) > 0 {
			outStr = "\n" + outStr
		}
		return 0, fmt.Errorf("Failed to query docker containers: %v%v", err, outStr)
	}

	// Each row of text returned is a docker container ID
	instanceIds := strings.Split(outStr, "\n")
	instances := len(instanceIds)
	// Ignore the trailing empty one if there is one!
	if instances > 0 && len(instanceIds[len(instanceIds)-1]) <= 0 {
		instances = instances - 1
	}

	return uint(instances), nil
}

func (jn *JobNode) startJob(jobItem *protos.JobQueueItem) error {
	// Set queue item to running so it doesn't get picked up again
	err := job.UpdateJobQueueItem(jobItem.JobId, protos.JobQueueItem_RUNNING, fmt.Sprintf("Running on instance: %v", jn.instanceId), jobItem.JobGroupId, jn.db, jn.ts)

	if err != nil {
		jn.log.Errorf("%v", err)
		return err
	}

	// Start counting up!
	jn.jobStartedCount = jn.jobStartedCount + 1

	// Set up the path to read the job from
	jobPath := filepaths.GetJobDataPath(jobItem.AssociatedScanId, jobItem.JobGroupId, "")

	if len(jn.jobContainer) <= 0 {
		// Mainly for tests, so we avoid docker and can run/debug all our code in one process
		err = jobrunner.RunJob(jn.jobBucket, jobPath, uint(jobItem.NodeIndex), jn.fs)
		if err != nil {
			jn.log.Errorf("Failed to start job %v (node %v): %v", jobItem.JobGroupId, jobItem.NodeIndex, err)
		}
	} else {
		// Run it in docker using our job runner container
		cmd := exec.Command("docker", "run",
			"-n", fmt.Sprintf("%v-%v-%v", jn.jobPrefix, jn.jobStartedCount, utils.RandStringBytesMaskImpr(6)),
			"-e", jobrunner.EnvBucketName, jn.jobBucket,
			"-e", jobrunner.EnvPathName, jobPath,
			"-e", jobrunner.EnvNodeIndexName, strconv.Itoa(int(jobItem.NodeIndex)),
			jn.jobContainer)

		out, err := cmd.CombinedOutput()
		outStr := string(out)
		if err != nil {
			if len(outStr) > 0 {
				outStr = "\n" + outStr
			}
			return fmt.Errorf("Failed to query docker containers: %v%v", err, outStr)
		}
	}

	// Once the job is finished, mark it so
	msg := ""
	state := protos.JobQueueItem_COMPLETE

	if err != nil {
		state = protos.JobQueueItem_FAILED
		msg = fmt.Sprintf("Failed on instance %v: %v", jn.instanceId, err)
	}

	err = job.UpdateJobQueueItem(jobItem.JobId, state, msg, jobItem.JobGroupId, jn.db, jn.ts)

	if err != nil {
		jn.log.Errorf("%v", err)
		return err
	}

	return nil
}
