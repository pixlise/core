package job

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Expected to be called from the thing running the job. This updates the DB status, which hopefully the go thread started by
// AddJob will notice and fire off an update
func UpdateJob(jobId string, status protos.JobStatus_Status, message string, logId string, db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) error {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	filter := bson.D{{Key: "_id", Value: jobId}}
	opt := options.Replace()

	jobStatus := &protos.JobStatus{
		JobId:                 jobId,
		Status:                status,
		Message:               message,
		LogId:                 logId,
		LastUpdateUnixTimeSec: uint32(ts.GetTimeNowSec()),
	}

	existingStatus, err := readJobStatus(jobId, coll)
	if err != nil {
		logger.Errorf("Failed to read existing job status when writing UpdateJob %v: %v", jobId, err)
	} else {
		jobStatus.StartUnixTimeSec = existingStatus.StartUnixTimeSec
		jobStatus.JobType = existingStatus.JobType
		jobStatus.JobItemId = existingStatus.JobItemId
		jobStatus.RequestorUserId = existingStatus.RequestorUserId
		jobStatus.Name = existingStatus.Name
		jobStatus.Elements = existingStatus.Elements
	}

	replaceResult, err := coll.ReplaceOne(ctx, filter, jobStatus, opt)
	if err != nil {
		logger.Errorf("UpdateJob %v: %v", jobId, err)
		return err
	}

	if replaceResult.MatchedCount != 1 && replaceResult.UpsertedCount != 1 {
		logger.Errorf("UpdateJob result had unexpected counts %+v id: %v", replaceResult, jobId)
	} else {
		logger.Infof("UpdateJob: %v with status %v, message: %v", jobId, protos.JobStatus_Status_name[int32(status.Number())], message)
	}

	return nil
}

// Expected to be called from the thing running the job. This allows setting some output fields
func CompleteJob(jobId string, success bool, message string, outputFilePath string, otherLogFiles []string, db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) error {
	status := protos.JobStatus_COMPLETE
	if !success {
		status = protos.JobStatus_ERROR
	}

	logger.Infof("Job: %v completed with status: %v, message: %v", jobId, status.String(), message)

	now := uint32(ts.GetTimeNowSec())

	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	filter := bson.D{{Key: "_id", Value: jobId}}
	opt := options.Replace()

	jobStatus := &protos.JobStatus{
		JobId:                 jobId,
		Status:                status,
		Message:               message,
		LogId:                 "",
		StartUnixTimeSec:      0,
		LastUpdateUnixTimeSec: now,
		EndUnixTimeSec:        now,
		OutputFilePath:        outputFilePath,
		OtherLogFiles:         otherLogFiles,
	}

	existingStatus, err := readJobStatus(jobId, coll)
	if err != nil {
		logger.Errorf("Failed to read existing job status when writing CompleteJob %v: %v", jobId, err)
	} else {
		jobStatus.LogId = existingStatus.LogId
		jobStatus.StartUnixTimeSec = existingStatus.StartUnixTimeSec
		jobStatus.JobType = existingStatus.JobType
		jobStatus.JobItemId = existingStatus.JobItemId
		jobStatus.RequestorUserId = existingStatus.RequestorUserId
		jobStatus.Name = existingStatus.Name
		jobStatus.Elements = existingStatus.Elements
	}

	replaceResult, err := coll.ReplaceOne(ctx, filter, jobStatus, opt)
	if err != nil {
		logger.Errorf("CompleteJob %v: %v", jobId, err)
		return err
	}

	if replaceResult.MatchedCount != 1 && replaceResult.UpsertedCount != 1 {
		logger.Errorf("CompleteJob result had unexpected counts %+v id: %v", replaceResult, jobId)
	} else {
		logger.Infof("CompleteJob: %v with status %v, message: %v", jobId, protos.JobStatus_Status_name[int32(status.Number())], message)
	}

	defer activeJobLock.Unlock()
	activeJobLock.Lock()

	// Only update the job status if we have an entry for this job
	// HINT: If we don't this code may be running in say a Lambda function and
	//       not a part of the API instance, so nothing in our memory space cares
	//       about the state of this job, we're just notifying out via the DB above!
	if _, ok := activeJobs[jobId]; ok {
		activeJobs[jobId] = false
	}
	return nil
}
