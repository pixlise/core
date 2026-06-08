package job

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ReadJobQueue(db *mongo.Database) (map[string][]*protos.JobQueueItem, error) {
	groupsAndJobs := map[string][]*protos.JobQueueItem{}

	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobQueueName)

	cursor, err := coll.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		return groupsAndJobs, err
	}

	queuedJobs := []*protos.JobQueueItem{}
	err = cursor.All(context.TODO(), &queuedJobs)
	if err != nil {
		return groupsAndJobs, err
	}

	// Early out, we found no existing jobs to worry about
	if len(queuedJobs) <= 0 {
		return groupsAndJobs, nil
	}

	// Build a list of all jobs as they belong to a job group
	for _, queuedJob := range queuedJobs {
		// Ensure group exists
		if _, ok := groupsAndJobs[queuedJob.JobGroupId]; !ok {
			groupsAndJobs[queuedJob.JobGroupId] = []*protos.JobQueueItem{}
		}

		// Add it
		groupsAndJobs[queuedJob.JobGroupId] = append(groupsAndJobs[queuedJob.JobGroupId], queuedJob)
	}

	return groupsAndJobs, nil
}

// Listens to the job queue collection. If the collection is dropped, it returns true signifying it can be retried
// but if it ends for any other reason, it will return false
func ListenToJobQueue(allowedOps []string, db *mongo.Database, ts timestamper.ITimeStamper, log logger.ILogger, rateLimitSec uint, runCheck func(*protos.JobQueueItem)) bool {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobQueueName)

	stream, err := coll.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		log.Errorf("Failed to watch job queue. Error: %v", err)
		return false
	}

	log.Infof("Listening for queued jobs...")
	lastOpWasInvalidate := false
	lastTimeSec := uint(0)

	for stream.Next(ctx) {
		now := uint(ts.GetTimeNowSec())

		if now-lastTimeSec <= rateLimitSec {
			log.Debugf("Rate limiting DB job queue change notification")
			continue
		}

		// Work out if we're interested at all
		operation, _ /*key*/, doc, err := ReadChangeStreamItem[*protos.JobQueueItem](stream)

		if err != nil {
			log.Errorf("Failed to decode change stream for job queue")
			continue
		}

		// Check if we're interested
		if utils.ItemInSlice(operation, allowedOps) {
			// NOTE: It appears Mongo doesn't give us the right keys! If we do an InsertMany for 6 items, we
			//       get the item.DocumentKey 6x here.
			// 		 Instead, we just treat this as a hint to look at the job queue. We take actions based on
			//       what we find in the queue

			// NOTE: if we update multiple documents at once, this gives them back one-by-one. To avoid
			// processing the same thing many times, here we wait for the rate limit to time out and notify
			// the listener at the end, so any DB reads affect the state post all the writes
			/*go func() {
				time.Sleep(time.Duration(rateLimitSec) * time.Second)
				runCheck(doc)
			}()*/
			runCheck(doc)

			lastTimeSec = now
		}

		lastOpWasInvalidate = operation == "invalidate"
	}

	return lastOpWasInvalidate
}

// Updates the job queue item and corresponding job group item if needed
func UpdateJobQueueItem(jobId string, state protos.JobQueueItem_State, message string, jobGroupId string, instanceId string, db *mongo.Database, ts timestamper.ITimeStamper) error {
	nowUnixSec := ts.GetTimeNowSec()
	ctx := context.TODO()

	dbResult, err := db.Collection(dbCollections.JobQueueName).UpdateByID(ctx, jobId, bson.D{{Key: "$set", Value: bson.M{
		"state":                       state,
		"message":                     message,
		"lastupdatedtimestampunixsec": nowUnixSec,
	}}})

	if err != nil {
		return err
	}

	if dbResult.ModifiedCount != 1 {
		return fmt.Errorf("UpdateJobQueueItem: Expected modified count of 1, got %v", dbResult.ModifiedCount)
	}
	return nil
}
