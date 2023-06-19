package awsutil

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sqs"
)

func PurgeQueue(sess session.Session, url string) error {
	sqsClient := sqs.New(&sess)

	in := sqs.PurgeQueueInput{QueueUrl: &url}
	_, err := sqsClient.PurgeQueue(&in)
	if err != nil {
		return err
	}
	return nil
}
