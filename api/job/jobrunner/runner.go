package jobrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

var EnvBucketName = "JOB_BUCKET"
var EnvPathName = "JOB_PATH"
var EnvNodeIndexName = "NODE_INDEX"

var ArgRepoUrlName = "repoUrl"
var ArgRepoUserName = "repoUser"
var ArgRepoSecretName = "repoSecret"
var ArgBranchName = "branch"
var ArgExecFileName = "scriptName"
var ArgClientAuthConfig = "clientAuthSecret"

// Downloads files required for job to run and sets up libraries. Requires JOB_CONFIG environment variable
// to be set to a JobConfig structure
// Parameters:
// - jobBucket: The S3 bucket to read job config from
// - jobPath:   Path to the job in S3
// - nodeIndex: Which node number are we on? Used to generate config for that node
// - repoDetails: If not nil, we expect this to not have any blank fields - it describes how to download a repository so we can run code from it
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
	var jobGroupCfg = &protos.JobGroupConfig{
		NodeConfig: &protos.JobConfig{},
	}
	err := remoteFS.ReadJSON(jobBucket, jobParamPath, &jobGroupCfg, false)
	if err != nil {
		return fmt.Errorf("Failed to read job config s3://%v/%v: %v", jobBucket, jobParamPath, err)
	}

	cfg := jobconfig.FlattenJobConfig(jobGroupCfg.NodeConfig, nodeIndex)

	if jobLog.GetLogLevel() == logger.LogDebug {
		cfgJson, _ := json.MarshalIndent(cfg, "", utils.PrettyPrintIndentForJSON)
		jobLog.Debugf("Job config struct: %v", string(cfgJson))
	}

	// Validate
	if len(cfg.Command) <= 0 {
		return fmt.Errorf("No command specified")
	}

	pythonPath := ""
	if strings.Contains(cfg.Command, "python") {
		jobLog.Infof("Using python virtual env...")
		// It worked, so set our python path!
		pythonPath, err = os.Getwd()
		if err == nil {
			pythonPath = filepath.Join(pythonPath, "bin")

			if _, err := os.Stat(pythonPath); err != nil {
				// Bin directory doesn't exist, maybe python isn't there. We may not be running on a job node, maybe it's local to the API
				// itself, or it's a test, or whatever... Try create a virtual env and save its path here
				out, err := runCommand("python3", []string{"-m", "venv", "./venv"})
				if err != nil {
					p := os.Getenv("PATH")
					return fmt.Errorf("Failed to create python venv: %v. PATH: %v", err, p)
				}

				if len(out) > 0 {
					jobLog.Infof("Python venv creation output: %v", out)
				}

				// At this point, assume we have a venv!
				pythonPath, _ = os.Getwd()
				pythonPath = filepath.Join(pythonPath, "venv", "bin")
			}
		}
	}

	// If arguments seem to want a repository downloaded, do that
	jobLog.Infof("Scanning arguments...")
	var repoLocalPath, scriptPath, origWD string
	for _, arg := range cfg.Args {
		if !strings.HasPrefix(arg, ArgRepoUrlName) {
			continue
		}

		jobLog.Infof("Detected argument \"%v\", downloading repository...", ArgRepoUrlName)

		// OK there's a repo URL defined, so ensure all fields we're interested in exist
		argLookup, err := utils.ReadKeyValueList([]string{ArgRepoUrlName, ArgRepoUserName, ArgRepoSecretName, ArgBranchName, ArgExecFileName}, cfg.Args)
		if err != nil {
			return fmt.Errorf("Not all fields defined for repository download: %v", err)
		}

		// Download repo contents
		// To do this, we just pull down the zip archive
		// Example: https://github.com/pixlise/pixlise-ui/archive/refs/heads/main.zip
		//          https://github.com/pixlise/pixlise-ui/archive/refs/heads/feature/back-end-expressions.zip
		// We already know the URL, get the branch name
		archiveLink := strings.TrimRight(argLookup[ArgRepoUrlName], " /")
		archiveLink += "/" + path.Join("archive", "refs", "heads", argLookup[ArgBranchName]+".zip")

		repoLocalPath, scriptPath, origWD, err = downloadRepository(archiveLink, argLookup[ArgRepoUserName], argLookup[ArgRepoSecretName], jobGroupCfg.JobGroupId, argLookup[ArgExecFileName], jobLog)

		if err != nil {
			return fmt.Errorf("Error while downloading repository \"%v\": %v", archiveLink, err)
		}

		// Set the script path argument to be relative to where we are now
		for c, arg := range cfg.Args {
			if strings.HasPrefix(arg, ArgExecFileName) {
				cfg.Args[c] = fmt.Sprintf("%v=%v", ArgExecFileName, scriptPath)
				break
			}
		}

		// Set the client library authentication environment variable if it's defined
		jobLog.Infof("Checking for argument %v...", ArgClientAuthConfig)

		argLookup, err = utils.ReadKeyValueList([]string{ArgClientAuthConfig}, cfg.Args)
		if err != nil {
			jobLog.Errorf("Argument %v not provided - client auth config not set", ArgClientAuthConfig)
		} else {
			auth := argLookup[ArgClientAuthConfig]
			err = os.Setenv(client.ConfigEnvVar, auth)
			if err != nil {
				return fmt.Errorf("Failed to set environment variable \"%v\" to something %v characters long: %v", client.ConfigEnvVar, len(auth), err)
			} else {
				jobLog.Infof("Set environment variable \"%v\", %v characters long", client.ConfigEnvVar, len(auth))
			}
		}
		break
	}

	// Download required files
	jobLog.Infof("Downloading files...")
	for _, reqFile := range cfg.RequiredFiles {
		err = downloadFile(jobLog, remoteFS, reqFile.RemoteBucket, reqFile.RemotePath, reqFile.LocalPath)
		if err != nil {
			return err
		}
	}

	defer cleanup(origWD, repoLocalPath, cfg, jobLog)

	jobLog.Infof("Checking for required libraries...")
	commandToRun := cfg.Command

	if strings.Contains(cfg.Command, "python") {
		//cfg.Args = append(cfg.Args, "jobId="+jobGroupCfg.JobGroupId)
		jobLog.Infof("Installing required python libraries...")
		err = installPythonLibs(pythonPath, jobLog)
		commandToRun = filepath.Join(pythonPath, "python3")
	}

	if err != nil {
		return err
	}

	jobLog.Infof("Running job...")

	// Run the actual job
	// Don't display secret stuff!
	dispArgs := hideSecretsInArgs(cfg.Args)
	jobLog.Debugf("exec.Command starting \"%v\", args: [%v]", commandToRun, strings.Join(dispArgs, ","))

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
			jobLog.Infof("StdOut:\n%v", cmdStdOut)
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

func hideSecretsInArgs(args []string) []string {
	dispArgs := []string{}
	secretTag := "secret="
	for _, arg := range args {
		idx := strings.Index(strings.ToLower(arg), secretTag)
		if idx < 0 {
			dispArgs = append(dispArgs, arg)
		} else if idx+len(secretTag) < len(arg) {
			// It's *secret=<secret>
			// Print out only the last few chars of the secret!
			dispArgs = append(dispArgs, arg[0:idx+len(secretTag)]+"***"+arg[len(arg)-4:])
		}
	}
	return dispArgs
}

func cleanup(origWD string, repoPath string, cfg *protos.JobConfig, jobLog logger.ILogger) {
	// Delete any downloaded files

	// Delete the repo path if there is one
	if len(repoPath) > 0 {
		if err := os.RemoveAll(repoPath); err != nil {
			jobLog.Errorf("Error cleaning \"%v\" after job: %v", repoPath, err)
		}
	}

	// If we've got to CD back into an original dir, do that
	if len(origWD) > 0 {
		if err := os.Chdir(origWD); err != nil {
			jobLog.Errorf("Error changing back to wd after job: %v", err)
		}
	}
}
