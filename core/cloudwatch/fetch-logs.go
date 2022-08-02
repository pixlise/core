// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package cloudwatch

import (
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudwatchlogs"
	"github.com/pixlise/core/api/services"
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
