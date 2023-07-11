package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/logger"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleLogReadReq(req *protos.LogReadReq, hctx wsHelpers.HandlerContext) (*protos.LogReadResp, error) {
	return nil, errors.New("HandleLogReadReq not implemented yet")
	/*
	   	if err := wsHelpers.CheckStringField(&req.LogStreamId, "LogStreamId", 1, 512); err != nil {
	   		return nil, err
	   	}

	   logGroup := "/dataset-importer/" + hctx.Svcs.Config.EnvironmentName
	   logs, err := cloudwatch.FetchLogs(hctx.Svcs, logGroup, req.LogStreamId)

	   	if aerr, ok := err.(awserr.Error); ok {
	   		if aerr.Code() == cloudwatchlogs.ErrCodeResourceNotFoundException {
	   			return nil, errorwithstatus.MakeNotFoundError(req.LogStreamId)
	   		}
	   	}

	   for _, logEntry := range logs {

	   }

	   // Got it, return it

	   	return &protos.LogReadResp{
	   		Entries: entries,
	   	}, nil
	*/
}

func HandleLogGetLevelReq(req *protos.LogGetLevelReq, hctx wsHelpers.HandlerContext) (*protos.LogGetLevelResp, error) {
	name, err := logger.GetLogLevelName(hctx.Svcs.Log.GetLogLevel())
	if err != nil {
		return nil, err
	}

	return &protos.LogGetLevelResp{
		LogLevelId: name,
	}, nil
}

func HandleLogSetLevelReq(req *protos.LogSetLevelReq, hctx wsHelpers.HandlerContext) (*protos.LogSetLevelResp, error) {
	if err := wsHelpers.CheckStringField(&req.LogLevelId, "LogLevelId", 1, 10); err != nil {
		return nil, err
	}

	logLevel, err := logger.GetLogLevel(req.LogLevelId)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Also set it on the actual logger
	hctx.Svcs.Log.SetLogLevel(logLevel)

	// Not really an error, but we log in this level to ensure it always gets printed
	hctx.Svcs.Log.Errorf("User %v request changed log level to: %v", hctx.SessUser.User.Id, req.LogLevelId)

	return &protos.LogSetLevelResp{LogLevelId: req.LogLevelId}, nil
}
