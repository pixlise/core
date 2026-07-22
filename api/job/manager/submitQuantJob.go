package jobmanager

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/filepaths"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/piquant"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// Submit function for each kind of job type we support
func (jm *JobManager) SubmitQuantJob(createParams *protos.QuantCreateParams, requestorUserSess *sessionuser.SessionUser, requestorSession *melody.Session) (*protos.JobStatus, error) {
	prefix := "quant"
	jobType := protos.JobType_JT_UNKNOWN
	jobCompletionMethod := ""

	if createParams.Command != "map" {
		prefix = "piquant" + createParams.Command
		jobType = protos.JobType_JT_RUN_FIT
		jobCompletionMethod = JobComplete_SingleCSV
	} else {
		jobType = protos.JobType_JT_RUN_QUANT
		jobCompletionMethod = JobComplete_CombineCSVs
	}

	// Call the internal one, log the resulting errors if any
	status, err := jm.internalSubmitQuantJob(createParams, requestorUserSess, requestorSession, prefix, jobType, jobCompletionMethod)
	if err != nil {
		jm.svcs.Log.Errorf("SubmitQuantJob error: %v", err)
	}
	return status, err
}

func (jm *JobManager) internalSubmitQuantJob(
	createParams *protos.QuantCreateParams,
	requestorUserSess *sessionuser.SessionUser,
	requestorSession *melody.Session,
	idPrefix string,
	jobType protos.JobType,
	completeMethod string) (*protos.JobStatus, error) {
	err := quantification.IsValidCreateParam(createParams, jm.svcs, requestorUserSess)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// At this point, we're assuming that the detector config is a valid config name / version. We need this to be the path of the config in S3
	// so here we convert it and ensure it's valid
	detectorConfigBits := strings.Split(createParams.DetectorConfig, "/")
	if len(detectorConfigBits) != 2 || len(detectorConfigBits[0]) <= 0 || len(detectorConfigBits[1]) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("DetectorConfig not in expected format"))
	}
	/*
		// Form the string
		// NOTE: we would want to use this:
		// req.DetectorConfig = filepaths.GetDetectorConfigPath(detectorConfigBits[0], detectorConfigBits[1], "")
		// But can't because then the root "/DetectorConfig" is added twice!
		createParams.DetectorConfig = path.Join(detectorConfigBits[0], filepaths.PiquantConfigSubDir, detectorConfigBits[1])
	*/
	// Get the config and calibration files
	piquantCfg, err := piquant.GetPIQUANTConfig(jm.svcs, detectorConfigBits[0], detectorConfigBits[1])
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a job ID
	jobId := fmt.Sprintf("%v-%v", idPrefix, jm.svcs.IDGen.GenObjectID())

	jobS3Path := filepaths.GetJobDataPath(createParams.ScanId, jobId, "")

	// If we don't have a user, use the built-in PIXLISE user
	requestorUserId := sessionuser.PIXLISESystemUserId
	if requestorUserSess != nil {
		requestorUserId = requestorUserSess.User.Id
	}

	// Dataset file
	remoteCreateParamsPath := path.Join(jobS3Path, quantification.JobRequestFileName)

	requiredFiles := []jobconfig.JobFilePath{
		{
			LocalPath:    filepaths.DatasetFileName,
			RemoteBucket: jm.svcs.Config.DatasetsBucket,
			RemotePath:   filepaths.GetScanFilePath(createParams.ScanId, filepaths.DatasetFileName),
		},
		{
			LocalPath:    quantification.JobRequestFileName,
			RemoteBucket: jm.svcs.Config.PiquantJobsBucket,
			RemotePath:   remoteCreateParamsPath,
		},
	}

	// Write the user request struct out to S3 job, so we can access it later when the job is completed
	err = jm.svcs.FS.WriteJSON(jm.svcs.Config.PiquantJobsBucket, remoteCreateParamsPath, createParams)
	if err != nil {
		return nil, err
	}

	// PIQUANT instrument config files (these are config-dependent)
	if len(piquantCfg.ConfigFile) > 0 {
		requiredFiles = append(requiredFiles, jobconfig.JobFilePath{
			LocalPath:    piquantCfg.ConfigFile,
			RemoteBucket: jm.svcs.Config.ConfigBucket,
			RemotePath:   filepaths.GetDetectorConfigPath(detectorConfigBits[0], detectorConfigBits[1], piquantCfg.ConfigFile),
		})
	}
	if len(piquantCfg.CalibrationFile) > 0 {
		requiredFiles = append(requiredFiles, jobconfig.JobFilePath{
			LocalPath:    piquantCfg.CalibrationFile,
			RemoteBucket: jm.svcs.Config.ConfigBucket,
			RemotePath:   filepaths.GetDetectorConfigPath(detectorConfigBits[0], detectorConfigBits[1], piquantCfg.CalibrationFile),
		})
	}
	if len(piquantCfg.OpticEfficiencyFile) > 0 {
		requiredFiles = append(requiredFiles, jobconfig.JobFilePath{
			LocalPath:    piquantCfg.OpticEfficiencyFile,
			RemoteBucket: jm.svcs.Config.ConfigBucket,
			RemotePath:   filepaths.GetDetectorConfigPath(detectorConfigBits[0], detectorConfigBits[1], piquantCfg.OpticEfficiencyFile),
		})
	}

	// PMC list(s)
	nodePMCFileName := "node.pmcs"
	pmcFiles, _ /*spectraPerNode*/, rois, combined, quantByROI, err := quantification.PreparePMCLists(
		createParams, requestorUserSess, nodePMCFileName, jobS3Path, jm.svcs, jm.useFileCache)

	if err != nil {
		return nil, err
	}
	if len(pmcFiles) <= 0 {
		return nil, fmt.Errorf("Failed to create required PMC lists for quantification job %v nodes", jobId)
	}

	nodePMCPath := path.Join(jobS3Path, nodePMCFileName)
	requiredFiles = append(requiredFiles, jobconfig.JobFilePath{
		LocalPath:      nodePMCFileName,
		RemoteBucket:   jm.svcs.Config.PiquantJobsBucket,
		RemotePath:     nodePMCPath,
		ApplyNodeIndex: jobconfig.NodeIndexMethod_Both,
	})

	elementListStr := strings.Join(createParams.Elements, ",")

	// TODO: Bring this back somehow, or name the job runner docker container to include PIQUANT version
	// piquantVersion, err := piquant.GetPiquantVersion(jm.svcs)
	// if err != nil {
	// 	return err
	// }

	csvTitleRow := fmt.Sprintf("PIQUANT version: %v DetectorConfig: %v", jm.svcs.Config.Jobs.RunnerDockerImage /*piquantVersion.Version*/, createParams.DetectorConfig)

	userArgs := "-Fe,1"
	if len(createParams.Parameters) > 0 {
		userArgs = createParams.Parameters
	}
	extraArgs := strings.Split(fmt.Sprintf("%v -t,%v", userArgs, jm.svcs.Config.Jobs.CoresPerNode), " ")
	/*
		Command,
		config_file,
		calibration_file,
		pmc_list,
		element_list,
		out_path,
	*/
	allArgs := []string{
		createParams.Command,
		piquantCfg.ConfigFile,
		piquantCfg.CalibrationFile,
		nodePMCFileName,
		elementListStr,
		"map.csv",
	}
	allArgs = append(allArgs, extraArgs...)

	jg := &jobconfig.JobGroupConfig{
		JobGroupId:       jobId,
		JobType:          jobType,
		CompletionMethod: completeMethod,
		DockerImage:      jm.svcs.Config.Jobs.RunnerDockerImage,
		NodeCount:        uint(len(pmcFiles)),
		NodeConfig: jobconfig.JobConfig{
			JobId: jobId + "-node",

			RequiredFiles: requiredFiles,

			Command:                    "./Piquant",
			Args:                       allArgs,
			ArgIndexToApplyNodeIndexes: []int{3, 5},

			OutputFiles: []jobconfig.JobFilePath{
				{
					LocalPath:      "stdout",
					RemoteBucket:   jm.svcs.Config.PiquantJobsBucket,
					RemotePath:     path.Join(jobS3Path, "piquant-logs", "stdout.log"),
					ApplyNodeIndex: jobconfig.NodeIndexMethod_Remote,
				},
				{
					LocalPath:      "map.csv_log.txt",
					RemoteBucket:   jm.svcs.Config.PiquantJobsBucket,
					RemotePath:     path.Join(jobS3Path, "piquant-logs", "piquant.log"),
					ApplyNodeIndex: jobconfig.NodeIndexMethod_Both,
				},
				{
					LocalPath:      "map.csv",
					RemoteBucket:   jm.svcs.Config.PiquantJobsBucket,
					RemotePath:     path.Join(jobS3Path, "output", quantification.OutputCSVName),
					ApplyNodeIndex: jobconfig.NodeIndexMethod_Both,
				},
			},
		},
		AssociatedScanId: createParams.ScanId,
		JobName:          createParams.Name,
		ElementList:      createParams.Elements,
		OutputTitle:      csvTitleRow,
		Combined:         combined,
		QuantByROI:       quantByROI,
		ROIs:             rois,
		RequestorUserId:  requestorUserId,
	}

	return jm.internalSubmitJob(jg, requestorSession)
}
