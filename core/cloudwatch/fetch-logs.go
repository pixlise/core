package cloudwatch

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pixlise/core/v3/api/services"
)

/* Used to query the latest log stream name within a group...

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
*/

type LogLine struct {
	TimestampUnixMs int64  `json:"timeStampUnixMs"`
	Message         string `json:"message"`
}

type LogData struct {
	Lines []LogLine `json:"lines"`
}

func FetchLogs(services *services.APIServices, logGroupName string, logStreamName string) (LogData, error) {
	var limit int64 = 10000

	result := LogData{Lines: []LogLine{}}

	//logStreamName, err := lookUpStreamName(services.AWSSessionCW, logGroupName)

	svc := cloudwatchlogs.New(services.AWSSessionCW)
	resp, err := svc.GetLogEvents(&cloudwatchlogs.GetLogEventsInput{
		Limit:         &limit,
		LogGroupName:  aws.String(logGroupName),
		LogStreamName: aws.String(logStreamName),
	})

	if err != nil {
		//log.Errorf("Got error getting log events: %v", err)
		return result, err
	}

	for _, event := range resp.Events {
		result.Lines = append(result.Lines, LogLine{TimestampUnixMs: *event.IngestionTime, Message: *event.Message})
	}

	return result, nil
}
