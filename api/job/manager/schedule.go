package jobmanager

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/utils"
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

func (jm *JobManager) GetScheduledJob(scheduledJobId string) (*protos.ScheduledJob, error) {
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobScheduleName)
	result := coll.FindOne(ctx, bson.M{"_id": scheduledJobId})
	if result.Err() != nil {
		return nil, fmt.Errorf("Failed to get scheduled job %v: %v", scheduledJobId, result.Err())
	}

	schedJob := &protos.ScheduledJob{}
	if err := result.Decode(schedJob); err != nil {
		return nil, fmt.Errorf("Failed to decode scheduled job %v: %v", scheduledJobId, err)
	}

	return schedJob, nil
}

var SCAN_ID_AUTO_IMPORTED = "imported"
var SCAN_ID_NONE = "none"
var QUANT_BY_NAME_PREFIX = "name:"
var QUANT_BY_ID_PREFIX = "id:"
var ELEMS_BY_SET = "set:"
var ELEMS_BY_LIST = "list:"

func validateJob(job *protos.ScheduledJob) error {
	validJobTypeParams := map[protos.JobType]map[string][]string{
		protos.JobType_JT_RUN_EXPRESSION: {
			"scanId":       {},
			"expressionId": {},
			"quant":        {QUANT_BY_NAME_PREFIX, QUANT_BY_ID_PREFIX},
		},
		protos.JobType_JT_RUN_QUANT: {
			"scanId":     {},
			"elements":   {ELEMS_BY_SET, ELEMS_BY_LIST},
			"quantName":  {},
			"configName": {},
			"quantMode":  {},
		},
		protos.JobType_JT_RUN_PYTHON_SCRIPT: {
			"scanId":       {},
			"quant":        {QUANT_BY_NAME_PREFIX, QUANT_BY_ID_PREFIX},
			"repositoryId": {},
			"scriptName":   {},
		},
	}

	acceptableJobs := utils.GetMapKeys(validJobTypeParams)

	if !utils.ItemInSlice(job.JobType, acceptableJobs) {
		js := []string{}
		for _, j := range acceptableJobs {
			js = append(js, j.String())
		}
		return fmt.Errorf("JobType must be one of: %v", strings.Join(js, ","))
	}

	if len(job.JobParameters) <= 0 {
		return errors.New("JobParameters must be set")
	}

	paramsAndPrefixes := validJobTypeParams[job.JobType]
	requiredParams := utils.GetMapKeys(paramsAndPrefixes)

	for _, check := range requiredParams {
		paramGiven := job.JobParameters[check]
		if len(paramGiven) <= 0 {
			return fmt.Errorf("JobParameters[%v] must be set", check)
		}

		// If this param has prefixes, make sure theyre provided
		prefixes := paramsAndPrefixes[check]
		if len(prefixes) > 0 {
			prefixOK := false
			for _, prefix := range prefixes {
				if strings.HasPrefix(paramGiven, prefix) {
					// We have the prefix, just make sure the rest is non-empty
					if len(paramGiven) <= len(prefix) {
						return fmt.Errorf("JobParameters[%v] missing value after prefix: %v", check, prefix)
					}

					prefixOK = true
					break
				}
			}

			if !prefixOK && paramGiven != "none" {
				return fmt.Errorf("JobParameters[%v] must have one of the following prefixes: %v", check, strings.Join(prefixes, ","))
			}
		}

		if check == "quantMode" {
			validModes := []string{"Combined", "AB"}
			if !utils.ItemInSlice(paramGiven, validModes) {
				return fmt.Errorf("JobParameters[%v] invalid value: %v, expected one of: %v", check, paramGiven, strings.Join(validModes, ","))
			}
		}
	}

	if job.ScheduleType == protos.ScheduledJob_AFTER_IMPORT {
		if /*job.ScheduledFirstTimeUnixSec > 0 ||*/ job.IntervalSec > 0 {
			return errors.New("AFTER_IMPORT jobs must not have time fields set")
		}

		// Check that scan id is set to use the imported one
		acceptableScanIds := []string{SCAN_ID_AUTO_IMPORTED}
		if job.JobType == protos.JobType_JT_RUN_PYTHON_SCRIPT {
			acceptableScanIds = append(acceptableScanIds, SCAN_ID_NONE)
		}
		if !utils.ItemInSlice(job.JobParameters["scanId"], acceptableScanIds) {
			return fmt.Errorf("JobParameters[scanId] expected to be one of [%v], set to %v", strings.Join(acceptableScanIds, ","), job.JobParameters["scanId"])
		}
	} else if job.ScheduleType == protos.ScheduledJob_TIME_BASED {
		if job.JobOrder != 0 {
			return errors.New("TIME_BASED jobs must not have order set")
		}

		// Set the first time to something reasonable
		// if job.ScheduledFirstTimeUnixSec < 1561982400 {
		// 	return errors.New("Job run first time must be contemporary")
		// }

		// Interval can't be too low!
		if job.IntervalSec < 900 {
			return errors.New("Job run interval must be at least 15 minutes (900 seconds)")
		}
	} else if job.ScheduleType != protos.ScheduledJob_AFTER_IMPORT && job.ScheduleType != protos.ScheduledJob_TIME_BASED {
		return errors.New("ScheduleType must be AFTER_IMPORT or TIME_BASED")
	}
	/*
		if job.Instrument == protos.ScanInstrument_UNKNOWN_INSTRUMENT {
				return errors.New("Scan instrument must be set")

		}
	*/
	return nil
}

func (jm *JobManager) SetScheduledJob(job *protos.ScheduledJob) (*protos.ScheduledJob, error) {
	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobScheduleName)

	// Check it's valid
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
