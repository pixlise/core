package jobmanager

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

// Updates job status in DB and sends out notification to listening clients
func (jm *JobManager) updateJobStatus(jobId string, status protos.JobStatus_Status, message string, updateClient bool) error {
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobStatusName)

	updResult, err := coll.UpdateByID(ctx, jobId, bson.D{{Key: "$set", Value: bson.M{
		"status":                status,
		"message":               message,
		"lastupdateunixtimesec": uint32(jm.svcs.TimeStamper.GetTimeNowSec()),
	}}})

	if err != nil {
		return fmt.Errorf("Failed to update existing job status %v to: %v. Error: %v", jobId, status, err)
	}

	if updResult.MatchedCount != 1 && updResult.UpsertedCount != 1 {
		jm.svcs.Log.Errorf("updateJobStatus result had unexpected counts %+v id: %v", updResult, jobId)
	} else {
		// If we're in local mode (for testing), we don't show the PREPARING_NODES log because this breaks tests due to
		// its asynchronous nature, the order where it get logged is non-deterministic
		if !jm.isLocalTestMode() || status != protos.JobStatus_PREPARING_NODES {
			jm.svcs.Log.Infof("updateJobStatus: %v with status %v, message: %v", jobId, protos.JobStatus_Status_name[int32(status.Number())], message)
		}
	}

	// Send out notifications so client knows the job state has changed
	if updateClient {
		dbStatus, err := jm.readJobStatus(jobId)
		if err != nil {
			jm.svcs.Log.Errorf("updateJobStatus failed to read job status for %v while sending to client. Error: %v", jobId, err)
		} else if len(dbStatus.RequestorUserId) > 0 && dbStatus.RequestorUserId != sessionuser.PIXLISESystemUserId {
			if sess, ok := jm.userSessionLookup[dbStatus.RequestorUserId]; ok && sess != nil {
				wsUpd := protos.WSMessage{
					Contents: &protos.WSMessage_QuantCreateUpd{
						QuantCreateUpd: &protos.QuantCreateUpd{
							Status: dbStatus,
						},
					},
				}

				wsHelpers.SendForSession(sess, &wsUpd)
			}
		}
	}

	return nil
}

func (jm *JobManager) readJobStatus(jobId string) (*protos.JobStatus, error) {
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobStatusName)

	dbStatusResult := coll.FindOne(ctx, bson.M{"_id": jobId})
	if dbStatusResult.Err() != nil {
		return nil, dbStatusResult.Err()
	}

	dbStatus := &protos.JobStatus{}
	err := dbStatusResult.Decode(&dbStatus)
	return dbStatus, err
}
