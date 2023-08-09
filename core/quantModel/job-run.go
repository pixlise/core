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

package quantModel

import (
	"encoding/json"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/notifications"

	datasetModel "github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/piquant"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/roiModel"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"

	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/logger"
)

const quantModeSeparateAB = "AB"
const quantModeCombinedAB = "Combined"
const quantModeSeparateABBulk = "ABBulk"
const quantModeCombinedABBulk = "CombinedBulk"
const QuantModeCombinedManualUpload = "ABManual"
const QuantModeABManualUpload = "ABManual"
const QuantModeCombinedMultiQuant = "CombinedMultiQuant"
const QuantModeABMultiQuant = "ABMultiQuant"

// CreateJob - creates a new quantification job
func CreateJob(svcs *services.APIServices, createParams JobCreateParams, wg *sync.WaitGroup) (string, error) {
	jobID := svcs.IDGen.GenObjectID()

	// NOTE: if we're NOT running a map job, we make weird job IDs that help us identify this as a piquant that doesn't need to be
	// treated as a long-running job
	if createParams.Command != "map" {
		// Make the name and ID the same, and start with something that stands out
		jobID = "cmd-" + createParams.Command + "-" + jobID
		createParams.Name = jobID
	}

	startTime := time.Now().Unix()

	createMsg := fmt.Sprintf("quantCreate: %v, %v pmcs, elems=%v, cfg=%v, params=%v. Job ID: %v", createParams.DatasetPath, len(createParams.PMCs), createParams.Elements, createParams.DetectorConfig, createParams.Parameters, jobID)
	svcs.Log.Infof(createMsg)

	coresPerNode := svcs.Config.CoresPerNode

	var jobLog logger.ILogger
	var err error

	// Init a logger for this job
	//if svcs.Config.EnvironmentName == "local" {
	jobLog = &logger.StdOutLogger{}
	/*} else {
		jobLog, err = logger.InitCloudWatchLogger(
			svcs.AWSSessionCW,
			"/api/"+svcs.Config.EnvironmentName,
			"job-"+jobID,
			svcs.Config.LogLevel,
			30, // Log retention for 30 days
			3,  // Send logs every 3 seconds in batches
		)
		if err != nil {
			svcs.Log.Errorf("Failed to create logger for Job ID: %v", jobID)
		}
	}*/

	jobLog.Infof(createMsg)

	if len(createParams.PMCs) <= 0 {
		txt := "No PMCs specified, quant job not created"
		jobLog.Errorf(txt)
		return jobID, errors.New(txt)
	}

	// Search for weird characters in parameters. We don't want to allow people to do
	// command injection attacks here!! PIQUANT commands are fairly simple and take
	// flags eg -b often with values right after, comma separated. So we allow
	// only a few characters, to exclude things like ; and & so users can't form other
	// commands
	if len(createParams.Parameters) > 0 {
		err := validateParameters(createParams.Parameters)
		if err != nil {
			jobLog.Errorf("%v", err)
			return jobID, err
		}
	}

	// Get configured PIQUANT docker container
	piquantVersion, err := piquant.GetPiquantVersion(svcs)

	if err != nil || len(piquantVersion.Version) <= 0 {
		txt := "Failed to get PIQUANT version configuration"
		jobLog.Errorf(txt)
		return jobID, errors.New(txt)
	}

	// Set up starting parameters
	params := JobStartingParametersWithPMCs{
		PMCs: createParams.PMCs,
		JobStartingParameters: &JobStartingParameters{
			Name:       createParams.Name,
			DataBucket: svcs.Config.DatasetsBucket,
			//ConfigBucket:      svcs.Config.ConfigBucket,
			DatasetPath:       createParams.DatasetPath,
			DatasetID:         createParams.DatasetID,
			PiquantJobsBucket: svcs.Config.PiquantJobsBucket,
			DetectorConfig:    createParams.DetectorConfig,
			Elements:          createParams.Elements,
			Parameters:        createParams.Parameters,
			RunTimeSec:        createParams.RunTimeSec,
			CoresPerNode:      coresPerNode,
			StartUnixTime:     startTime,
			Creator:           createParams.Creator,
			RoiID:             createParams.RoiID,
			RoiIDs:            createParams.RoiIDs,
			ElementSetID:      createParams.ElementSetID,
			PIQUANTVersion:    piquantVersion.Version,
			QuantMode:         createParams.QuantMode,
			IncludeDwells:     createParams.IncludeDwells,
			Command:           createParams.Command,
		},
	}

	// Fallback if we're using wrong UI version with this API version
	// TODO: Could be removed after about August 2022...
	if len(params.Command) <= 0 {
		params.Command = "map"
	}

	// Save...
	paramsPath := filepaths.GetJobDataPath(createParams.DatasetID, jobID, JobParamsFileName)
	err = svcs.FS.WriteJSON(svcs.Config.PiquantJobsBucket, paramsPath, params)
	if err != nil {
		jobLog.Errorf("Failed to upload params: %v", err)
		return jobID, err
	}

	// Save job status
	var status JobStatus
	status.JobID = jobID
	setJobStatus(&status, JobStarting, "Job started")
	saveQuantJobStatus(svcs, createParams.DatasetID, params.Name, &status, jobLog, createParams.Creator)

	wg.Add(1)

	// Trigger task to start in a go routine, so we don't block!
	go triggerPiquantNodes(svcs, jobLog, jobID, svcs.Config, params, svcs.Notifications, createParams.Creator, wg)

	return jobID, nil
}

// This should be triggered as a go routine from quant creation endpoint so we can return a job id there quickly and do the processing offline
func triggerPiquantNodes(svcs *services.APIServices, jobLog logger.ILogger, jobID string, cfg config.APIConfig, params JobStartingParametersWithPMCs, notifications notifications.NotificationManager, creator pixlUser.UserInfo, wg *sync.WaitGroup) {
	defer wg.Done()

	jobRoot := filepaths.GetJobDataPath(params.DatasetID, "", "")
	jobDataPath := filepaths.GetJobDataPath(params.DatasetID, jobID, "")

	var status JobStatus
	status.JobID = jobID

	// Get quant runner interface
	runner, err := getQuantRunner(cfg.QuantExecutor)
	if err != nil {
		setJobError(&status, fmt.Sprintf("Failed to start quant runner: %v", err))
		saveQuantJobStatus(svcs, params.DatasetID, params.Name, &status, jobLog, creator)
		return
	}

	setJobStatus(&status, JobPreparingNodes, fmt.Sprintf("Cores/Node: %v", params.CoresPerNode))
	saveQuantJobStatus(svcs, params.DatasetID, params.Name, &status, jobLog, creator)

	datasetFileName := path.Base(params.DatasetPath)
	datasetPathOnly := path.Dir(params.DatasetPath)

	// Gather required params (these are static, same data passed to each node)
	piquantParams := PiquantParams{
		RunTimeEnv:  cfg.EnvironmentName,
		JobID:       jobID,
		JobsPath:    jobRoot,
		DatasetPath: datasetPathOnly,
		// NOTE: not using path.Join because we want this as / deliberately, this is being
		//       saved in a config file that runs in docker/linux
		DetectorConfig: filepaths.RootDetectorConfig + "/" + params.DetectorConfig + "/",
		Elements:       params.Elements,
		Parameters:     fmt.Sprintf("%v -t,%v", params.Parameters, params.CoresPerNode),
		//DatasetsBucket:    params.DatasetsBucket,
		//ConfigBucket:      params.ConfigBucket,
		DatasetsBucket:    cfg.DatasetsBucket,
		ConfigBucket:      cfg.ConfigBucket,
		PiquantJobsBucket: params.PiquantJobsBucket,
		QuantName:         params.Name,
		PMCListName:       "", // PMC List Name will be filled in later
		Command:           params.Command,
	}

	piquantParamsStr, err := json.MarshalIndent(piquantParams, "", utils.PrettyPrintIndentForJSON)
	if err == nil {
		jobLog.Debugf("Piquant parameters: %v\n", string(piquantParamsStr))
	}

	// Generate the lists, and then save each, and start the quantification
	// NOTE: empty == combined, just to honor the previous mode of operation before quantMode field was added
	combined := params.QuantMode == "" || params.QuantMode == quantModeCombinedABBulk || params.QuantMode == quantModeCombinedAB
	quantByROI := params.QuantMode == quantModeCombinedABBulk || params.QuantMode == quantModeSeparateABBulk || params.Command != "map"

	// If we're quantifying ROIs, do that
	pmcFiles := []string{}
	spectraPerNode := int32(0)
	err = nil
	rois := []ROIWithPMCs{}

	// Download the dataset itself because we'll need it to generate our .pmcs files for each node to run
	datasetPath := filepaths.GetDatasetFilePath(params.DatasetID, filepaths.DatasetFileName)
	dataset, err := datasetModel.GetDataset(svcs, datasetPath)
	if err != nil {
		jobLog.Errorf("Error: %v", err)
		setJobError(&status, err.Error())
		saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)
		return
	}
	if quantByROI {
		pmcFile := ""
		pmcFile, spectraPerNode, rois, err = makePMCListFilesForQuantROI(svcs, combined, jobLog, cfg, datasetFileName, jobDataPath, params, dataset)
		pmcFiles = []string{pmcFile}
	} else {
		pmcFiles, spectraPerNode, err = makePMCListFilesForQuantPMCs(svcs, combined, jobLog, cfg, datasetFileName, jobDataPath, params, dataset)
	}

	if err != nil {
		setJobError(&status, err.Error())
		saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)
		return
	}

	// Save running state as we are blocked after this!
	setJobStatus(&status, JobNodesRunning, fmt.Sprintf("Node count: %v, Spectra/Node: %v", len(pmcFiles), spectraPerNode))
	saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)

	// Run piquant job(s)
	runner.runPiquant(params.PIQUANTVersion, piquantParams, pmcFiles, cfg, notifications, creator, jobLog)

	// Generate the output path for all generated data files & logs
	quantOutPath := filepaths.GetUserQuantPath(params.Creator.UserID, params.DatasetID, "")

	outputCSVName := ""
	outputCSVBytes := []byte{}
	outputCSV := ""

	if params.Command == "map" {
		setJobStatus(&status, JobGatheringResults, fmt.Sprintf("Combining CSVs from %v nodes...", len(pmcFiles)))
		// Gather log files straight away, we want any status updates to include the logs!
		status.PiquantLogList, err = copyAllLogs(
			svcs.FS,
			jobLog,
			params.PiquantJobsBucket,
			jobDataPath,
			cfg.UsersBucket,
			path.Join(quantOutPath, filepaths.MakeQuantLogDirName(jobID)),
			jobID,
		)
		saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)

		// Now we can combine the outputs from all runners
		csvTitleRow := fmt.Sprintf("PIQUANT version: %v DetectorConfig: %v", params.PIQUANTVersion, params.DetectorConfig)
		err = nil

		// Again, if we're in ROI mode, we act differently
		errMsg := ""

		if quantByROI {
			outputCSV, err = processQuantROIsToPMCs(svcs.FS, cfg.PiquantJobsBucket, jobDataPath, csvTitleRow, pmcFiles[0], combined, rois)
			errMsg = "Error when duplicating quant rows for ROI PMCs"
		} else {
			outputCSV, err = combineQuantOutputs(svcs.FS, cfg.PiquantJobsBucket, jobDataPath, csvTitleRow, pmcFiles)
			errMsg = "Error when combining quants"
		}
		if err != nil {
			setJobError(&status, fmt.Sprintf("%v: %v", errMsg, err))
			saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)
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

		data, err := svcs.FS.ReadObject(cfg.PiquantJobsBucket, piquantOutputPath)
		if err != nil {
			jobLog.Errorf("Failed to read PIQUANT output data from: s3://%v/%v. Error: %v", cfg.PiquantJobsBucket, piquantOutputPath, err)
			outputCSVBytes = []byte{}
			outputCSVName = ""
		} else {
			outputCSVBytes = data
			outputCSVName = "result.csv"
		}
	}

	// Save to S3
	csvOutPath := path.Join(jobRoot, jobID, "output", outputCSVName)
	svcs.FS.WriteObject(cfg.PiquantJobsBucket, csvOutPath, outputCSVBytes)

	if params.Command != "map" {
		// Map commands are more complicated, where they generate status and summaries, the csv, and the protobuf bin version of the csv, etc
		// but all other commands are far simpler.

		// Clear the previously written files
		csvUserFilePath := filepaths.GetUserLastPiquantOutputPath(params.Creator.UserID, params.DatasetID, params.Command, filepaths.QuantLastOutputFileName+".csv")
		userLogFilePath := filepaths.GetUserLastPiquantOutputPath(params.Creator.UserID, params.DatasetID, params.Command, filepaths.QuantLastOutputLogName)

		err = svcs.FS.DeleteObject(cfg.UsersBucket, csvUserFilePath)
		if err != nil {
			jobLog.Errorf("Failed to delete previous piquant output for command %v from s3://%v/%v. Error: %v", params.Command, cfg.UsersBucket, csvUserFilePath, err)
		}
		err = svcs.FS.DeleteObject(cfg.UsersBucket, userLogFilePath)
		if err != nil {
			jobLog.Errorf("Failed to delete previous piquant log for command %v from s3://%v/%v. Error: %v", params.Command, cfg.UsersBucket, userLogFilePath, err)
		}

		// Upload the results file to the user bucket spot
		if len(outputCSVBytes) > 0 {
			err = svcs.FS.WriteObject(cfg.UsersBucket, csvUserFilePath, outputCSVBytes)

			if err != nil {
				jobLog.Errorf("Failed to write output data (length=%v bytes) to user destination path s3://%v/%v", len(outputCSVBytes), cfg.UsersBucket, csvUserFilePath, err)
			}
		}

		// We also write out the log file to the user bucket
		logSourcePath := path.Join(jobDataPath, filepaths.PiquantLogSubdir)
		files, err := svcs.FS.ListObjects(cfg.PiquantJobsBucket, logSourcePath)
		if err != nil {
			jobLog.Errorf("Failed to retrieve log files from PIQUANT run from: s3://%v/%v", cfg.PiquantJobsBucket, logSourcePath)
		} else {
			// It should have only ONE log! Anyway, write the first one...
			if len(files) > 0 {
				err := svcs.FS.CopyObject(
					cfg.PiquantJobsBucket,
					files[0],
					cfg.UsersBucket,
					userLogFilePath,
				)

				if err != nil {
					jobLog.Errorf("Failed to copy log file: %v://%v to data bucket destination: %v", cfg.PiquantJobsBucket, files[0], userLogFilePath)
				}
			}
		}

		// STOP HERE! Non-map commands are simpler, map commands do a whole bunch more to maintain state files which are picked up
		// by quant listing generation
		return
	}

	// Convert to binary format
	binFileBytes, elements, err := ConvertQuantificationCSV(jobLog, outputCSV, []string{"PMC", "SCLK", "RTT", "filename"}, "", false, "", false)
	if err != nil {
		setJobError(&status, fmt.Sprintf("Error when converting quant CSV to PIXLISE bin: %v", err))
		saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)
		return
	}

	// Figure out file paths
	binFilePath := filepaths.GetUserQuantPath(params.Creator.UserID, params.DatasetID, filepaths.MakeQuantDataFileName(jobID))
	summaryFilePath := filepaths.GetUserQuantPath(params.Creator.UserID, params.DatasetID, filepaths.MakeQuantSummaryFileName(jobID))
	csvFilePath := filepaths.GetUserQuantPath(params.Creator.UserID, params.DatasetID, filepaths.MakeQuantCSVFileName(jobID))

	// Save bin quant to S3
	err = svcs.FS.WriteObject(cfg.UsersBucket, binFilePath, binFileBytes)
	if err != nil {
		setJobError(&status, fmt.Sprintf("Error when uploading converted PIXLISE bin file to s3 at \"s3://%v / %v\": %v", cfg.UsersBucket, binFilePath, err))
		saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)
		return
	}

	// Save status info
	setJobStatus(&status, JobComplete, fmt.Sprintf("Nodes ran: %v", len(pmcFiles)))
	status.EndUnixTime = time.Now().Unix()
	status.OutputFilePath = quantOutPath

	saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)

	// Finally, output a "summary" file to go with the quant, so API can quickly load up its metadata
	summary := JobSummaryItem{
		Shared:    false,
		Params:    MakeJobStartingParametersWithPMCCount(params),
		Elements:  elements,
		JobStatus: &status,
	}

	summaryData, err := json.MarshalIndent(summary, "", utils.PrettyPrintIndentForJSON)
	if err == nil {
		err = svcs.FS.WriteObject(cfg.UsersBucket, summaryFilePath, summaryData)
	}

	if err != nil {
		setJobError(&status, fmt.Sprintf("Failed to upload quant summary file to s3 at \"s3://%v / %v\": %v", cfg.UsersBucket, summaryFilePath, err))
		saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)
	}

	// Save combined CSV to where we have the bin file too
	err = svcs.FS.WriteObject(cfg.UsersBucket, csvFilePath, outputCSVBytes)
	if err != nil {
		setJobError(&status, fmt.Sprintf("Failed to upload quant CSV file to s3 at \"s3://%v / %v\": %v", cfg.UsersBucket, csvFilePath, err))
		saveQuantJobStatus(svcs, params.DatasetID, piquantParams.QuantName, &status, jobLog, creator)
	}
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

type ROIWithPMCs struct {
	PMCs []int
	ID   string
	*roiModel.ROISavedItem
}

// REFACTOR: deduplicate, we have too many getROIs!
func getROIs(svcs *services.APIServices, params JobStartingParametersWithPMCs, locIdxToPMCLookup map[int32]int32, dataset *protos.Experiment) ([]ROIWithPMCs, error) {
	result := []ROIWithPMCs{}
	var err error

	if len(params.RoiIDs) <= 0 {
		// If we're in a map command, this is bad, as we want to have a list of ROIs to generate for
		if params.Command == "map" {
			return result, errors.New("No ROI IDs specified for sum-then-quantify mode")
		} else {
			// For anything else, we can just add the single ROI specified (or AllPoints if that's empty)
			if len(params.RoiID) <= 0 {
				params.RoiIDs = append(params.RoiIDs, "AllPoints")
			} else {
				params.RoiIDs = append(params.RoiIDs, params.RoiID)
			}
		}
	}

	userROIs := roiModel.ROILookup{}
	sharedROIs := roiModel.ROILookup{}

	s3Path := filepaths.GetROIPath(params.Creator.UserID, params.DatasetID)
	userROIsError := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &userROIs, true)
	if userROIsError != nil && !svcs.FS.IsNotFoundError(err) {
		return result, fmt.Errorf("Failed to download user ROI list: %v", err)
	}

	s3Path = filepaths.GetROIPath(pixlUser.ShareUserID, params.DatasetID)
	sharedROIsError := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &sharedROIs, true)
	if sharedROIsError != nil && !svcs.FS.IsNotFoundError(err) {
		return result, fmt.Errorf("Failed to download shared ROI list: %v", err)
	}

	// Find the ROIs the user specified
	for _, roiID := range params.RoiIDs {
		if roiID == "AllPoints" {
			roiWithPMCs := makeAllPointsROI(dataset)
			result = append(result, *roiWithPMCs)
		} else {
			if roi, ok := userROIs[roiID]; ok {
				roiWithPMCs, err := makeROIWithPMCs(roiID, roi, locIdxToPMCLookup)
				if err != nil {
					return result, err
				}
				result = append(result, *roiWithPMCs)
			} else if roi, ok := sharedROIs[utils.SharedItemIDPrefix+roiID]; ok {
				roiWithPMCs, err := makeROIWithPMCs(roiID, roi, locIdxToPMCLookup)
				if err != nil {
					return result, err
				}
				result = append(result, *roiWithPMCs)
			}
		}
	}

	return result, nil
}

func makeROIWithPMCs(roiID string, roi roiModel.ROISavedItem, locIdxToPMCLookup map[int32]int32) (*ROIWithPMCs, error) {
	pmcs := []int{}
	for _, locIdx := range roi.LocationIndexes {
		if pmc, ok := locIdxToPMCLookup[locIdx]; ok {
			pmcs = append(pmcs, int(pmc))
		}
		// We used to error here, but now that we're filtering out PMCs that have no normal/dwell spectra, this is a valid scenario
		// where an ROI contained a housekeeping PMC and the quant would've failed unless we filter out the bad PMC here.
		/* else {
			return nil, fmt.Errorf("Failed to find PMC for loc idx: %v in ROI: %v, ROI id: %v", locIdx, roi.Name, roiID)
		}*/
	}

	sort.Ints(pmcs)

	roiWithPMCs := &ROIWithPMCs{
		PMCs:         pmcs,
		ID:           roiID,
		ROISavedItem: &roi,
	}

	return roiWithPMCs, nil
}

func makeAllPointsROI(dataset *protos.Experiment) *ROIWithPMCs {
	const roiID = "AllPoints"
	allPoints := roiModel.GetAllPointsROI(dataset)

	// Need ints here :-/
	allPointsI := make([]int, len(allPoints.PMCs))
	for c, pmc := range allPoints.PMCs {
		allPointsI[c] = int(pmc)
	}

	sort.Ints(allPointsI)

	roi := roiModel.ROISavedItem{
		ROIItem: &roiModel.ROIItem{
			Name:            "All Points",
			LocationIndexes: allPoints.LocationIdxs,
			Description:     "All Points",
		},
		//ImageName: "",
		//PixelIndexes []int32
		//Shared: false,
		//Creator: UserInfo{
	}

	roiWithPMCs := &ROIWithPMCs{
		PMCs:         allPointsI,
		ID:           roiID,
		ROISavedItem: &roi,
	}

	return roiWithPMCs
}

func makePMCListFilesForQuantROI(svcs *services.APIServices, combinedSpectra bool, jobLog logger.ILogger, cfg config.APIConfig, datasetFileName string, jobDataPath string, params JobStartingParametersWithPMCs, dataset *protos.Experiment) (string, int32, []ROIWithPMCs, error) {
	// We're quantifying by ROIs, so we are actually adding all spectra in the ROI before quantifying once. First we need to download the ROIs
	// We will also need the dataset file so we can convert our roi LocIdx to PMCs
	locIdxToPMCLookup, err := datasetModel.MakeLocToPMCLookup(dataset, true)
	if err != nil {
		return "", 0, []ROIWithPMCs{}, err
	}

	rois, err := getROIs(svcs, params, locIdxToPMCLookup, dataset)
	if err != nil {
		return "", 0, rois, err
	}

	// Save list to file in S3 for piquant to pick up
	quantCount := int32(len(rois))
	if !combinedSpectra {
		quantCount *= 2
	}

	pmcHasDwellLookup, err := datasetModel.MakePMCHasDwellLookup(dataset)
	if err != nil {
		return "", 0, rois, err
	}

	contents, err := makeROIPMCListFileContents(rois, datasetFileName, combinedSpectra, params.IncludeDwells, pmcHasDwellLookup)
	if err != nil {
		return "", 0, rois, fmt.Errorf("Error when preparing quant ROI node list. Error: %v", err)
	}

	pmcListName, err := savePMCList(svcs, params.PiquantJobsBucket, contents, 1, jobDataPath)
	if err != nil {
		return "", 0, rois, err
	}

	return pmcListName, quantCount, rois, nil
}

func makePMCListFilesForQuantPMCs(svcs *services.APIServices, combinedSpectra bool, jobLog logger.ILogger, cfg config.APIConfig, datasetFileName string, jobDataPath string, params JobStartingParametersWithPMCs, dataset *protos.Experiment) ([]string, int32, error) {
	pmcFiles := []string{}

	// Work out how many quants we're running, therefore how many nodes we need to generate in a reasonable time frame
	spectraCount := int32(len(params.PMCs))
	if !combinedSpectra {
		spectraCount *= 2
	}

	nodeCount := estimateNodeCount(spectraCount, int32(len(params.Elements)), params.RunTimeSec, params.CoresPerNode, cfg.MaxQuantNodes)

	if cfg.NodeCountOverride > 0 {
		nodeCount = cfg.NodeCountOverride
		jobLog.Infof("Using node count override: %v", nodeCount)
	}

	// NOTE: if we're running anything but the map command, the result is pretty quick, so we don't need to farm it out to multiple nodes
	if params.Command != "map" {
		nodeCount = 1
	}

	spectraPerNode := filesPerNode(spectraCount, nodeCount)
	pmcsPerNode := spectraPerNode
	if !combinedSpectra {
		// If we're separate, we have 2x as many spectra as PMCs, so here we calculate how many
		// pmcs per node accurately for the next step to generate the right number of PMC lists
		pmcsPerNode /= 2
	}

	jobLog.Debugf("spectraPerNode: %v, PMCs per node: %v for %v spectra, nodes: %v", spectraPerNode, pmcsPerNode, spectraCount, nodeCount)

	// Generate the lists and save to S3
	pmcLists := makeQuantJobPMCLists(params.PMCs, int(pmcsPerNode))

	pmcHasDwellLookup, err := datasetModel.MakePMCHasDwellLookup(dataset)
	if err != nil {
		return []string{}, 0, err
	}

	for i, pmcList := range pmcLists {
		// Serialise the data for the list
		contents, err := makeIndividualPMCListFileContents(pmcList, datasetFileName, combinedSpectra, params.IncludeDwells, pmcHasDwellLookup)

		if err != nil {
			return pmcFiles, 0, fmt.Errorf("Error when preparing node PMC list: %v. Error: %v", i, err)
		}

		pmcListName, err := savePMCList(svcs, params.PiquantJobsBucket, contents, i+1, jobDataPath)
		if err != nil {
			return []string{}, 0, err
		}

		pmcFiles = append(pmcFiles, pmcListName)
	}

	return pmcFiles, spectraPerNode, nil
}

func combineQuantOutputs(fs fileaccess.FileAccess, jobsBucket string, jobPath string, header string, pmcFilesUsed []string) (string, error) {
	// Try to load each PMC file, if any fail, fail due to 1 node either not finishing/crashing/etc
	jobOutputPath := path.Join(jobPath, "output")

	var sb strings.Builder

	// Write header:
	sb.WriteString(header + "\n")

	pmcLineLookup := map[int][]string{}
	pmcs := []int{}

	for c, v := range pmcFilesUsed {
		// Make the assumed output path
		piquantOutputPath := path.Join(jobOutputPath, v+"_result.csv")

		data, err := fs.ReadObject(jobsBucket, piquantOutputPath)
		if err != nil {
			return "", errors.New("Failed to combine map segment: " + piquantOutputPath)
		}

		// Read all rows in. We want to sort these by PMC, so store the rows in map by PMC
		rows := strings.Split(string(data), "\n")

		// We have the data, append it to our output data
		dataStartRow := 2 // PIQUANT CSV outputs usually have 2 rows of header data...

		for i, row := range rows {
			// Ensure PMC is 1st column
			if i == 1 && !strings.HasPrefix(row, "PMC,") {
				return "", fmt.Errorf("Map segment: %v, did not have PMC as first column", piquantOutputPath)
			}

			// If we're reading the first file, output its headers to the output file
			if c <= 0 && i > 0 && i < dataStartRow {
				sb.WriteString(row + "\n")
			}

			// Normal rows: save to our map so we can sort them before writing
			if i >= dataStartRow && len(row) > 0 {
				pmcPos := strings.Index(row, ",")
				if pmcPos < 1 {
					return "", fmt.Errorf("Failed to combine map segment: %v, no PMC at line %v", piquantOutputPath, i+1)
				}

				pmcStr := row[0:pmcPos]
				pmc64, err := strconv.ParseInt(pmcStr, 10, 32)
				if err != nil {
					return "", fmt.Errorf("Failed to combine map segment: %v, invalid PMC %v at line %v", piquantOutputPath, pmcStr, i+1)
				}

				pmc := int(pmc64)
				if _, ok := pmcLineLookup[pmc]; !ok {
					// Add an array for this PMC
					pmcLineLookup[pmc] = []string{}

					// Also save in pmc list so it can be sorted
					pmcs = append(pmcs, pmc)
				}

				// add it to the list of lines for this row
				pmcLineLookup[pmc] = append(pmcLineLookup[pmc], row)
			}
		}
	}

	// Sort the PMCs and read from map into file
	sort.Ints(pmcs)

	// Read PMCs in order and write to file
	for _, pmc := range pmcs {
		rows, ok := pmcLineLookup[pmc]
		if !ok {
			return "", fmt.Errorf("Failed to save row for PMC: %v", pmc)
		}

		for _, row := range rows {
			sb.WriteString(row + "\n")
		}
	}

	return sb.String(), nil
}

func processQuantROIsToPMCs(fs fileaccess.FileAccess, jobsBucket string, jobPath string, header string, piquantCSVFile string, combinedQuant bool, rois []ROIWithPMCs) (string, error) {
	// PIQUANT has summed then quantified the spectra belonging to PMCs in each ROI. We now have to take those rows
	// and copy them so each PMC in the ROI has a copy of the quantification row.
	jobOutputPath := path.Join(jobPath, "output")

	var sb strings.Builder

	// Write header:
	sb.WriteString(header + "\n")

	roiIdxToLineLookup := make([][]string, len(rois), len(rois))

	// Read in the piquant generated output that we're going to process
	// Make the assumed output path
	piquantOutputPath := path.Join(jobOutputPath, piquantCSVFile+"_result.csv")

	data, err := fs.ReadObject(jobsBucket, piquantOutputPath)
	if err != nil {
		return "", errors.New("Failed to read map CSV: " + piquantOutputPath)
	}

	// Read all rows in. We want to sort these by PMC, so store the rows in map by PMC
	rows := strings.Split(string(data), "\n")

	// We have the data, append it to our output data
	dataStartRow := 2 // PIQUANT CSV outputs usually have 2 rows of header data...
	fileNameColIdx := -1
	colCount := 0

	for i, row := range rows {
		// Ignore first row
		if i == 0 {
			continue
		}

		// Ensure PMC is 1st column
		if i == 1 {
			cols := strings.Split(row, ",")
			colCount = len(cols) // save for later
			for colIdx, col := range cols {
				colClean := strings.Trim(col, " \t")
				if colClean == "filename" {
					fileNameColIdx = colIdx
					break
				}
			}

			if fileNameColIdx < 0 {
				return "", fmt.Errorf("Map csv: %v, does not contain a filename column (used to match up ROIs)", piquantOutputPath)
			}
		}

		// Copy the header row
		if i < dataStartRow {
			sb.WriteString(row + "\n")
		} else {
			if len(row) > 0 {
				// Read the file name column and work out the ROI ID
				values := strings.Split(row, ",")

				// Verify we have the right amount
				if len(values) != colCount {
					return "", fmt.Errorf("Unexpected column count on map CSV: %v, line %v", piquantOutputPath, i+1)
				}

				fileName := strings.Trim(values[fileNameColIdx], " \t")

				// We expect file names of the form:
				// Normal_A_roiid
				// or Normal_Combined_roiid
				// This way we can confirm we're reading what we expect, and we know which roi to match to
				fileNameBits := strings.Split(fileName, "_")
				if len(fileNameBits) != 3 || fileNameBits[0] != "Normal" || (fileNameBits[1] != "Combined" && fileNameBits[1] != "A" && fileNameBits[1] != "B") || len(fileNameBits[2]) <= 0 {
					return "", fmt.Errorf("Invalid file name read: %v from map CSV: %v, line %v", fileName, piquantOutputPath, i+1)
				}

				// Work out the index of the ROI this applies to
				roiIdx := -1
				for idx, roi := range rois {
					if roi.ID == fileNameBits[2] {
						roiIdx = idx
						break
					}
				}

				// Make sure we found it...
				if roiIdx < 0 {
					return "", fmt.Errorf("CSV contained unexpected roi: \"%v\" when processing map CSV: %v", fileNameBits[2], piquantOutputPath)
				}

				// Also parse & validate PMC so we can read the rest of the row after it!
				pmcPos := strings.Index(row, ",")
				if pmcPos < 1 {
					return "", fmt.Errorf("Failed to process map CSV: %v, no PMC at line %v", piquantOutputPath, i+1)
				}

				pmcStr := row[0:pmcPos]
				pmc64, err := strconv.ParseInt(pmcStr, 10, 32)
				if err != nil {
					return "", fmt.Errorf("Failed to process map CSV: %v, invalid PMC %v at line %v", piquantOutputPath, pmcStr, i+1)
				}

				// Add line to the lookup
				roiIdxToLineLookup[roiIdx] = append(roiIdxToLineLookup[roiIdx], row[pmcPos:])

				// Sanity check: Verify that the PMC read exists in the ROI we think we're reading for
				pmc := int(pmc64)

				pmcFound := false
				for _, roiPMC := range rois[roiIdx].PMCs {
					if roiPMC == pmc {
						pmcFound = true
						break
					}
				}

				if !pmcFound {
					return "", fmt.Errorf("PMC %v in CSV: %v doesn't exist in ROI: %v", pmcStr, piquantOutputPath, rois[roiIdx].Name)
				}
			}
		}
	}

	// Now run through ROIs and write out line copies for each PMC
	for c, roi := range rois {
		for _, pmc := range roi.PMCs {
			for _, row := range roiIdxToLineLookup[c] {
				sb.WriteString(fmt.Sprintf("%v%v\n", pmc, row))
			}
		}
	}

	return sb.String(), nil
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

// Checks parameters don't contain something unexpected, to filter out
// code execution. Returns error or nil
func validateParameters(params string) error {
	r, err := regexp.Compile("^[a-zA-Z0-9 -.,_\"]+$")
	if err != nil {
		return err
	}

	if !r.MatchString(params) {
		return errors.New("Invalid parameters passed: " + params)
	}
	return nil
}
