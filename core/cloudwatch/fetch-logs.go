package cloudwatch

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pixlise/core/v2/api/services"
)

func getLogEvents(sess *session.Session, limit *int64, logGroupName *string, logStreamName *string) (*cloudwatchlogs.GetLogEventsOutput, error) {
	svc := cloudwatchlogs.New(sess)

	resp, err := svc.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		Limit:         limit,
		LogGroupName:  logGroupName,
		LogStreamName: logStreamName,
	})
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func lookUpStreamName(sess *session.Session, logGroup string) (*string, error) {
	svc := cloudwatchlogs.New(sess)
	streams, err := svc.DescribeLogStreams(&cloudwatchlogs.DescribeLogStreamsInput{
		LogGroupName: &logGroup,
		Descending:   aws.Bool(true),
		OrderBy:      aws.String("LastEventTime"),
	})
	if err != nil {
		return nil, err
	}
	if len(streams.LogStreams) > 0 {
		return streams.LogStreams[0].LogStreamName, nil
	} else {
		return nil, errors.New("could not find any log groups")
	}
}

type LogLine struct {
	TimestampUnixMs int64  `json:"timeStampUnixMs"`
	Message         string `json:"message"`
}

type LogData struct {
	Lines []LogLine `json:"lines"`
}

func FetchLogs(logGroupName string, services *services.APIServices) (LogData, error) {
	var limit int64 = 10000
	//log := services.Log

	result := LogData{Lines: []LogLine{}}

	sname, err := lookUpStreamName(services.AWSSessionCW, logGroupName)

	resp, err := getLogEvents(services.AWSSessionCW, &limit, &logGroupName, sname)
	if err != nil {
		//log.Errorf("Got error getting log events: %v", err)
		return result, err
	}

	//log.Infof("Event messages for log group  " + logGroupName)

	for _, event := range resp.Events {
		result.Lines = append(result.Lines, LogLine{TimestampUnixMs: *event.IngestionTime, Message: *event.Message})
	}

	return result, nil
}
