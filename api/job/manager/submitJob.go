package jobmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/sessionuser"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*
	func (jm *JobManager) SubmitPythonJob(pythonScriptName string) error {
		// Call the internal one, log the resulting errors if any
		err := jm.internalSubmitPythonJob(pythonScriptName)
		if err != nil {
			jm.svcs.Log.Errorf("SubmitPythonJob error: %v", err)
		}
		return err
	}
*/

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
*/

func (jm *JobManager) internalSubmitJob(jg *jobconfig.JobGroupConfig, requestorSession *melody.Session) (*protos.JobStatus, error) {
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

	// Store session for completion-side use if we're sending notifications
	if len(jg.RequestorUserId) > 0 && jg.RequestorUserId != sessionuser.PIXLISESystemUserId && requestorSession != nil {
		jm.userSessionLookup[jg.RequestorUserId] = requestorSession
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
		JobId: jg.JobGroupId,
		// For backwards compatibility with old quants... so tests pass. Hopefully not needed in future!
		//LogId:            jg.JobGroupId,
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
