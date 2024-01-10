package job

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/idgen"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/timestamper"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var jobUpdateIntervalSec = uint32(10)
var activeJobs = map[string]bool{}
var activeJobLock = sync.Mutex{}

// Expected to be called by API to create the initial record of a job. It can then trigger it however it needs to
// (eg AWS lambda or running PIQUANT nodes) and this sticks around monitoring the DB entry for changes, calling
// the sendUpdate callback function on change. Returns the snapshot of the "added" job that was saved
func AddJob(idPrefix string, jobTimeoutSec uint32, db *mongo.Database, idgen idgen.IDGenerator, ts timestamper.ITimeStamper, logger logger.ILogger, sendUpdate func(*protos.JobStatus)) (*protos.JobStatus, error) {
	// Generate a new job Id that this job will write to
	// which we also return to the caller, so they can track what happens
	// with this async task
	jobId := fmt.Sprintf("%v-%s", idPrefix, idgen.GenObjectID())
	now := uint32(ts.GetTimeNowSec())

	job := &protos.JobStatus{
		JobId:            jobId,
		Status:           protos.JobStatus_STARTING,
		StartUnixTimeSec: now,
		OtherLogFiles:    []string{},
	}

	if _, ok := activeJobs[jobId]; ok {
		return job, errors.New("Job already exists: " + jobId)
	}

	watchUntilUnixSec := now + jobTimeoutSec

	// Add to DB
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)
	result, err := coll.InsertOne(ctx, job, options.InsertOne())
	if err != nil {
		return job, err
	}

	if result.InsertedID != jobId {
		logger.Errorf("Inserted job %v doesn't match db id %v", jobId, result.InsertedID)
	}

	// We'll watch this one and send out updates
	activeJobs[jobId] = true

	// Start a thread to watch this job
	go watchJob(jobId, watchUntilUnixSec, db, logger, ts, sendUpdate)

	logger.Infof("AddJob: %v", jobId)
	return job, nil
}

// Expected to be called from the thing running the job. This updates the DB status, which hopefully the go thread started by
// AddJob will notice and fire off an update
func UpdateJob(jobId string, status protos.JobStatus_Status, message string, logId string, db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) error {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	filter := bson.D{{"_id", jobId}}
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

	filter := bson.D{{"_id", jobId}}
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
	activeJobs[jobId] = false
	return nil
}

func watchJob(jobId string, watchUntilUnixSec uint32, db *mongo.Database, logger logger.ILogger, ts timestamper.ITimeStamper, sendUpdate func(*protos.JobStatus)) {
	logger.Infof(">> Start watching job: %v...", jobId)

	// NOTE: we subscribe for changes to the jobs collection in Mongo and if we see a change for
	// the job we're watching, we can send notifications out. We only listen for a certain amount of
	// time after which we assume the job has timed out
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	stream, err := coll.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		logger.Errorf("Failed to watch job status: %v, no notifications will be sent. Error: %v", jobId, err)
		return
	}

	for stream.Next(ctx) {
		// A status has changed! Check if it's ours and process it
		// otherwise check if we've timed out
		type ChangeStreamId struct {
			Id string `bson:"_id"`
		}
		type ChangeStreamItem struct {
			OperationType string            `bson:"operationType"`
			DocumentKey   ChangeStreamId    `bson:"documentKey"`
			FullDocument  *protos.JobStatus `bson:"fullDocument"`
		}

		item := ChangeStreamItem{}
		err = stream.Decode(&item)
		if err != nil {
			logger.Errorf("Failed to decode change stream for job status while watching for job: %v", jobId)
			continue
		}

		// Check if we're interested
		if item.FullDocument != nil && item.DocumentKey.Id == jobId {
			// Send an update
			sendUpdate(item.FullDocument)

			// If job has completed, stop here
			if item.FullDocument.Status == protos.JobStatus_COMPLETE || item.FullDocument.Status == protos.JobStatus_ERROR {
				break
			}
		} else {
			// Not one of ours, but check if we've timed out
			now := ts.GetTimeNowSec()
			if now > int64(watchUntilUnixSec) {
				// We've timed out
				sendUpdate(&protos.JobStatus{
					JobId:          jobId,
					Status:         protos.JobStatus_ERROR,
					Message:        "Timed out while waiting for status update",
					EndUnixTimeSec: uint32(ts.GetTimeNowSec()),
					OutputFilePath: "",
					OtherLogFiles:  []string{},
				})

				break
			}
		}
	}

	defer activeJobLock.Unlock()
	activeJobLock.Lock()
	activeJobs[jobId] = false
	logger.Infof(">> Finish watching job: %v...", jobId)
}

func readJobStatus(jobId string, coll *mongo.Collection) (*protos.JobStatus, error) {
	dbStatusResult := coll.FindOne(context.TODO(), bson.M{"_id": jobId})
	if dbStatusResult.Err() != nil {
		return nil, dbStatusResult.Err()
	}

	dbStatus := &protos.JobStatus{}
	err := dbStatusResult.Decode(&dbStatus)
	return dbStatus, err
}
