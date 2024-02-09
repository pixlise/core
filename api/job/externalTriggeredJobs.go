package job

import (
	"context"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

// This is here to monitor externally triggered jobs. The rest of the job code expects AddJob to be called within the
// API and then we start a thread to listen to those jobs for their duration. Here we also trigger a thread to listen
// to job updates, but only care about IDs with a special prefix marking them as externally triggered.
// An example of this is a data import via OCS (ie, data from NASA JPL) - these jobs are triggered via SNS
// and here we have code for our multiple API instances to listen for these events, pick a single API instance to
// handle it and send out notifications as needed to clients

func ListenForExternalTriggeredJobs(prefix string, callback func(*protos.JobStatus), db *mongo.Database, logger logger.ILogger) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.JobStatusName)

	stream, err := coll.Watch(ctx, mongo.Pipeline{})
	if err != nil {
		logger.Errorf("Failed to watch job statuses prefixed by: %v, no notifications will be sent. Error: %v", prefix, err)
		return
	}

	logger.Infof("Listening for externally triggered scan import jobs...")
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
			logger.Errorf("Failed to decode change stream for job status while watching for job statuses prefixed by: %v", prefix)
			continue
		}

		// Check if we're interested
		if item.FullDocument != nil && strings.HasPrefix(item.DocumentKey.Id, prefix) {
			// Notify our listener that this event happened
			logger.Infof("Detected externally triggered scan import: %v", item.DocumentKey.Id)
			callback(item.FullDocument)
		}
	}
}
