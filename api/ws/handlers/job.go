package wsHandler

import (
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
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

	jobs, activeJobs, totalJobs, err := hctx.Svcs.JobManager.ListJobs(isAdmin, hctx.SessUser.User.Id, limit, skip)
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
	job, err := hctx.Svcs.JobManager.SetScheduledJob(req.Job)
	if err != nil {
		return nil, err
	}

	return &protos.SetScheduledJobResp{Job: job}, nil
}
