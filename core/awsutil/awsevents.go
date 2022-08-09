package awsutil

import (
	"encoding/json"
	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go/aws/arn"
	"github.com/pkg/errors"
)

type eventType int

const (
	unknownEventType eventType = iota
	s3EventType
	snsEventType
	sqsEventType
)

type Record struct {
	EventSource    string
	EventSourceArn string
	AWSRegion      string
	S3             events.S3Entity
	SQS            events.SQSMessage
	SNS            events.SNSEntity
}

type Event struct {
	Records []Record
}

// getEventType - Get the event type from the stream
func (event *Event) getEventType(data []byte) eventType {
	temp := make(map[string]interface{})
	json.Unmarshal(data, &temp)

	recordsList, _ := temp["Records"].([]interface{})
	record, _ := recordsList[0].(map[string]interface{})

	var eventSource string

	if es, ok := record["EventSource"]; ok {
		eventSource = es.(string)

	} else if es, ok := record["eventSource"]; ok {
		eventSource = es.(string)
	}

	switch eventSource {
	case "aws:s3":
		return s3EventType
	case "aws:sns":
		return snsEventType
	case "aws:sqs":
		return sqsEventType
	}

	return unknownEventType
}

// mapS3EventRecords - Create an S3 record on receipt of an S3 Event
func (event *Event) mapS3EventRecords(s3Event *events.S3Event) error {
	event.Records = make([]Record, 0)

	for _, s3Record := range s3Event.Records {
		event.Records = append(event.Records, Record{
			EventSource:    s3Record.EventSource,
			EventSourceArn: s3Record.S3.Bucket.Arn,
			AWSRegion:      s3Record.AWSRegion,
			S3:             s3Record.S3,
		})
	}

	return nil
}

// mapSNSEventRecords - Create an SNS record on receipt of an SNS Event
func (event *Event) mapSNSEventRecords(snsEvent *events.SNSEvent) error {
	event.Records = make([]Record, 0)

	for _, snsRecord := range snsEvent.Records {
		// decode sns message to s3 event
		//s3Event := &events.S3Event{}
		//err := json.Unmarshal([]byte(snsRecord.SNS.Message), s3Event)
		//if err != nil {
		//	return errors.Wrap(err, "Failed to decode sns message to an S3 event")
		//}

		/*if len(s3Event.Records) == 0 {
			return errors.New("S3 Event Records is empty")
		}*/

		//for _, s3Record := range s3Event.Records {
		topicArn, err := arn.Parse(snsRecord.SNS.TopicArn)
		if err != nil {
			return err
		}

		event.Records = append(event.Records, Record{
			EventSource:    snsRecord.EventSource,
			EventSourceArn: snsRecord.SNS.TopicArn,
			AWSRegion:      topicArn.Region,
			SNS:            snsRecord.SNS,
		})
		//}
	}

	return nil
}

// mapSQSEventRecords - Decode the SQS Event
func (event *Event) mapSQSEventRecords(sqsEvent *events.SQSEvent) error {
	event.Records = make([]Record, 0)

	for _, sqsRecord := range sqsEvent.Records {
		// decode sqs body to s3 event
		s3Event := &events.S3Event{}
		err := json.Unmarshal([]byte(sqsRecord.Body), s3Event)
		if err != nil {
			return errors.Wrap(err, "Failed to decode sqs body to an S3 event")
		}

		if len(s3Event.Records) == 0 {
			return errors.New("S3 Event Records is empty")
		}

		for _, s3Record := range s3Event.Records {
			event.Records = append(event.Records, Record{
				EventSource:    sqsRecord.EventSource,
				EventSourceArn: sqsRecord.EventSourceARN,
				AWSRegion:      sqsRecord.AWSRegion,
				SQS:            sqsRecord,
				S3:             s3Record.S3,
			})
		}
	}

	return nil
}

// UnmarshalJSON - Decode the JSON to the correct Event type
func (event *Event) UnmarshalJSON(data []byte) error {
	//eType := event.getEventType(data)
	var err error
	switch event.getEventType(data) {
	case s3EventType:
		s3Event := &events.S3Event{}
		err = json.Unmarshal(data, s3Event)

		if err == nil {
			return event.mapS3EventRecords(s3Event)
		}

	case snsEventType:
		snsEvent := &events.SNSEvent{}
		err = json.Unmarshal(data, snsEvent)

		if err == nil {
			return event.mapSNSEventRecords(snsEvent)
		}

	case sqsEventType:
		sqsEvent := &events.SQSEvent{}
		err = json.Unmarshal(data, sqsEvent)

		if err == nil {
			return event.mapSQSEventRecords(sqsEvent)
		}
	}

	return err
}
