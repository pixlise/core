package job

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var activeJobs = map[string]bool{}
var activeJobLock = sync.Mutex{}

// Expected to be called by API to create the initial record of a job. It can then trigger it however it needs to
// (eg AWS lambda or running PIQUANT nodes) and this sticks around monitoring the DB entry for changes, calling
// the sendUpdate callback function on change. Returns the snapshot of the "added" job that was saved

func AddJob(
	idPrefix string,
	requestorUserId string,
	jobType protos.JobStatus_JobType,
	jobItemId string,
	jobName string,
	elementList []string, // optional, only set if it's a quant!
	jobTimeoutSec uint32,
	db *mongo.Database,
	idgen idgen.IDGenerator,
	ts timestamper.ITimeStamper,
	logger logger.ILogger,
	sendUpdate func(*protos.JobStatus)) (*protos.JobStatus, error) {
	// Generate a new job Id that this job will write to
	// which we also return to the caller, so they can track what happens
	// with this async task
	jobId := fmt.Sprintf("%v-%s", idPrefix, idgen.GenObjectID())
	now := uint32(ts.GetTimeNowSec())

	if len(jobItemId) <= 0 {
		jobItemId = jobId
	}

	job := &protos.JobStatus{
		JobId:            jobId,
		Status:           protos.JobStatus_STARTING,
		StartUnixTimeSec: now,
		OtherLogFiles:    []string{},
		JobType:          jobType,
		JobItemId:        jobItemId,
		Name:             jobName,
		Elements:         elementList,
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

	logger.Infof("AddJob: %v of type: %v working on item id: %v", jobId, jobType, jobItemId)
	return job, nil
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
