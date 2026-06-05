package jobmanager

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (jm *JobManager) updateJobStatus(jobId string, status protos.JobStatus_Status, message string, logId string, existingStatus *protos.JobStatus) (*protos.JobStatus, error) {
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobStatusName)

	filter := bson.D{{Key: "_id", Value: jobId}}
	opt := options.Replace()

	jobStatus := &protos.JobStatus{
		JobId:                 jobId,
		Status:                status,
		Message:               message,
		LogId:                 logId,
		LastUpdateUnixTimeSec: uint32(jm.svcs.TimeStamper.GetTimeNowSec()),
	}

	var err error
	if existingStatus == nil {
		// Only read it if it wasn't passed in
		existingStatus, err = jm.readJobStatus(jobId)
	}

	if err != nil {
		jm.svcs.Log.Errorf("Failed to read existing job status when writing updateJobStatus %v: %v", jobId, err)
	} else {
		jobStatus.StartUnixTimeSec = existingStatus.StartUnixTimeSec
		jobStatus.JobType = existingStatus.JobType
		jobStatus.JobItemId = existingStatus.JobItemId
		jobStatus.RequestorUserId = existingStatus.RequestorUserId
		jobStatus.Name = existingStatus.Name
		jobStatus.Elements = existingStatus.Elements

		if len(logId) <= 0 {
			jobStatus.LogId = existingStatus.LogId
		}
	}

	replaceResult, err := coll.ReplaceOne(ctx, filter, jobStatus, opt)
	if err != nil {
		jm.svcs.Log.Errorf("updateJobStatus %v: %v", jobId, err)
		return jobStatus, err
	}

	if replaceResult.MatchedCount != 1 && replaceResult.UpsertedCount != 1 {
		jm.svcs.Log.Errorf("updateJobStatus result had unexpected counts %+v id: %v", replaceResult, jobId)
	} else {
		jm.svcs.Log.Infof("updateJobStatus: %v with status %v, message: %v", jobId, protos.JobStatus_Status_name[int32(status.Number())], message)
	}

	// Send out notifications so client knows the job state has changed
	if len(existingStatus.RequestorUserId) > 0 && existingStatus.RequestorUserId != sessionuser.PIXLISESystemUserId {
		if sess, ok := jm.userSessionLookup[existingStatus.RequestorUserId]; ok && sess != nil {
			wsUpd := protos.WSMessage{
				Contents: &protos.WSMessage_QuantCreateUpd{
					QuantCreateUpd: &protos.QuantCreateUpd{
						Status: jobStatus,
					},
				},
			}

			wsHelpers.SendForSession(sess, &wsUpd)
		}
	}

	return jobStatus, nil
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
