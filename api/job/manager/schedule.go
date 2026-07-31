package jobmanager

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (jm *JobManager) ListScheduledJobs() ([]*protos.ScheduledJob, error) {
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobScheduleName)
	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("Failed to list scheduled jobs: %v", err)
	}

	result := []*protos.ScheduledJob{}
	err = cursor.All(ctx, &result)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode scheduled job list: %v", err)
	}

	return result, nil
}

var SCAN_ID_AUTO_IMPORTED = "imported"
var QUANT_BY_NAME_PREFIX = "name:"
var QUANT_BY_ID_PREFIX = "id:"

func validateJob(job *protos.ScheduledJob) error {
	if len(job.JobParameters) <= 0 {
		return errors.New("JobParameters must be set")
	}

	checkParamsSet := []string{"scanId", "expressionId", "quant"}
	for _, check := range checkParamsSet {
		if len(job.JobParameters[check]) <= 0 {
			return fmt.Errorf("JobParameters[%v] must be set", check)
		}
	}

	if job.ScheduleType == protos.ScheduledJob_AFTER_IMPORT {
		if job.ScheduledFirstTimeUnixSec > 0 || job.IntervalSec > 0 {
			return errors.New("AFTER_IMPORT jobs must not have time fields set")
		}

		// Check that scan id is set to use the imported one
		if job.JobParameters["scanId"] != SCAN_ID_AUTO_IMPORTED {
			return fmt.Errorf("JobParameters[scanId] expected to be %v, set to %v", SCAN_ID_AUTO_IMPORTED, job.JobParameters["scanId"])
		}
	} else if job.ScheduleType == protos.ScheduledJob_TIME_BASED {
		if job.JobOrder != 0 {
			return errors.New("TIME_BASED jobs must not have order set")
		}
	} else if job.ScheduleType != protos.ScheduledJob_AFTER_IMPORT && job.ScheduleType != protos.ScheduledJob_TIME_BASED {
		return errors.New("ScheduleType must be AFTER_IMPORT or TIME_BASED")
	}

	return nil
}

func (jm *JobManager) SetScheduledJob(job *protos.ScheduledJob) (*protos.ScheduledJob, error) {
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobScheduleName)

	// Check it's valid
	if job.JobType != protos.JobType_JT_RUN_EXPRESSION { //&& job.JobType != protos.JobType_JT_RUN_PYTHON &&
		return nil, errors.New("JobType must be JT_RUN_EXPRESSION")
	}

	if err := validateJob(job); err != nil {
		return nil, err
	}

	if len(job.Id) <= 0 {
		// We're doing an insert
		// Generate an ID
		job.Id = fmt.Sprintf("sched-job-%v", jm.svcs.IDGen.GenObjectID())
		result, err := coll.InsertOne(ctx, job)
		if err != nil {
			return nil, fmt.Errorf("Failed to insert scheduled job \"%v\": %v", job.Id, err)
		}
		if result.InsertedID != job.Id {
			jm.svcs.Log.Errorf("SetScheduledJob: Inserted job id %v doesn't match specified %v", result.InsertedID, job.Id)
		}
	} else {
		// We're doing an update
		opt := options.Update() //.SetUpsert(true)

		result, err := coll.UpdateOne(ctx, bson.D{{Key: "_id", Value: job.Id}}, bson.D{{Key: "$set", Value: job}}, opt)
		if err != nil {
			return nil, fmt.Errorf("Failed to set scheduled job \"%v\": %v", job.Id, err)
		}

		if result.MatchedCount == 0 && result.ModifiedCount == 0 && result.UpsertedCount == 0 {
			return nil, fmt.Errorf("Failed to update scheduled job \"%v\": Not found", job.Id)
		}

		if result.ModifiedCount != 1 && result.UpsertedCount != 1 {
			jm.svcs.Log.Infof("SetScheduledJob: Unexpected result %+v", result)
		}
	}

	return job, nil
}

func (jm *JobManager) DeleteScheduledJob(id string) error {
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobScheduleName)
	result, err := coll.DeleteOne(context.TODO(), bson.M{"_id": id})
	if err != nil {
		return fmt.Errorf("Failed to delete scheduled job: %v. Error: %v", id, err)
	}

	if result.DeletedCount == 0 {
		return fmt.Errorf("Failed to delete scheduled job \"%v\": Not found", id)
	}

	return nil
}

/*
func HandleScanWriteJobReq(req *protos.ScanWriteJobReq, hctx wsHelpers.HandlerContext) (*protos.ScanWriteJobResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.JobsName)
	result, err := coll.UpdateByID(ctx, req.Job.JobGroupId, bson.D{{Key: "$set", Value: req.Job}}, options.Update().SetUpsert(true))
	if err != nil {
		return nil, err
	}

	if result.UpsertedCount == 0 && result.ModifiedCount == 0 {
		hctx.Svcs.Log.Errorf("Unexpected update result for job: %+v", result)
	}

	return &protos.ScanWriteJobResp{}, nil
}*/
