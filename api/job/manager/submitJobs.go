package jobmanager

import (
	"context"
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/piquant"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Submit function for each kind of job type we support
func (jm *JobManager) SubmitQuantJob(createParams *protos.QuantCreateParams, requestorUserSess *sessionuser.SessionUser) (*protos.JobStatus, error) {
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
	status, err := jm.internalSubmitQuantJob(createParams, requestorUserSess, prefix, jobType, jobCompletionMethod)
	if err != nil {
		jm.svcs.Log.Errorf("SubmitQuantJob error: %v", err)
	}
	return status, err
}

/*
	func (jm *JobManager) SubmitPythonJob(pythonScriptName string) error {
		// Call the internal one, log the resulting errors if any
		err := jm.internalSubmitPythonJob(pythonScriptName)
		if err != nil {
			jm.svcs.Log.Errorf("SubmitPythonJob error: %v", err)
		}
		return err
	}

	func (jm *JobManager) SubmitExpressionJob(scanId, quantId, expressionId string) error {
		// Call the internal one, log the resulting errors if any
		err := jm.internalSubmitExpressionJob(scanId, quantId, expressionId, roiId)
		if err != nil {
			jm.svcs.Log.Errorf("SubmitExpressionJob error: %v", err)
		}
		return err
	}
*/
func (jm *JobManager) internalSubmitQuantJob(createParams *protos.QuantCreateParams, requestorUserSess *sessionuser.SessionUser, idPrefix string, jobType protos.JobType, completeMethod string) (*protos.JobStatus, error) {
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

	csvTitleRow := fmt.Sprintf("PIQUANT version: %v DetectorConfig: %v", jm.svcs.Config.JobRunnerDockerImage /*piquantVersion.Version*/, createParams.DetectorConfig)

	userArgs := "-Fe,1"
	if len(createParams.Parameters) > 0 {
		userArgs = createParams.Parameters
	}
	extraArgs := strings.Split(fmt.Sprintf("%v -t,%v", userArgs, jm.svcs.Config.CoresPerNode), " ")
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
		DockerImage:      jm.svcs.Config.JobRunnerDockerImage,
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

	return jm.internalSubmitJob(jg)
}

/*
	func (jm *JobManager) uploadJobFile(localPath string, remoteBucket string, remotePath string) (job.JobFilePath, error) {
		result := job.JobFilePath{
			LocalPath:    localPath,
			RemoteBucket: remoteBucket,
			RemotePath:   remotePath,
		}

		jm.svcs.Log.Infof("Upload %v -> s3://%v/%v", localPath, remoteBucket, remotePath)
		bytes, err := os.ReadFile(localPath)
		if err != nil {
			return result, err
		}

		err = jm.svcs.FS.WriteObject(remoteBucket, remotePath, bytes)
		return result, err
	}

/*

	func (jm *JobManager) internalSubmitPythonJob(pythonScriptName string, requestorUserId string) error {
		jg := &JobGroupConfig{
			DockerImage: jm.svcs.Config.JobRunnerDockerImage,
			NodeCount: 1,
	    	NodeConfig:  job.JobConfig{
				RequiredFiles: []JobFilePath{
					{
						LocalPath: pythonScriptName
						RemoteBucket: jm.svcs.Config.DatasetsBucket,
						RemotePath: filepaths.GetJobDataPath(scanId, jg.JobGroupId, pythonScriptName),
					},
				},
				Command: "python",
				Args:    []string{pythonScriptName},
			},
			//AssociatedScanId
			JobName: pythonScriptName,
			//ElementList
			RequestorUserId: requestorUserId,
		}

		return jm.internalSubmitJob(jg)
	}

	func (jm *JobManager) internalSubmitExpressionJob(scanId, quantId, expressionId string) error {
		// Read and validate scan
		scanItem, err := scan.ReadScanItem(scanId, jm.svcs.MongoDB)
		if err != nil {
			return err
		}

		// Read and validate quantification
		filter := bson.M{"_id": quantId}
		opts := options.FindOne()
		quantResult := jm.svcs.MongoDB.Collection(dbCollections.QuantificationsName).FindOne(context.TODO(), filter, opts)

		if quantResult.Err() != nil {
			return quantResult.Err()
		}

		quant := &protos.QuantificationSummary{}
		err := quantResult.Decode(quant)
		if err != nil {
			return err
		}

		// Read and validate expression
		filter = bson.M{"_id": expressionId}
		exprResult := jm.svcs.MongoDB.Collection(dbCollections.ExpressionsName).FindOne(context.TODO(), filter, opts)

		if exprResult.Err() != nil {
			return exprResult.Err()
		}

		expr := &protos.DataExpression{}
		err := exprResult.Decode(expr)
		if err != nil {
			return err
		}

		// We only support Lua expressions here!
		if expr.SourceLanguage != "LUA" {
			return fmt.Errorf("Expression language %v not supported for cloud execution", expr.SourceLanguage)
		}

job

		// Read all modules associated with the expression
		sourceFiles := map[string]string{}
		sourceFiles["main.lua"] = expr.SourceCode

		for _, ref := range expr.ModuleReferences {
			refstr := ref.ModuleID + "_" + ref.Version

			mod, err := wsHelpers.GetModuleVersion(ref.ModuleID, ref.Version, jm.svcs.MongoDB)
			if err != nil {
				return err
			}

			sourceFiles[refstr + ".lua"] = mod.SourceCode
		}

		// Upload source files and make list of required files for job to execute
		requiredFiles := []JobFilePath{}

		for name, src := range sourceFiles {

			jm.svcs.RemoteFS.WriteObject(jm.svcs.Config.JobBucket, name, byte[](src))

			requiredFiles = append(requiredFiles, JobFilePath{
				LocalPath: name,
				RemoteBucket: jm.svcs.Config.JobBucket,
				RemotePath: filepaths.GetJobDataPath(scanId, jg.JobGroupId, name),
			})
		}

		jg := &JobGroupConfig{
			DockerImage: jm.svcs.Config.JobRunnerDockerImage,
			NodeCount: 1,
	    	NodeConfig:  job.JobConfig{
				RequiredFiles: requiredFiles,
				Command: "lua5.3",
				Args:    []string{},
				OutputFiles: []JobFilePath{
					{
						LocalPath: "stdout",
						RemoteBucket: jm.svcs.Config.PiquantJobsBucket,
						RemotePath: ,
					},
				},
			},
			AssociatedScanId: scanId,
			JobName: fmt.Sprintf("%v-%v", scanItem.Title, expr.Name),
			//ElementList
			RequestorUserId: requestorUserId
		}

		return jm.internalSubmitJob(jg)
	}
*/
func (jm *JobManager) internalSubmitJob(jg *jobconfig.JobGroupConfig) (*protos.JobStatus, error) {
	if len(jg.JobGroupId) <= 0 {
		return nil, errors.New("SubmitJob: JobGroupId not specified")
	}

	// Check other fields are valid
	if len(jg.AssociatedScanId) > 100 {
		return nil, errors.New("SubmitJob: AssociatedScanId too long")
	}

	if len(jg.DockerImage) <= 0 {
		jm.svcs.Log.Infof("WARNING: SubmitJob - DockerImage not specified, this will result in local job runners, recommended only for testing")
	}

	if len(jg.RequestorUserId) <= 0 {
		return nil, errors.New("SubmitJob: RequestorUserId not specified")
	}

	if jg.NodeCount <= 0 {
		return nil, errors.New("SubmitJob: NodeCount must be at least 1")
	}

	if len(jg.NodeConfig.Command) <= 0 {
		return nil, errors.New("SubmitJob: Command not specified")
	}

	// Write job as JSON to S3 jobs bucket
	jobsPath := filepaths.GetJobDataPath(jg.AssociatedScanId, jg.JobGroupId, quantification.JobParamsFileName)
	err := jm.svcs.FS.WriteJSON(jm.svcs.Config.PiquantJobsBucket, jobsPath, jg)
	if err != nil {
		return nil, err
	}

	// Now that we've uploaded the job params, include it in the list of files the job can download
	jg.NodeConfig.RequiredFiles = append(jg.NodeConfig.RequiredFiles, jobconfig.JobFilePath{
		LocalPath:    quantification.JobParamsFileName,
		RemoteBucket: jm.svcs.Config.PiquantJobsBucket,
		RemotePath:   jobsPath,
	})

	// Write starting job status to mongo
	now := uint32(jm.svcs.TimeStamper.GetTimeNowSec())

	// To emulate "legacy" jobs, so we're backwards compatible - we retrieve the name, element list and
	// import jobs have the jobItemId set to the scan id
	// TODO: Probably can be removed as this is kind of not a real job type that we execute?!
	itemId := jg.JobGroupId
	if jg.JobType == protos.JobType_JT_IMPORT_SCAN || jg.JobType == protos.JobType_JT_REIMPORT_SCAN {
		itemId = jg.AssociatedScanId
	}

	job := &protos.JobStatus{
		JobId:            jg.JobGroupId,
		Status:           protos.JobStatus_STARTING,
		StartUnixTimeSec: now,
		OtherLogFiles:    []string{},
		JobType:          jg.JobType,
		JobItemId:        itemId,
		Name:             jg.JobName,
		Elements:         jg.ElementList,
		RequestorUserId:  jg.RequestorUserId,
	}

	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobStatusName)
	result, err := coll.InsertOne(ctx, job, options.InsertOne())
	if err != nil {
		return job, err
	}

	if result.InsertedID != jg.JobGroupId {
		return job, fmt.Errorf("Inserted job stats for %v doesn't match db id %v", jg.JobGroupId, result.InsertedID)
	}

	// Also write out the job config we're running to the jobs table. That doesn't get updated as status changes, it's more a record of what we started with
	coll = jm.svcs.MongoDB.Collection(dbCollections.JobsName)
	result, err = coll.InsertOne(ctx, jg, options.InsertOne())
	if err != nil {
		return job, err
	}

	if result.InsertedID != jg.JobGroupId {
		return job, fmt.Errorf("Inserted job %v doesn't match db id %v", jg.JobGroupId, result.InsertedID)
	}

	// Queue up each individual job so it can run on a node. Job queue will eventually be empty and the job completes
	return job, jm.QueueJob(jg)
}

/*
func makeJobIdAndType(jg *JobGroupConfig, idg idgen.IDGenerator) (string, protos.JobType) {
	prefix := "job-"
	jobType := protos.JobType_JT_UNKNOWN

	cmd := strings.ToLower(jg.NodeConfig.Command)
	if strings.Contains(cmd, "lua") {
		prefix = "expr-"
	} else if strings.Contains(cmd, "python") {
		prefix = "python-"
	} else if strings.Contains(cmd, "piquant") {
		// Creating a new quantification command is actually "map"
		if jg.NodeConfig.Args[0] == "map" {
			prefix = "quant-"
			jobType = protos.JobType_JT_RUN_QUANT
		} else {
			// Fit command is actually "quant"
			if jg.NodeConfig.Args[0] == "quant" {
				jobType = protos.JobType_JT_RUN_FIT
			}
			prefix = "piquant-" + jg.NodeConfig.Args[0] + "-"
		}
	}

	return fmt.Sprintf("%v-%v", prefix, idg.GenObjectID()), jobType
}
*/
