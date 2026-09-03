package wsHandler

import (
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	jobconfig "github.com/pixlise/core/v4/api/job/config"
	expressionrunner "github.com/pixlise/core/v4/api/job/jobrunner/expression-runner"
	jobmanager "github.com/pixlise/core/v4/api/job/manager"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
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

func HandleJobGetReq(req *protos.JobGetReq, hctx wsHelpers.HandlerContext) (*protos.JobGetResp, error) {
	// Work out if requestor is an admin or a normal user
	isAdmin := hctx.SessUser.Permissions["PIXLISE_ADMIN"]

	// Read everything we can about this job
	status, config, err := hctx.Svcs.JobManager.GetJob(req.JobId, isAdmin, hctx.SessUser.User.Id)
	if err != nil {
		return nil, err
	}

	return &protos.JobGetResp{
		Status: status,
		Config: config,
	}, nil
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

	var scanItem *protos.ScanItem

	if scanId != jobmanager.SCAN_ID_NONE {
		err = expressionrunner.ReadOne(dbCollections.ScansName, bson.M{"_id": scanId}, &scanItem, hctx.Svcs.MongoDB)
		if err != nil {
			return nil, fmt.Errorf("Could not read scan %v: %v", scanId, err)
		}
	}

	if quant, ok := req.JobParameters["quant"]; ok {
		scheduledJob.JobParameters["quant"] = quant
	}

	jobId, err := hctx.Svcs.JobManager.RunScheduledJob(scheduledJob, scanItem)
	if err != nil {
		return nil, err
	}

	return &protos.TriggerScheduledJobResp{JobId: jobId}, nil
}

func HandleJobOutputGetReq(req *protos.JobOutputGetReq, hctx wsHelpers.HandlerContext) (*protos.JobOutputGetResp, error) {
	if err := wsHelpers.CheckStringField(&req.JobId, "JobId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.FilePath, "FilePath", 1, 512); err != nil {
		return nil, err
	}

	// Get the job config so we can independently get the bucket/path etc, confirming it exists and this isn't
	// some client code stabbing in the dark!
	filter := bson.M{"_id": req.JobId}
	config := &protos.JobGroupConfig{}
	if err := expressionrunner.ReadOne(dbCollections.JobsName, filter, config, hctx.Svcs.MongoDB); err != nil {
		return nil, err
	}

	if req.NodeIndex >= config.NodeCount {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid node index: %v", req.NodeIndex))
	}

	nodeConfig := jobconfig.FlattenJobConfig(config.NodeConfig, uint(req.NodeIndex))

	// At this point, we should be able to find the requested path
	for _, out := range nodeConfig.OutputFiles {
		if out.RemotePath == req.FilePath {
			// Found it!
			fileData, err := hctx.Svcs.FS.ReadObject(out.RemoteBucket, out.RemotePath)
			if err != nil {
				if hctx.Svcs.FS.IsNotFoundError(err) {
					return nil, errorwithstatus.MakeNotFoundError(req.FilePath)
				}
				return nil, err
			}

			return &protos.JobOutputGetResp{
				Content: fileData,
			}, nil
		}
	}

	return nil, errorwithstatus.MakeNotFoundError(fmt.Sprintf("Node: %v, Path %v", req.NodeIndex, req.FilePath))
}
