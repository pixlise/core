package jobmanager

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (jm *JobManager) ListJobs(isAdmin bool, requestorUserId string, skip, limit int64) ([]*protos.JobStatus, []*protos.JobStatus, uint32, error) {
	filter := bson.M{}

	ctx := context.TODO()
	coll := jm.svcs.MongoDB.Collection(dbCollections.JobStatusName)

	sort := bson.D{{"lastupdateunixtimesec", -1}}

	opts := options.Find().SetSort(sort).SetLimit(limit).SetSkip(skip)
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, nil, 0, err
	}

	items := []*protos.JobStatus{}
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, nil, 0, err
	}

	itemsToSend := []*protos.JobStatus{}
	if isAdmin {
		itemsToSend = items
	} else {
		// Find only the jobs that were requested by this user
		for _, item := range items {
			if item.RequestorUserId == requestorUserId {
				itemsToSend = append(itemsToSend, item)
			}
		}
	}

	jobs := []*protos.JobStatus{}
	activeJobs := []*protos.JobStatus{}

	//nowUnixSec := hctx.Svcs.TimeStamper.GetTimeNowSec()

	for _, j := range itemsToSend {
		if j.Status == protos.JobStatus_COMPLETE || j.Status == protos.JobStatus_ERROR /*|| nowUnixSec-int64(j.LastUpdateUnixTimeSec) > 3600*/ {
			jobs = append(jobs, j)
		} else {
			activeJobs = append(activeJobs, j)
		}
	}

	return jobs, activeJobs, uint32(len(itemsToSend)), nil
}

func (jm *JobManager) GetJob(JobId string) (jobconfig.JobGroupConfig, error) {
	return jobconfig.JobGroupConfig{}, nil
}

/*
func HandleScanListJobsReq(req *protos.ScanListJobsReq, hctx wsHelpers.HandlerContext) (*protos.ScanListJobsResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.JobsName)
	cursor, err := coll.Find(ctx, bson.M{}, options.Find())
	if err != nil {
		return nil, err
	}

	jobs := []*protos.JobGroupConfig{}
	err = cursor.All(context.TODO(), &jobs)
	if err != nil {
		return nil, err
	}

	return &protos.ScanListJobsResp{
		Jobs: jobs,
	}, nil
}*/
