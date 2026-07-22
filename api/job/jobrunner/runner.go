package jobrunner

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

var EnvBucketName = "JOB_BUCKET"
var EnvPathName = "JOB_PATH"
var EnvNodeIndexName = "NODE_INDEX"

// Downloads files required for job to run and sets up libraries. Requires JOB_CONFIG environment variable
// to be set to a JobConfig structure
// Parameters:
// - jobBucket: The S3 bucket to read job config from
// - jobPath:   Path to the job in S3
// - nodeIndex: Which node number are we on? Used to generate config for that node
// - runFunc:   nil or a function to call when running the actual job
func RunJob(jobBucket string, jobPath string, nodeIndex uint, remoteFS fileaccess.FileAccess, runFunc CommandRunner) error {
	if runFunc == nil {
		runFunc = runCommand
	}

	if len(jobBucket) <= 0 {
		return fmt.Errorf("RunJob: bucket not set")
	}
	if len(jobPath) <= 0 {
		return fmt.Errorf("RunJob: path not set")
	}
	if nodeIndex > 100000 {
		return fmt.Errorf("RunJob: nodeIndex too high")
	}

	// Init logger - this used to be local=stdout, cloud env=cloudwatch, but we now write all logs to stdout
	jobLog := &logger.StdOutLogger{}
	jobLog.Infof("Running job from s3://%v/%v for node %v", jobBucket, jobPath, nodeIndex)

	// Read config from S3 (or our local simulator!)
	jobParamPath := path.Join(jobPath, quantification.JobParamsFileName)
	var jobGroupCfg jobconfig.JobGroupConfig
	err := remoteFS.ReadJSON(jobBucket, jobParamPath, &jobGroupCfg, false)
	if err != nil {
		return fmt.Errorf("Failed to read job config s3://%v/%v: %v", jobBucket, jobParamPath, err)
	}

	cfg := jobGroupCfg.NodeConfig.FlattenJobConfig(nodeIndex)

	jobLog.Debugf("Job config struct: %#v", cfg)

	// Validate
	if len(cfg.Command) <= 0 {
		return fmt.Errorf("No command specified")
	}

	// Download required files
	jobLog.Infof("Downloading files...")
	for _, reqFile := range cfg.RequiredFiles {
		err := downloadFile(jobLog, remoteFS, reqFile.RemoteBucket, reqFile.RemotePath, reqFile.LocalPath)
		if err != nil {
			return err
		}
	}

	pythonPath := ""
	if strings.Contains(cfg.Command, "python") {
		jobLog.Infof("Using python virtual env...")
		// It worked, so set our python path!
		pythonPath, err = os.Getwd()
		if err == nil {
			pythonPath = filepath.Join(pythonPath, "bin")
		}
	}

	jobLog.Infof("Checking for required libraries...")
	commandToRun := cfg.Command

	if strings.Contains(cfg.Command, "python") {
		jobLog.Infof("Installing required python libraries...")
		err = installPythonLibs(pythonPath)

		// Modify the command!
		commandToRun = filepath.Join(pythonPath, cfg.Command)
	} /*else if strings.Contains(cfg.Command, "lua") {
		jobLog.Infof("Installing required lua libraries...")
		err = installLuaLibs()
	}*/

	if err != nil {
		return err
	}

	jobLog.Infof("Running job...")

	// Run the actual job
	jobLog.Debugf("exec.Command starting \"%v\", args: [%v]", commandToRun, strings.Join(cfg.Args, ","))

	// We support the concept of a "no-op" command only for testing - because tests can run on different platforms
	// we want to be able to write tests that don't actually run a command, this area is very OS specific...
	// This way we can just test the file download and upload capabilities separately
	startUnixSec := time.Now().Unix()

	// Execute the runner function - it may start a new process, it may run locally, whatever...
	cmdStdOut, err := runFunc(commandToRun, cfg.Args)
	if err != nil {
		outErr := fmt.Errorf("Job %v failed: %v", cfg.JobId, err)
		jobLog.Errorf("%v", outErr)
		if len(cmdStdOut) > 0 {
			jobLog.Infof("%v", cmdStdOut)
		}

		return outErr
	}

	runTimeUnixSec := time.Now().Unix() - startUnixSec
	if runTimeUnixSec < 10 {
		// For tests, we want to output something consistant for quick runs
		jobLog.Infof("Job %v runtime was < 10 sec", cfg.JobId)
	} else {
		jobLog.Infof("Job %v runtime was %v sec", cfg.JobId, runTimeUnixSec)
	}

	// Upload output files required, error if not found
	failedOutputFiles := []string{}
	for _, outputFile := range cfg.OutputFiles {
		if outputFile.LocalPath == "stdout" {
			// "Special" file, we just output the stdout of running the command
			err = remoteFS.WriteObject(outputFile.RemoteBucket, outputFile.RemotePath, []byte(cmdStdOut))
			if err != nil {
				jobLog.Errorf("Failed to upload stdout log to s3://%v/%v: %v", outputFile.RemoteBucket, outputFile.RemotePath, err)
			} else {
				jobLog.Debugf("Uploaded stdout log to: s3://%v/%v", outputFile.RemoteBucket, outputFile.RemotePath)
			}
		} else {
			_, err := os.Stat(outputFile.LocalPath)
			if err != nil {
				jobLog.Errorf("Job %v did not generate expected output file: %v", cfg.JobId, outputFile.LocalPath)
				failedOutputFiles = append(failedOutputFiles, outputFile.LocalPath)
			} else {
				err := uploadFile(jobLog, remoteFS, outputFile.LocalPath, outputFile.RemoteBucket, outputFile.RemotePath)
				if err != nil {
					jobLog.Errorf("Job %v failed to upload file: %v. Error: %v", cfg.JobId, outputFile.LocalPath, err)
					failedOutputFiles = append(failedOutputFiles, outputFile.LocalPath)
				}
			}
		}
	}

	if len(failedOutputFiles) > 0 {
		return fmt.Errorf("Job %v failed to generate/upload output files: %v", cfg.JobId, strings.Join(failedOutputFiles, ", "))
	}

	return nil
}
