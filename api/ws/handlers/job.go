package wsHandler

import (
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	jobmanager "github.com/pixlise/core/v4/api/job/manager"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleJobListReq(req *protos.JobListReq, hctx wsHelpers.HandlerContext) (*protos.JobListResp, error) {
	// Work out if requestor is an admin or a normal user
	isAdmin := hctx.SessUser.Permissions["PIXLISE_ADMIN"]

	skip := int64(0)
	limit := int64(1000)

	if req.JobCount > 0 {
		limit = int64(req.JobCount)
	}
	if req.SkipJobs > 0 {
		skip = int64(req.SkipJobs)
	}

	jobs, activeJobs, totalJobs, err := hctx.Svcs.JobManager.ListJobs(isAdmin, hctx.SessUser.User.Id, skip, limit, req.JobTypes)
	if err != nil {
		return nil, err
	}

	result := &protos.JobListResp{
		Jobs:          jobs,
		ActiveJobs:    activeJobs,
		TotalJobCount: uint32(totalJobs),
	}

	return result, nil
}

func HandleDeleteScheduledJobReq(req *protos.DeleteScheduledJobReq, hctx wsHelpers.HandlerContext) (*protos.DeleteScheduledJobResp, error) {
	err := hctx.Svcs.JobManager.DeleteScheduledJob(req.Id)
	if err != nil {
		return nil, err
	}

	return &protos.DeleteScheduledJobResp{}, nil
}

func HandleScheduledJobListReq(req *protos.ScheduledJobListReq, hctx wsHelpers.HandlerContext) (*protos.ScheduledJobListResp, error) {
	jobs, err := hctx.Svcs.JobManager.ListScheduledJobs()
	if err != nil {
		return nil, err
	}

	return &protos.ScheduledJobListResp{Jobs: jobs}, nil
}

func HandleSetScheduledJobReq(req *protos.SetScheduledJobReq, hctx wsHelpers.HandlerContext) (*protos.SetScheduledJobResp, error) {
	if req.Job == nil {
		return nil, errors.New("Job must be set in SetScheduledJobReq")
	}

	job, err := hctx.Svcs.JobManager.SetScheduledJob(req.Job)
	if err != nil {
		return nil, err
	}

	return &protos.SetScheduledJobResp{Job: job}, nil
}

func HandleTriggerScheduledJobReq(req *protos.TriggerScheduledJobReq, hctx wsHelpers.HandlerContext) (*protos.TriggerScheduledJobResp, error) {
	if len(req.ScheduledJobId) <= 0 {
		return nil, errors.New("ScheduledJobId must be set in TriggerScheduledJobReq")
	}

	if len(req.JobParameters) <= 0 {
		return nil, errors.New("JobParameters must be set in TriggerScheduledJobReq")
	}

	// Get the job
	scheduledJob, err := hctx.Svcs.JobManager.GetScheduledJob(req.ScheduledJobId)
	if err != nil {
		return nil, err
	}

	// Read the scan - if job is set to use the imported one, we expect there to be a scan specified
	// in the request!
	scanId := scheduledJob.JobParameters["scanId"]
	if scanId == jobmanager.SCAN_ID_AUTO_IMPORTED {
		if reqScanId, ok := req.JobParameters["scanId"]; ok {
			scanId = reqScanId
		}
	}

	if scanId == jobmanager.SCAN_ID_AUTO_IMPORTED {
		return nil, fmt.Errorf("Failed to determine actual scan id for job, only got %v", jobmanager.SCAN_ID_AUTO_IMPORTED)
	}

	scanItem := &protos.ScanItem{}
	err = expressionrunner.ReadOne(dbCollections.ScansName, bson.M{"_id": scanId}, &scanItem, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, fmt.Errorf("Could not read scan %v: %v", scanId, err)
	}

	err = hctx.Svcs.JobManager.RunScheduledJob(scheduledJob, scanItem)
	if err != nil {
		return nil, err
	}

	return &protos.TriggerScheduledJobResp{}, nil
}
