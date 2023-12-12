// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package quantification

import (
	"encoding/json"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/job"
	"github.com/pixlise/core/v3/api/piquant"
	"github.com/pixlise/core/v3/api/quantification/quantRunner"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
)

// JobParamsFileName - File name of job params file
const JobParamsFileName = "params.json"

// CreateJob - creates a new quantification job
func CreateJob(createParams *protos.QuantCreateParams, requestorUserId string, hctx wsHelpers.HandlerContext, wg *sync.WaitGroup, sendUpdate func(*protos.JobStatus)) (*protos.JobStatus, error) {
	svcs := hctx.Svcs

	// Get configured PIQUANT docker container
	piquantVersion, err := piquant.GetPiquantVersion(svcs)

	if err != nil || len(piquantVersion.Version) <= 0 {
		return nil, fmt.Errorf("Failed to get PIQUANT version configuration. Error: %v", err)
	}

	scanFilePath := filepaths.GetScanFilePath(createParams.ScanId, filepaths.DatasetFileName)

	jobIdPrefix := "quant"

	// NOTE: if we're NOT running a map job, we make weird job IDs that help us identify this as a piquant that doesn't need to be
	// treated as a long-running job
	if createParams.Command != "map" {
		// Make the name and ID the same, and start with something that stands out
		jobIdPrefix = "cmd-" + createParams.Command
	}

	jobStatus, err := job.AddJob(jobIdPrefix, uint32(svcs.Config.ImportJobMaxTimeSec), svcs.MongoDB, svcs.IDGen, svcs.TimeStamper, svcs.Log, sendUpdate)
	jobId := ""
	if jobStatus != nil {
		jobId = jobStatus.JobId
	}

	if err != nil || len(jobId) < 0 {
		svcs.Log.Errorf("Failed to add job watcher for quant Job ID: %v", jobId)
	}

	// If not a map command, use the name as the job id just to have a non-empty name and be trackable
	if createParams.Command != "map" {
		createParams.Name = jobId
	}

	createMsg := fmt.Sprintf("quantCreate: %v, %v pmcs, elems=%v, cfg=%v, params=%v. Job ID: %v", scanFilePath, len(createParams.Pmcs), createParams.Elements, createParams.DetectorConfig, createParams.Parameters, jobId)
	svcs.Log.Infof(createMsg)

	// Set up starting parameters
	params := &protos.QuantStartingParameters{
		UserParams:        createParams,
		PmcCount:          uint32(len(createParams.Pmcs)),
		ScanFilePath:      scanFilePath,
		DataBucket:        svcs.Config.DatasetsBucket,
		PiquantJobsBucket: svcs.Config.PiquantJobsBucket,
		CoresPerNode:      uint32(svcs.Config.CoresPerNode),
		StartUnixTimeSec:  uint32(svcs.TimeStamper.GetTimeNowSec()),
		RequestorUserId:   requestorUserId,
		PIQUANTVersion:    piquantVersion.Version,
		//Comments:          createParams.Comments,
	}

	// Save params to file in S3 (so nodes can read it)
	paramsPath := filepaths.GetJobDataPath(createParams.ScanId, jobId, JobParamsFileName)
	err = svcs.FS.WriteJSON(svcs.Config.PiquantJobsBucket, paramsPath, params)
	if err != nil {
		return nil, err
	}

	wg.Add(1)

	// Trigger task to start in a go routine, so we don't block!
	go triggerPiquantNodes(jobId, params, hctx, wg)

	return jobStatus, nil
}

// This should be triggered as a go routine from quant creation endpoint so we can return a job id there quickly and do the processing offline
func triggerPiquantNodes(jobId string, quantStartSettings *protos.QuantStartingParameters, hctx wsHelpers.HandlerContext, wg *sync.WaitGroup) {
	defer wg.Done()

	svcs := hctx.Svcs
	userParams := quantStartSettings.UserParams

	// TODO: figure out log id!
	logId := jobId

	jobRoot := filepaths.GetJobDataPath(userParams.ScanId, "", "")
	jobDataPath := filepaths.GetJobDataPath(userParams.ScanId, jobId, "")

	// Get quant runner interface
	runner, err := quantRunner.GetQuantRunner(svcs.Config.QuantExecutor)
	if err != nil {
		job.CompleteJob(jobId, false, fmt.Sprintf("Failed to start quant runner: %v", err), "", []string{}, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
		return
	}

	job.UpdateJob(jobId, protos.JobStatus_PREPARING_NODES, fmt.Sprintf("Cores/Node: %v", quantStartSettings.CoresPerNode), logId, svcs.MongoDB, svcs.TimeStamper, svcs.Log)

	datasetFileName := path.Base(quantStartSettings.ScanFilePath)
	datasetPathOnly := path.Dir(quantStartSettings.ScanFilePath)

	// Gather required params (these are static, same data passed to each node)
	piquantParams := quantRunner.PiquantParams{
		RunTimeEnv:  svcs.Config.EnvironmentName,
		JobID:       jobId,
		JobsPath:    jobRoot,
		DatasetPath: datasetPathOnly,
		// NOTE: not using path.Join because we want this as / deliberately, this is being
		//       saved in a config file that runs in docker/linux
		DetectorConfig: filepaths.RootDetectorConfig + "/" + userParams.DetectorConfig + "/",
		Elements:       userParams.Elements,
		Parameters:     fmt.Sprintf("%v -t,%v", userParams.Parameters, quantStartSettings.CoresPerNode),
		//DatasetsBucket:    params.DatasetsBucket,
		//ConfigBucket:      params.ConfigBucket,
		DatasetsBucket:    svcs.Config.DatasetsBucket,
		ConfigBucket:      svcs.Config.ConfigBucket,
		PiquantJobsBucket: quantStartSettings.PiquantJobsBucket,
		QuantName:         userParams.Name,
		PMCListName:       "", // PMC List Name will be filled in later
		Command:           userParams.Command,
	}

	piquantParamsStr, err := json.MarshalIndent(piquantParams, "", utils.PrettyPrintIndentForJSON)
	if err == nil {
		svcs.Log.Debugf("Piquant parameters: %v\n", string(piquantParamsStr))
	}

	// Generate the lists, and then save each, and start the quantification
	// NOTE: empty == combined, just to honor the previous mode of operation before quantMode field was added
	combined := userParams.QuantMode == "" || userParams.QuantMode == quantModeCombinedABBulk || userParams.QuantMode == quantModeCombinedAB
	quantByROI := userParams.QuantMode == quantModeCombinedABBulk || userParams.QuantMode == quantModeSeparateABBulk || userParams.Command != "map"

	// If we're quantifying ROIs, do that
	pmcFiles := []string{}
	spectraPerNode := int32(0)
	err = nil
	rois := []roiItemWithPMCs{}

	// Download the dataset itself because we'll need it to generate our .pmcs files for each node to run
	dataset, err := wsHelpers.ReadDatasetFile(userParams.ScanId, svcs)
	if err != nil {
		job.CompleteJob(jobId, false, fmt.Sprintf("Error: %v", err), "", []string{}, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
		return
	}
	if quantByROI {
		pmcFile := ""
		pmcFile, spectraPerNode, rois, err = makePMCListFilesForQuantROI(hctx, combined, svcs.Config, datasetFileName, jobDataPath, quantStartSettings, dataset)
		pmcFiles = []string{pmcFile}
	} else {
		pmcFiles, spectraPerNode, err = makePMCListFilesForQuantPMCs(svcs, combined, svcs.Config, datasetFileName, jobDataPath, quantStartSettings, dataset)
	}

	if err != nil {
		job.CompleteJob(jobId, false, fmt.Sprintf("Error: %v", err), "", []string{}, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
		return
	}

	// Save running state as we are blocked after this!
	job.UpdateJob(jobId, protos.JobStatus_RUNNING, fmt.Sprintf("Node count: %v, Spectra/Node: %v", len(pmcFiles), spectraPerNode), logId, svcs.MongoDB, svcs.TimeStamper, svcs.Log)

	// Run piquant job(s)
	runner.RunPiquant(quantStartSettings.PIQUANTVersion, piquantParams, pmcFiles, svcs.Config, quantStartSettings.RequestorUserId, svcs.Log)

	// Generate the output path for all generated data files & logs
	quantOutPath := filepaths.GetQuantPath(quantStartSettings.RequestorUserId, userParams.ScanId, "")

	outputCSVName := ""
	outputCSVBytes := []byte{}
	outputCSV := ""

	piquantLogList := []string{}

	if userParams.Command == "map" {
		job.UpdateJob(jobId, protos.JobStatus_GATHERING_RESULTS, fmt.Sprintf("Combining CSVs from %v nodes...", len(pmcFiles)), logId, svcs.MongoDB, svcs.TimeStamper, svcs.Log)

		// Gather log files straight away, we want any status updates to include the logs!
		piquantLogList, err = copyAllLogs(
			svcs.FS,
			svcs.Log,
			quantStartSettings.PiquantJobsBucket,
			jobDataPath,
			svcs.Config.UsersBucket,
			path.Join(quantOutPath, filepaths.MakeQuantLogDirName(jobId)),
			jobId,
		)

		// Now we can combine the outputs from all runners
		csvTitleRow := fmt.Sprintf("PIQUANT version: %v DetectorConfig: %v", quantStartSettings.PIQUANTVersion, userParams.DetectorConfig)
		err = nil

		// Again, if we're in ROI mode, we act differently
		errMsg := ""

		if quantByROI {
			outputCSV, err = processQuantROIsToPMCs(svcs.FS, svcs.Config.PiquantJobsBucket, jobDataPath, csvTitleRow, pmcFiles[0], combined, rois)
			errMsg = "Error when duplicating quant rows for ROI PMCs"
		} else {
			outputCSV, err = combineQuantOutputs(svcs.FS, svcs.Config.PiquantJobsBucket, jobDataPath, csvTitleRow, pmcFiles)
			errMsg = "Error when combining quants"
		}
		if err != nil {
			job.CompleteJob(jobId, false, fmt.Sprintf("%v: %v", errMsg, err), "", piquantLogList, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
			return
		}

		outputCSVBytes = []byte(outputCSV)
		outputCSVName = "combined.csv"
	} else {
		// NOTE: Missing status writes - we only write those for map commands! saveQuantJobStatus quits if it's not a map anyway...

		// Complete writing to the jobs bucket
		// Read the resulting CSV
		jobOutputPath := path.Join(jobDataPath, "output")

		// Make the assumed output path
		piquantOutputPath := path.Join(jobOutputPath, pmcFiles[0]+"_result.csv")

		data, err := svcs.FS.ReadObject(svcs.Config.PiquantJobsBucket, piquantOutputPath)
		if err != nil {
			svcs.Log.Errorf("Failed to read PIQUANT output data from: s3://%v/%v. Error: %v", svcs.Config.PiquantJobsBucket, piquantOutputPath, err)
			outputCSVBytes = []byte{}
			outputCSVName = ""
		} else {
			outputCSVBytes = data
			outputCSVName = "result.csv"
		}
	}

	// Save to S3
	csvOutPath := path.Join(jobRoot, jobId, "output", outputCSVName)
	svcs.FS.WriteObject(svcs.Config.PiquantJobsBucket, csvOutPath, outputCSVBytes)

	if userParams.Command != "map" {
		// Map commands are more complicated, where they generate status and summaries, the csv, and the protobuf bin version of the csv, etc
		// but all other commands are far simpler.

		// Clear the previously written files
		csvUserFilePath := filepaths.GetUserLastPiquantOutputPath(quantStartSettings.RequestorUserId, userParams.ScanId, userParams.Command, filepaths.QuantLastOutputFileName+".csv")
		userLogFilePath := filepaths.GetUserLastPiquantOutputPath(quantStartSettings.RequestorUserId, userParams.ScanId, userParams.Command, filepaths.QuantLastOutputLogName)

		err = svcs.FS.DeleteObject(svcs.Config.UsersBucket, csvUserFilePath)
		if err != nil {
			svcs.Log.Errorf("Failed to delete previous piquant output for command %v from s3://%v/%v. Error: %v", userParams.Command, svcs.Config.UsersBucket, csvUserFilePath, err)
		}
		err = svcs.FS.DeleteObject(svcs.Config.UsersBucket, userLogFilePath)
		if err != nil {
			svcs.Log.Errorf("Failed to delete previous piquant log for command %v from s3://%v/%v. Error: %v", userParams.Command, svcs.Config.UsersBucket, userLogFilePath, err)
		}

		// Upload the results file to the user bucket spot
		if len(outputCSVBytes) > 0 {
			err = svcs.FS.WriteObject(svcs.Config.UsersBucket, csvUserFilePath, outputCSVBytes)

			if err != nil {
				svcs.Log.Errorf("Failed to write output data (length=%v bytes) to user destination path s3://%v/%v", len(outputCSVBytes), svcs.Config.UsersBucket, csvUserFilePath, err)
			}
		}

		// We also write out the log file to the user bucket
		logSourcePath := path.Join(jobDataPath, filepaths.PiquantLogSubdir)
		files, err := svcs.FS.ListObjects(svcs.Config.PiquantJobsBucket, logSourcePath)
		if err != nil {
			svcs.Log.Errorf("Failed to retrieve log files from PIQUANT run from: s3://%v/%v", svcs.Config.PiquantJobsBucket, logSourcePath)
		} else {
			// It should have only ONE log! Anyway, write the first one...
			if len(files) > 0 {
				err := svcs.FS.CopyObject(
					svcs.Config.PiquantJobsBucket,
					files[0],
					svcs.Config.UsersBucket,
					userLogFilePath,
				)

				if err != nil {
					svcs.Log.Errorf("Failed to copy log file: %v://%v to data bucket destination: %v", svcs.Config.PiquantJobsBucket, files[0], userLogFilePath)
				}
			}
		}

		// STOP HERE! Non-map commands are simpler, map commands do a whole bunch more to maintain state files which are picked up
		// by quant listing generation
		return
	}

	// Convert to binary format
	binFileBytes, elements, err := ConvertQuantificationCSV(svcs.Log, outputCSV, []string{"PMC", "SCLK", "RTT", "filename"}, nil, false, "", false)
	if err != nil {
		job.CompleteJob(jobId, false, fmt.Sprintf("Error when converting quant CSV to PIXLISE bin: %v", err), quantOutPath, piquantLogList, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
		return
	}

	// Figure out file paths
	binFilePath := filepaths.GetQuantPath(quantStartSettings.RequestorUserId, userParams.ScanId, filepaths.MakeQuantDataFileName(jobId))
	csvFilePath := filepaths.GetQuantPath(quantStartSettings.RequestorUserId, userParams.ScanId, filepaths.MakeQuantCSVFileName(jobId))

	// Save bin quant to S3
	err = svcs.FS.WriteObject(svcs.Config.UsersBucket, binFilePath, binFileBytes)
	if err != nil {
		msg := fmt.Sprintf("Error when uploading converted PIXLISE bin file to s3 at \"s3://%v / %v\": %v", svcs.Config.UsersBucket, binFilePath, err)
		job.CompleteJob(jobId, false, msg, quantOutPath, piquantLogList, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
		return
	}

	// Save combined CSV to where we have the bin file too
	err = svcs.FS.WriteObject(svcs.Config.UsersBucket, csvFilePath, outputCSVBytes)
	if err != nil {
		// Non-job-ending error, can't save the CSV... it means it just won't be available when exporting. Still log error about it
		svcs.Log.Errorf("Failed to upload quant CSV file to s3 at \"s3://%v / %v\": %v", svcs.Config.UsersBucket, csvFilePath, err)
	}

	completeMsg := fmt.Sprintf("Nodes ran: %v", len(pmcFiles))
	now := svcs.TimeStamper.GetTimeNowSec()
	summary := &protos.QuantificationSummary{
		Id:       jobId,
		ScanId:   userParams.ScanId,
		Params:   quantStartSettings,
		Elements: elements,
		Status: &protos.JobStatus{
			JobId:          jobId,
			Status:         protos.JobStatus_COMPLETE,
			Message:        completeMsg,
			EndUnixTimeSec: uint32(now),
			OutputFilePath: quantOutPath,
			OtherLogFiles:  piquantLogList,
		},
	}

	ownerItem, err := wsHelpers.MakeOwnerForWrite(jobId, protos.ObjectType_OT_QUANTIFICATION, quantStartSettings.RequestorUserId, now)
	if err != nil {
		msg := fmt.Sprintf("Failed to create ownership info for quant job %v. Error was: %v", jobId, err)
		job.CompleteJob(jobId, false, msg, quantOutPath, piquantLogList, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
		return
	}

	err = writeQuantAndOwnershipToDB(summary, ownerItem, hctx.Svcs.MongoDB)
	if err != nil {
		job.CompleteJob(jobId, false, fmt.Sprintf("Failed to write quantification and ownership to DB: %v. Id: %v", err, jobId), quantOutPath, piquantLogList, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
		return
	}

	// Report success
	job.CompleteJob(jobId, true, completeMsg, quantOutPath, piquantLogList, svcs.MongoDB, svcs.TimeStamper, svcs.Log)
}

func copyAllLogs(fs fileaccess.FileAccess, jobLog logger.ILogger, jobBucket string, jobDataPath string, usersBucket string, logSavePath string, jobID string) ([]string, error) {
	result := []string{}

	logSourcePath := path.Join(jobDataPath, filepaths.PiquantLogSubdir)
	files, err := fs.ListObjects(jobBucket, logSourcePath)
	if err != nil {
		return result, err
	}

	for _, item := range files {
		// Copy all log files to the data bucket and generate a link to save in the status object
		fileName := cleanLogName(filepath.Base(item))

		dstPath := path.Join(logSavePath, fileName)

		err := fs.CopyObject(
			jobBucket,
			item,
			usersBucket,
			dstPath,
		)

		if err != nil {
			jobLog.Errorf("Failed to copy log file: %v://%v to data bucket destination: %v", jobBucket, item, dstPath)
		}

		// Remember link to the file... this is really now just a list of file names
		result = append(result, fileName)
	}

	return result, nil
}

func cleanLogName(logName string) string {
	result := logName

	// We've got this ugly situation where we just append something to pmcs file name for the node
	// to form a log name. We can fix it here

	toReplace := ".pmcs_"
	lowered := strings.ToLower(logName)
	idx := strings.Index(lowered, toReplace)
	if idx > -1 {
		result = logName[0:idx] + "_" + logName[idx+len(toReplace):]
	}

	return result
}
