package awsutil

import (
	"github.com/aws/aws-sdk-go/service/sqs"
)

type SQSInterface interface {
	SendMessage(msg *sqs.SendMessageInput) (*sqs.SendMessageOutput, error)
}

type RealSQS struct {
	SQS *sqs.SQS
}

func (rsqs RealSQS) SendMessage(input *sqs.SendMessageInput) (*sqs.SendMessageOutput, error) {
	return rsqs.SQS.SendMessage(input)
}
