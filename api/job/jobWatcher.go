package job

import (
	"context"
	"errors"
	"fmt"
	"time"

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
	go watchJob(jobId, now, watchUntilUnixSec, db, logger, ts, sendUpdate)

	return job, nil
}

// Expected to be called from the thing running the job. This updates the DB status, which hopefully the go thread started by
// AddJob will notice and fire off an update
func UpdateJob(jobId string, status protos.JobStatus_Status, message string, logId string, db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) error {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	filter := bson.D{{"_id", jobId}}
	opt := options.Update()

	data := bson.D{
		{"$set", bson.D{
			{"status", status},
			{"message", message},
			{"logid", logId},
			{"lastupdateunixtimesec", uint32(ts.GetTimeNowSec())},
		}},
	}

	result, err := coll.UpdateOne(ctx, filter, data, opt)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 && result.UpsertedCount != 1 {
		logger.Errorf("UpdateJob result had unexpected counts %+v id: %v", result, jobId)
	}

	return nil
}

// Expected to be called from the thing running the job. This allows setting some output fields
func CompleteJob(jobId string, success bool, message string, outputFilePath string, otherLogFiles []string, db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) error {
	status := protos.JobStatus_COMPLETE
	if !success {
		status = protos.JobStatus_ERROR
	}

	now := uint32(ts.GetTimeNowSec())

	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	filter := bson.D{{"_id", jobId}}
	opt := options.Update()

	data := bson.D{
		{"$set", bson.D{
			{"status", status},
			{"message", message},
			{"lastupdateunixtimesec", now},
			{"endunixtimesec", now},
			{"outputfilepath", outputFilePath},
			{"otherlogfiles", otherLogFiles},
		}},
	}

	result, err := coll.UpdateOne(ctx, filter, data, opt)
	if err != nil {
		return err
	}

	if result.MatchedCount != 1 && result.UpsertedCount != 1 {
		logger.Errorf("CompleteJob result had unexpected counts %+v id: %v", result, jobId)
	}

	activeJobs[jobId] = false
	return nil
}

func watchJob(jobId string, nowUnixSec uint32, watchUntilUnixSec uint32, db *mongo.Database, logger logger.ILogger, ts timestamper.ITimeStamper, sendUpdate func(*protos.JobStatus)) {
	logger.Infof(">> Start watching job: %v...", jobId)

	// Check the DB for updates periodically until watchUntilUnixSec at which point if the job isn't
	// complete we can assume it died/timed-out/whatever
	// Firstly, lets work out how many updates we'll need to send
	maxUpdates := (watchUntilUnixSec - nowUnixSec) / jobUpdateIntervalSec

	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	// Run that many times
	lastUpdateUnixSec := uint32(0)

	for c := uint32(0); c < maxUpdates; c++ {
		time.Sleep(time.Duration(jobUpdateIntervalSec) * time.Second)
		logger.Infof(">> Checking watched job: %v...", jobId)

		filter := bson.D{{"_id", jobId}}
		opt := options.FindOne()

		dbStatusResult := coll.FindOne(ctx, filter, opt)
		if dbStatusResult.Err() != nil {
			logger.Errorf("Failed to find DB entry for job status: %v", jobId)
		} else {
			// Check if update field differs
			dbStatus := protos.JobStatus{}
			err := dbStatusResult.Decode(&dbStatus)
			if err != nil {
				logger.Errorf("Failed to decode DB entry for job status: %v", jobId)
			} else if lastUpdateUnixSec != dbStatus.LastUpdateUnixTimeSec {
				logger.Infof(">> Update sent for watched job: %v...", jobId)

				// OK we have a status update! Send
				sendUpdate(&dbStatus)

				// If the job is finished, stop here
				if dbStatus.Status == protos.JobStatus_COMPLETE || dbStatus.Status == protos.JobStatus_ERROR {
					logger.Infof(">> Stop watching completed job: %v, status: %v", jobId, dbStatus.Status)
					return
				}

				// Remember the new update time
				lastUpdateUnixSec = dbStatus.LastUpdateUnixTimeSec
			}
		}
	}

	logger.Errorf(">> Stop watching TIMED-OUT job: %v", jobId)
	sendUpdate(&protos.JobStatus{
		JobId:          jobId,
		Status:         protos.JobStatus_ERROR,
		Message:        "Timed out while waiting for status",
		EndUnixTimeSec: uint32(ts.GetTimeNowSec()),
		OutputFilePath: "",
		OtherLogFiles:  []string{},
	})
	activeJobs[jobId] = false
}
