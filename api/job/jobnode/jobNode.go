package jobnode

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"sync"

	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/api/job/jobrunner"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
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
	jobRunnerNamePrefix string
	db                  *mongo.Database
	instanceId          string
	log                 logger.ILogger
	ts                  timestamper.ITimeStamper
	jobContainer        string // If empty string we run jobs in this process, mainly for testing. Otherwise run jobs in Docker
	jobBucket           string
	fs                  fileaccess.FileAccess

	jobStartedCount uint
	//lastJobStartUnixSec uint
}

func CreateJobNode(
	jobRunnerNamePrefix string,
	jobContainer string,
	jobBucket string,
	instanceId string,
	fs fileaccess.FileAccess,
	db *mongo.Database,
	log logger.ILogger,
	ts timestamper.ITimeStamper) *JobNode {
	return &JobNode{jobRunnerNamePrefix, db, instanceId, log, ts, jobContainer, jobBucket, fs, 0}
}

func (jn *JobNode) StartJobs(jobIds []string) {
	// Put them in a map for deduplication and removal purposes
	jobIdMap := map[string]bool{}
	for _, id := range jobIds {
		jobIdMap[id] = true
	}

	// Read jobs from job queue
	jobGroups, err := job.ReadJobQueue(jn.db)
	if err != nil {
		jn.log.Errorf("Instance %v failed to query jobs on node startup: %v", jn.instanceId, err)
		return
	}

	// Find each job we were told to run
	var wg sync.WaitGroup
	for _, jobs := range jobGroups {
		for _, jobItem := range jobs {
			if _, ok := jobIdMap[jobItem.JobId]; ok == true {
				// Mark this job spoken for
				jobIdMap[jobItem.JobId] = false

				// Run this job
				// NOTE: if we're in "local" mode for testing, we run the job synchronously so we get
				// consistant output
				wg.Add(1)

				if len(jn.jobContainer) <= 0 {
					jn.startJob(jobItem, &wg)
				} else {
					go jn.startJob(jobItem, &wg)
				}
			}
		}
	}

	// If we have any jobs that weren't found, report
	for id, waiting := range jobIdMap {
		if waiting {
			jn.log.Errorf("Instance %v failed to find job %v in queue, skipped", jn.instanceId, id)
		}
	}

	// If we need to, wait for it
	if len(jn.jobContainer) > 0 {
		wg.Wait()
	}
}

func (jn *JobNode) GetActiveJobCount() (uint, error) {
	// Get docker container count
	cmd := exec.Command("docker", "ps", "-q", "-f", fmt.Sprintf("name=%v.*", jn.jobRunnerNamePrefix))

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

func (jn *JobNode) startJob(jobItem *protos.JobQueueItem, wg *sync.WaitGroup) {
	defer wg.Done()

	jn.log.Infof("Instance %v starting job \"%v\"...", jn.instanceId, jobItem.JobId)

	// Set queue item to running so it doesn't get picked up again
	err := job.UpdateJobQueueItem(
		jobItem.JobId,
		protos.JobQueueItem_RUNNING,
		fmt.Sprintf("Running on instance: %v", jn.instanceId),
		jobItem.JobGroupId,
		jn.instanceId,
		jn.db, jn.ts)

	if err != nil {
		jn.log.Errorf("%v", err)
		return
	}

	// Start counting up!
	jn.jobStartedCount = jn.jobStartedCount + 1

	// Set up the path to read the job from
	jobPath := filepaths.GetJobDataPath(jobItem.AssociatedScanId, jobItem.JobGroupId, "")

	if len(jn.jobContainer) <= 0 {
		fmt.Println("WARNING: Running job locally, recommended for use for tests only!")

		// Mainly for tests, so we avoid docker and can run/debug all our code in one process
		err = jobrunner.RunJob(jn.jobBucket, jobPath, uint(jobItem.NodeIndex), jn.fs)
		if err != nil {
			jn.log.Errorf("Failed to start job %v (node %v): %v", jobItem.JobGroupId, jobItem.NodeIndex, err)
		}
	} else {
		// Run it in docker using our job runner container
		cmd := exec.Command("docker", "run",
			"--name", fmt.Sprintf("%v-%v-%v", jn.jobRunnerNamePrefix, jn.jobStartedCount, utils.RandStringBytesMaskImpr(6)),
			"-e", "AWS_ACCESS_KEY_ID="+os.Getenv("AWS_ACCESS_KEY_ID"),
			"-e", "AWS_SECRET_ACCESS_KEY="+os.Getenv("AWS_SECRET_ACCESS_KEY"),
			"-e", "AWS_REGION="+os.Getenv("AWS_REGION"),
			"-e", "AWS_DEFAULT_REGION="+os.Getenv("AWS_DEFAULT_REGION"),
			"-e", fmt.Sprintf("%v=%v", jobrunner.EnvBucketName, jn.jobBucket),
			"-e", fmt.Sprintf("%v=%v", jobrunner.EnvPathName, jobPath),
			"-e", fmt.Sprintf("%v=%v", jobrunner.EnvNodeIndexName, strconv.Itoa(int(jobItem.NodeIndex))),
			jn.jobContainer)

		out, err := cmd.CombinedOutput()
		outStr := string(out)
		if err != nil {
			if len(outStr) > 0 {
				outStr = "\n" + outStr
			}

			logEnd := ""
			if len(outStr) > 300 {
				logEnd = "..." + outStr[len(outStr)-300:]
			} else {
				logEnd = outStr
			}

			err2 := job.UpdateJobQueueItem(
				jobItem.JobId,
				protos.JobQueueItem_FAILED,
				fmt.Sprintf("Job Failed: %v.\nEnd of log: %v", err, logEnd),
				jobItem.JobGroupId,
				jn.instanceId,
				jn.db, jn.ts)
			if err2 != nil {
				jn.log.Errorf("Failed to update job queue item %v to failed status: %v", jobItem.JobId, err2)
			}
			jn.log.Errorf("Job run for %v failed: %v%v", jobItem.JobId, err, outStr)
		}

		jn.log.Infof("Job %v run complete, output:\n-----------------\n%v\n-----------------\n", jobItem.JobId, outStr)
	}

	// Once the job is finished, mark it so
	msg := ""
	state := protos.JobQueueItem_COMPLETE

	if err != nil {
		state = protos.JobQueueItem_FAILED
		msg = fmt.Sprintf("Failed on instance %v: %v", jn.instanceId, err)
	}

	err = job.UpdateJobQueueItem(
		jobItem.JobId,
		state,
		msg,
		jobItem.JobGroupId,
		jn.instanceId,
		jn.db, jn.ts)

	if err != nil {
		jn.log.Errorf("%v", err)
	}
}
