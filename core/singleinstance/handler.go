package singleinstance

import (
	"context"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Call this to ensure this is handled ONCE among all API instances. It requires a "jobId", but this can be
// any unique string attributed to the task being handled. It also requires an instanceId (of this running
// API instance).
// Internally this works by writing our instance ID to DB, waiting a bit, then reading it back. If it wasn't
// overwritten by another API instance, we are the handler, and handleCallback is called, otherwise
// we see the instance ID is not ours and we stop further processing.
func HandleOnce(jobId string, instanceId string, handleCallback func(string), db *mongo.Database, ts timestamper.ITimeStamper, logger logger.ILogger) error {
	logger.Infof("HandleOnce: called for instance %v to handle job %v", instanceId, jobId)

	// Upsert to DB
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobHandlersName)

	handler := &protos.JobHandlerDBItem{
		JobId:             jobId,
		HandlerInstanceId: instanceId,
		TimeStampUnixSec:  uint32(ts.GetTimeNowSec()),
	}

	opt := options.Update().SetUpsert(true)
	result, err := coll.UpdateByID(ctx, jobId, bson.D{{Key: "$set", Value: handler}}, opt)
	if err != nil {
		return err
	}

	if result.UpsertedCount == 0 && result.ModifiedCount == 0 {
		logger.Errorf("HandleOnce: Write to DB had unexpected result: %+v", result)
	}

	// Now we wait a little bit and read it back. If the instance ID matches ours, we handle it!
	time.AfterFunc(2*time.Second, func() {
		readResult := coll.FindOne(ctx, bson.M{"_id": jobId}, options.FindOne())
		if readResult.Err() != nil {
			logger.Errorf("HandleOnce: Failed to read back JobHandlerDBItem: %v", readResult.Err())
		} else {
			readHandler := protos.JobHandlerDBItem{}
			if err = readResult.Decode(&readHandler); err != nil {
				logger.Errorf("HandleOnce: Failed to decode back JobHandlerDBItem: %v", readResult.Err())
			} else {
				// Check if its our ID
				if readHandler.HandlerInstanceId == instanceId {
					logger.Infof("HandleOnce: %v chosen to handle job %v", instanceId, jobId)
					handleCallback(jobId)
				} else {
					logger.Infof("HandleOnce: %v NOT handling job %v", instanceId, jobId)
				}
			}
		}
	})
	return nil
}
