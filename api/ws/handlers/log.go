package wsHandler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleLogReadReq(req *protos.LogReadReq, hctx wsHelpers.HandlerContext) ([]*protos.LogReadResp, error) {
	if err := wsHelpers.CheckStringField(&req.LogStreamId, "LogStreamId", 1, 512); err != nil {
		return nil, err
	}

	// We now just send the AWS cloudwatch log group+stream id out, so expect this to be directly accessible
	bits := strings.Split(req.LogStreamId, "/|/")
	if len(bits) != 2 {
		return nil, errors.New("Failed to get log group and stream from: " + req.LogStreamId)
	}

	logs, err := fetchLogs(hctx.Svcs, bits[0], bits[1])

	if aerr, ok := err.(awserr.Error); ok {
		if aerr.Code() == cloudwatchlogs.ErrCodeResourceNotFoundException {
			return nil, errorwithstatus.MakeNotFoundError(req.LogStreamId)
		}
	}

	return []*protos.LogReadResp{&protos.LogReadResp{
		Entries: logs,
	}}, nil
}

func HandleLogGetLevelReq(req *protos.LogGetLevelReq, hctx wsHelpers.HandlerContext) ([]*protos.LogGetLevelResp, error) {
	name, err := logger.GetLogLevelName(hctx.Svcs.Log.GetLogLevel())
	if err != nil {
		return nil, err
	}

	return []*protos.LogGetLevelResp{&protos.LogGetLevelResp{
		LogLevelId: name,
	}}, nil
}

func HandleLogSetLevelReq(req *protos.LogSetLevelReq, hctx wsHelpers.HandlerContext) ([]*protos.LogSetLevelResp, error) {
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

	return []*protos.LogSetLevelResp{&protos.LogSetLevelResp{LogLevelId: req.LogLevelId}}, nil
}

var cloudwatchSvc *cloudwatchlogs.CloudWatchLogs = nil

func fetchLogs(services *services.APIServices, logGroupName string, logStreamName string) ([]*protos.LogLine, error) {
	var limit int64 = 10000

	result := []*protos.LogLine{}

	if cloudwatchSvc == nil {
		sess, err := awsutil.GetSession()
		if err != nil {
			return result, fmt.Errorf("Failed to create AWS session. Error: %v", err)
		}

		// NOTE: previously here we used a session: AWSSessionCW which could be configured to a different region... don't know why
		// this was required but seemed redundant, it was in the same region lately...
		cloudwatchSvc = cloudwatchlogs.New(sess)
	}

	if cloudwatchSvc == nil {
		return result, fmt.Errorf("No connection to cloudwatch")
	}

	resp, err := cloudwatchSvc.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		Limit:         &limit,
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
	})

	if err != nil {
		//log.Errorf("Got error getting log events: %v", err)
		return result, err
	}

	for _, event := range resp.Events {
		result = append(result, &protos.LogLine{
			// Split it up, we don't like sending uint64 via proto because deserialisation to JS turns to shit
			TimeStampUnixSec: uint32(*event.IngestionTime / 1000),
			TimeStampMs:      uint32(*event.IngestionTime % 1000),
			Message:          *event.Message,
		})
	}

	return result, nil
}
