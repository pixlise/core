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
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
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
	configBucket        string
	usersBucket         string
	datasetsBucket      string
	fs                  fileaccess.FileAccess

	jobStartedCount uint
	//lastJobStartUnixSec uint
}

var LuaExpressionCommand = "lua-expression"

func CreateJobNode(
	jobRunnerNamePrefix string,
	jobContainer string,
	jobBucket string,
	configBucket string,
	usersBucket string,
	datasetsBucket string,
	instanceId string,
	fs fileaccess.FileAccess,
	db *mongo.Database,
	log logger.ILogger,
	ts timestamper.ITimeStamper) *JobNode {
	return &JobNode{
		jobRunnerNamePrefix: jobRunnerNamePrefix,
		db:                  db,
		instanceId:          instanceId,
		log:                 log,
		ts:                  ts,
		jobContainer:        jobContainer,
		jobBucket:           jobBucket,
		configBucket:        configBucket,
		usersBucket:         usersBucket,
		datasetsBucket:      datasetsBucket,
		fs:                  fs,
		jobStartedCount:     0,
	}
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

	var outStr, msg string
	local := false
	var jobFunc jobrunner.CommandRunner

	if jobItem.JobType == protos.JobType_JT_RUN_EXPRESSION {
		fmt.Println("Running lua expression job locally!")
		local = true
		jobFunc = jn.runLocalLuaExpression
	} else if len(jn.jobContainer) <= 0 {
		fmt.Println("WARNING: Running job locally, recommended for use for tests only!")
		local = true
	}

	if local {
		// Mainly for tests, so we avoid docker and can run/debug all our code in one process
		err = jobrunner.RunJob(jn.jobBucket, jobPath, uint(jobItem.NodeIndex), jn.fs, jobFunc)
		if err != nil {
			jn.log.Errorf("Failed to start job %v (node %v): %v", jobItem.JobGroupId, jobItem.NodeIndex, err)
		}

		outStr = "No output saved from local job run"
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
		outStr = string(out)
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

			msg = fmt.Sprintf("Job %v failed on instance %v: %v.\nEnd of log: %v", jobItem.JobGroupId, jn.instanceId, err, logEnd)
		}
	}

	jn.log.Infof("Job %v run complete: \"%v\"\nOutput:\n-----------------\n%v\n-----------------", jobItem.JobId, msg, outStr)

	// Once the job is finished, mark it so
	state := protos.JobQueueItem_COMPLETE

	if err != nil {
		state = protos.JobQueueItem_FAILED
		if len(msg) <= 0 {
			msg = fmt.Sprintf("Failed on instance %v: %v", jn.instanceId, err)
		}
	}

	err = job.UpdateJobQueueItem(
		jobItem.JobId,
		state,
		msg,
		jobItem.JobGroupId,
		jn.instanceId,
		jn.db, jn.ts)

	if err != nil {
		jn.log.Errorf("Failed to update job queue item %v to failed status: %v", jobItem.JobId, err)
	}
}

var ExpressionJobOutputFileName = "output.csv"

func (jn *JobNode) runLocalLuaExpression(command string, args []string) (string, error) {
	if command != LuaExpressionCommand {
		return "", fmt.Errorf("Expected job command: %v, got %v", LuaExpressionCommand, command)
	}

	// Read args, expect key=value pairs
	argLookup, err := utils.ReadKeyValueList([]string{"scanId", "quantId", "expressionId", "memoKey"}, args)
	if err != nil {
		return "", fmt.Errorf("Lua expression failed to run - %v", err)
	}

	// Run the expression locally because we have all the DB and whatever access required
	outputMap, _, _, err := expressionrunner.RunExpression(argLookup["expressionId"], argLookup["scanId"], argLookup["quantId"],
		jn.log, jn.db, jn.ts, jn.fs,
		jn.configBucket, jn.usersBucket, jn.datasetsBucket, true, false)

	if err != nil {
		return "", err
	}

	if len(outputMap.Values) <= 0 {
		return "", fmt.Errorf("Lua expression %v returned empty result", argLookup["expressionId"])
	}

	// Save the expression result map as a CSV to local path where it'll get picked up
	// and written out by the job runner that called us
	var sb strings.Builder
	_, err = sb.WriteString("\"PMC\",\"value\"\n")
	if err != nil {
		return "", fmt.Errorf("Failed to write result CSV header for lua expression %v: %v", argLookup["expressionId"], err)
	}
	for c, v := range outputMap.Values {
		val := "null"
		if !v.IsUndefined {
			val = fmt.Sprintf("%v", v.Value)
		}

		_, err = sb.WriteString(fmt.Sprintf("%v,%v\n", v.PMC, val))
		if err != nil {
			return "", fmt.Errorf("Failed to write line %v for lua expression %v: %v", c, argLookup["expressionId"], err)
		}
	}

	// Stop here, CSV should be picked up by job runner & put in S3, from there PIXLISE will run the completion
	// routine which should read that and memoise it granting access for the client
	err = os.WriteFile(ExpressionJobOutputFileName, []byte(sb.String()), 0777)

	return "", err
}
