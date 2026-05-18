package jobrunner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pixlise/core/v4/api/job"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

func logJobPrep(cfgStr string, cfg job.JobConfig, jobLog logger.ILogger) {
	jobLog.Infof("Preparing job: %v\nConfig: %+v", cfg.JobId, cfgStr)
	jobLog.Infof("Job config struct: %#v", cfg)
}

// Downloads files required for job to run and sets up libraries. Requires JOB_CONFIG environment variable
// to be set to a JobConfig structure
// Parameters:
// - localBucketPath - use this to set a local path to simulate bucket access from, useful for unit tests
func RunJob(localBucketPath string) error {
	// Get the config from env var
	cfgStr := os.Getenv(job.JobConfigEnvVar)

	if len(cfgStr) <= 0 {
		return fmt.Errorf("%v env var not set", job.JobConfigEnvVar)
	}

	var cfg job.JobConfig
	err := json.Unmarshal([]byte(cfgStr), &cfg)
	if err != nil {
		return fmt.Errorf("Failed to parse env var %v: %v", job.JobConfigEnvVar, err)
	}

	sess, err := awsutil.GetSession()
	if err != nil {
		return fmt.Errorf("Failed to create AWS session. Error: %v", err)
	}

	// Init logger - this used to be local=stdout, cloud env=cloudwatch, but we now write all logs to stdout
	var jobLog logger.ILogger

	// if len(cfg.LogCloudwatchGroup) > 0 && len(cfg.LogCloudwatchStream) > 0 {
	// 	cwlog, err := logger.InitCloudWatchLogger(
	// 		sess,
	// 		cfg.LogCloudwatchGroup,
	// 		cfg.LogCloudwatchStream,
	// 		logger.LogDebug,
	// 		30,
	// 		5,
	// 	)

	// 	if err == nil {
	// 		jobLog = cwlog
	// 	} else {
	// 		jobLog = logger.StdOutLogger{}
	// 		jobLog.Errorf("Failed to create cloudwatch logger, logging to stdout")
	// 	}
	// } else {
	jobLog = &logger.StdOutLogger{}
	// if testMode {
	// 	jobLog.SetLogLevel(logger.LogInfo)
	// }
	//}

	logJobPrep(cfgStr, cfg, jobLog)

	// Validate
	if len(cfg.Command) <= 0 {
		return fmt.Errorf("No command specified")
	}

	var remoteFS fileaccess.FileAccess
	if len(localBucketPath) <= 0 {
		// Init AWS stuff
		jobLog.Infof("AWS S3 setup...")

		s3svc, err := awsutil.GetS3(sess)
		if err != nil {
			return fmt.Errorf("Failed to create AWS S3 service. Error: %v", err)
		}

		remoteFS = fileaccess.MakeS3Access(s3svc)
	} else {
		remoteFS = fileaccess.MakeFSAccessS3Simulator(localBucketPath)
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
	}
	if strings.Contains(cfg.Command, "lua") {
		jobLog.Infof("Installing required lua libraries...")
		err = installLuaLibs()
	}

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

	cmdStdOut, err := runCommand(commandToRun, cfg.Args)
	if err != nil {
		outErr := fmt.Errorf("Job %v failed: %v", cfg.JobId, err)
		jobLog.Errorf("%v", outErr)
		if len(cmdStdOut) > 0 {
			jobLog.Infof(cmdStdOut)
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

// const configIdxFileName = "config.json"

// var pmcListName string

// func init() {
// 	flag.StringVar(&pmcListName, "pmclistname", "", "Override the PMCListName")
// }

// func getDatasetFileFromPMCList(pmcListPath string) (string, error) {
// 	datasetFileName := ""

// 	file, err := os.Open(pmcListPath)
// 	if err != nil {
// 		return datasetFileName, err
// 	}
// 	defer file.Close()

// 	scanner := bufio.NewScanner(file)

// 	// Read 1 line
// 	if !scanner.Scan() {
// 		return datasetFileName, fmt.Errorf("Failed to read dataset file name from PMC list file: %v", pmcListPath)
// 	}

// 	return scanner.Text(), nil
// }
/*
type configFileIndex struct {
	Description     string `json:"description"`
	Config          string `json:"config-file"`
	OpticEfficiency string `json:"optic-efficiency"`
	Calibration     string `json:"calibration-file"`
	Standards       string `json:"standards-file"`
}

type configPaths struct {
	dataset       string
	pmcList       string
	config        string
	calibration   string
	jobsPath      string
	remoteJobPath string
}

func prepConfigsForPiquant(
	params quantModel.PiquantParams,
	cwlog logger.ILogger,
	remoteFS fileaccess.FileAccess,
) (configPaths, error) {

	var result configPaths
	localTmpPath := os.TempDir()

	localFS := fileaccess.FSAccess{}

	result.remoteJobPath = path.Join(params.JobsPath, params.JobID)

	result.jobsPath = path.Join(localTmpPath, params.JobID)
	localConfigPath := path.Join(result.jobsPath, "config")

	// Delete any temp files from a previous run
	os.RemoveAll(result.jobsPath)

	// Make sure they all exist
	tmppaths := []string{localTmpPath, result.jobsPath, localConfigPath}

	for _, tmppath := range tmppaths {
		err := os.MkdirAll(tmppath, dirperm)
		if err != nil {
			cwlog.Errorf("Failed to create local path \"%v\": %v", tmppath, err)
			return result, err
		}
	}

	// List the config files
	configPaths, err := remoteFS.ListObjects(params.ConfigBucket, params.DetectorConfig)
	if err != nil {
		cwlog.Errorf("Failed to list config files in: \"s3://%v/%v\"", params.ConfigBucket, params.DetectorConfig)
		return result, err
	}

	// Download each config file
	localConfigIdxPath := ""
	for _, path := range configPaths {
		cwlog.Infof("Downloading config file: %v", path)
		savedPath, err := downloadFile(remoteFS, params.ConfigBucket, path, localConfigPath)

		if err != nil {
			cwlog.Errorf("Failed to download config file \"s3://%v/%v\": %v", params.ConfigBucket, path, err)
			return result, err
		}

		// Check if it's the config path, we'll need it later
		fileName := filepath.Base(path)
		if fileName == configIdxFileName {
			localConfigIdxPath = savedPath
		}
	}

	// If config.json was not found, error!
	if len(localConfigIdxPath) <= 0 {
		cwlog.Errorf("No %v found in detector config %v", configIdxFileName, params.DetectorConfig)
		return result, err
	}

	// Download PMC list file
	remotePMCListPath := path.Join(result.remoteJobPath, params.PMCListName)
	cwlog.Infof("Downloading PMC list file: %v", remotePMCListPath)

	result.pmcList, err = downloadFile(remoteFS, params.PiquantJobsBucket, remotePMCListPath, result.jobsPath)

	if err != nil {
		cwlog.Errorf("Failed to download PMC list file \"s3://%v/%v\": %v", params.PiquantJobsBucket, remotePMCListPath, err)
		return result, err
	}

	// Get dataset file name (first line in PMC list)
	datasetFileName, err := getDatasetFileFromPMCList(result.pmcList)
	if err != nil {
		cwlog.Errorf("%v", err)
		return result, err
	}

	// Download dataset file
	remoteDatasetFilePath := path.Join(params.DatasetPath, datasetFileName)
	cwlog.Infof("Downloading dataset file: %v", remoteDatasetFilePath)

	result.dataset, err = downloadFile(remoteFS, params.DatasetsBucket, remoteDatasetFilePath, result.jobsPath)

	if err != nil {
		cwlog.Errorf("Failed to download dataset file \"%v\": %v", remoteDatasetFilePath, err)
		return result, err
	}

	// Set up optics file configuration
	cwlog.Infof("Preparing config files for PIQUANT...")

	// Read the config file
	var configIdx configFileIndex
	err = localFS.ReadJSON(localConfigIdxPath, "", &configIdx, false)
	if err != nil {
		cwlog.Errorf("Failed to read config index \"%v\": %v", localConfigIdxPath, err)
		return result, err
	}

	// Need to add an OPTICFILE entry to config, with the right path for the optic file (so Piquant can access the path directly)
	result.config = path.Join(localConfigPath, configIdx.Config)

	if len(configIdx.OpticEfficiency) > 0 {
		localOpticEfficiencyPath := path.Join(localConfigPath, configIdx.OpticEfficiency)

		cwlog.Debugf("Amending config file \"%v\" with optic file path: \"%v\"", result.config, localOpticEfficiencyPath)

		cfgFile, err := os.OpenFile(result.config, os.O_APPEND|os.O_WRONLY, dirperm)

		if err != nil {
			cwlog.Errorf("Failed to open config index \"%v\" for amending: %v", result.config, err)
			return result, err
		} else {
			defer cfgFile.Close()

			if _, err = cfgFile.WriteString(fmt.Sprintf("\n##OPTICFILE : %v", localOpticEfficiencyPath)); err != nil {
				cwlog.Errorf("Failed to write amended OPTICFILE to config index \"%v\": %v", result.config, err)
				return result, err
			}
		}
	} else {
		cwlog.Debugf("Config file  \"%v\" NOT amended with optic file path, as no file specified...", result.config)
	}

	// Set the resulting paths
	result.calibration = path.Join(localConfigPath, configIdx.Calibration)

	return result, nil
}
*/
/*
func saveOutputs(jobBucket string,
	remoteJobPath string,
	pmcListName string,
	outputPath string,
	logPath string,
	piquantStdOut string,
	cwlog logger.ILogger,
	remoteFS fileaccess.FileAccess) {

	// Check if output file was generated
	_, err := os.Stat(outputPath)
	if err != nil {
		cwlog.Errorf("PIQUANT did not generate an output file: %v", err)
	} else {
		// Save the output in S3
		remoteJobOutputPath := path.Join(remoteJobPath, "output", pmcListName+"_result.csv")

		err = uploadFile(remoteFS, jobBucket, remoteJobOutputPath, outputPath)
		if err != nil {
			cwlog.Errorf("Failed to upload PIQUANT output file %v to %v: %v", outputPath, remoteJobOutputPath, err)
		} else {
			cwlog.Infof("Uploaded %v", remoteJobOutputPath)
		}
	}

	// If output log (from map threads) was generated, upload that too
	_, err = os.Stat(logPath)
	if err != nil {
		cwlog.Errorf("PIQUANT did not generate a job log file: %v", err)
	} else {
		// Save the log in S3
		remoteJobLogPath := path.Join(remoteJobPath, "piquant-logs", pmcListName+"_piquant.log")

		err = uploadFile(remoteFS, jobBucket, remoteJobLogPath, logPath)
		if err != nil {
			cwlog.Errorf("Failed to upload PIQUANT job log file %v to %v: %v", logPath, remoteJobLogPath, err)
		} else {
			cwlog.Infof("Uploaded job log %v", remoteJobLogPath)
		}
	}

	// Save stdout as a log file to S3
	remoteJobStdOutLogPath := path.Join(remoteJobPath, "piquant-logs", pmcListName+"_stdout.log")
	err = remoteFS.WriteObject(jobBucket, remoteJobStdOutLogPath, []byte(piquantStdOut))

	if err != nil {
		cwlog.Errorf("Failed to upload PIQUANT stdout log to %v: %v", remoteJobStdOutLogPath, err)
	} else {
		cwlog.Infof("Uploaded stdout log %v", remoteJobStdOutLogPath)
	}
}
*/
// func loadParams() quantModel.PiquantParams {
// 	// Parameters need to be in an env var
// 	paramStr := os.Getenv("QUANT_PARAMS")

// 	var params quantModel.PiquantParams
// 	err := json.Unmarshal([]byte(paramStr), &params)
// 	if err != nil {
// 		log.Fatalf("Failed to parse QUANT_PARAMS: %v", err)
// 	}

// 	// Allow user to override the PMCListName via another environment variable; this allows a single k8s
// 	//  Job to run many parallel pods where we use JOB_COMPLETION_INDEX (that k8s sets on indexed job types)
// 	if pmcListName != "" {
// 		params.PMCListName = pmcListName
// 	} else {
// 		if nodeIndex, set := os.LookupEnv("JOB_COMPLETION_INDEX"); set {
// 			log.Printf("JOB_COMPLETION_INDEX: %v", nodeIndex)
// 			nodeIndexInt, _ := strconv.Atoi(nodeIndex)
// 			// TODO: Use a shared function from the quantModel package to keep pmcListFile name consistent; e.g.
// 			// params.PMCListName = quantRunner.MakePMCFileName(nodeIndexInt)
// 			params.PMCListName = fmt.Sprintf("node%05d.pmcs", nodeIndexInt+1)
// 		} else {
// 			log.Println("JOB_COMPLETION_INDEX not set")
// 			if params.PMCListName == "" {
// 				log.Fatalln("PMC list name is empty, and no JOB_COMPLETION_INDEX found")
// 			}
// 		}
// 	}

// 	return params
// }

//func main() {
// flag.Parse()
// params := loadParams()
// // Init AWS SDK
// sess, err := awsutil.GetSession()
// if err != nil {
// 	log.Fatalf("Failed to create AWS session: %v", err)
// }

// s3svc, err := awsutil.GetS3(sess)
// if err != nil {
// 	log.Fatalf("Failed to create AWS S3 session: %v", err)
// }

// fs := fileaccess.MakeS3Access(s3svc)

// Create a logger for this job
// nodeName := params.PMCListName
// pmcsSuffix := ".pmcs"
// nodeName = strings.TrimSuffix(nodeName, pmcsSuffix)

// logGroup := fmt.Sprintf("/run-piquant/%v", params.RunTimeEnv)
// logStream := fmt.Sprintf("job-%v-node-%v", params.JobID, nodeName)

// cwlog, err := logger.InitCloudWatchLogger(
// 	sess,
// 	logGroup,
// 	logStream,
// 	logger.LogDebug,
// 	30,
// 	5,
// )

// if err != nil {
// 	log.Printf("Failed to create logger, will just log to stdout. Error was: %v", err)
// 	cwlog = &logger.StdOutLogger{}
// }

// cwlog.Infof("RunPiquant called with params: %v", os.Getenv("QUANT_PARAMS"))
// cwlog.Infof("RunPiquant params struct: %#v", params)

// paths, err := prepConfigsForPiquant(params, cwlog, fs)
// if err != nil {
// 	// Assume error logging happend already
// 	os.Exit(1)
// }

// outPath := path.Join(paths.jobsPath, "output.csv")

// NOTE: we can't just provide params.Parameters here, as it itself is a space-separated command line arg string
// so we need to split it into its components, and pass them as individual args
// elementListStr := strings.Join(params.Elements, ",")
// paramList := strings.Split(params.Parameters, " ")

// // Form a single list of arguments
// args := []string{
// 	params.Command,
// 	paths.config,
// 	paths.calibration,
// 	paths.pmcList,
// 	elementListStr,
// 	outPath,
// }
// args = append(args, paramList...)
// cwlog.Debugf("exec.Command args: %v", args)

// cmd := exec.Command("./Piquant", args...)

// startUnixSec := time.Now().Unix()

// // Run it, save output text
// out, err := cmd.CombinedOutput()
// if err != nil {
// 	cwlog.Errorf("Running piquant %v in docker failed: %v\n", params.PMCListName, err)
// 	cwlog.Infof(string(out))
// 	os.Exit(1)
// }

// runTimeUnixSec := time.Now().Unix() - startUnixSec
// cwlog.Infof("Running piquant took %v sec\n", runTimeUnixSec)

// Work out what the log path may be
// 	localLogPath := path.Join(filepath.Dir(outPath), "output.csv_log.txt")

// 	saveOutputs(params.PiquantJobsBucket, paths.remoteJobPath, params.PMCListName, outPath, localLogPath, string(out), cwlog, fs)
// }
