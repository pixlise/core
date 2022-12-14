// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package awsutil

import (
	"fmt"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/ses"
	"github.com/aws/aws-sdk-go/service/sns"
)

func getSES() *ses.SES {
	sess, err := GetSession()
	if err != nil {
		return nil
	}

	svc := ses.New(sess)
	return svc
}

func getSNS() (*sns.SNS, error) {
	sess, err := GetSession()
	if err != nil {
		return nil, err
	}

	svc := sns.New(sess)
	return svc, nil
}

//SNSSendSms Send an SMS via SNS
func SNSSendSms(phonenumber string, message string) error {
	svc, err := getSNS()
	if err != nil {
		return err
	}
	params := &sns.PublishInput{
		Message:     aws.String(message),
		PhoneNumber: aws.String(phonenumber),
		MessageAttributes: map[string]*sns.MessageAttributeValue{
			"AWS.SNS.SMS.SenderID": {StringValue: aws.String("Pixlise"), DataType: aws.String("String")},
			"AWS.SNS.SMS.SMSType":  {StringValue: aws.String("Transactional"), DataType: aws.String("String")},
		},
	}
	_, err = svc.Publish(params)
	if err != nil {
		return err
	}

	return nil
}

//SESSendEmail - Send an email via SES
func SESSendEmail(emailAddress string, charset string, textBody string, htmlBody string, subject string, sender string, cc []string, bcc []string) {
	svc := getSES()

	Recipient := fmt.Sprintf("%v", emailAddress)

	var ccaddr []*string
	for _, c := range cc {
		ccaddr = append(ccaddr, aws.String(c))
	}
	var bccaddr []*string
	for _, c := range bcc {
		bccaddr = append(bccaddr, aws.String(c))
	}
	// Assemble the email.
	input := &ses.SendEmailInput{
		Destination: &ses.Destination{
			CcAddresses:  ccaddr,
			BccAddresses: bccaddr,
			ToAddresses: []*string{
				aws.String(Recipient),
			},
		},
		Message: &ses.Message{
			Body: &ses.Body{
				Html: &ses.Content{
					Charset: aws.String(charset),
					Data:    aws.String(htmlBody),
				},
				Text: &ses.Content{
					Charset: aws.String(charset),
					Data:    aws.String(textBody),
				},
			},
			Subject: &ses.Content{
				Charset: aws.String(charset),
				Data:    aws.String(subject),
			},
		},
		Source: aws.String(sender),
	}

	// Attempt to send the email.
	_, err := svc.SendEmail(input)

	// Display error messages if they occur.
	if err != nil {
		if aerr, ok := err.(awserr.Error); ok {
			switch aerr.Code() {
			case ses.ErrCodeMessageRejected:
				fmt.Println(ses.ErrCodeMessageRejected, aerr.Error())
			case ses.ErrCodeMailFromDomainNotVerifiedException:
				fmt.Println(ses.ErrCodeMailFromDomainNotVerifiedException, aerr.Error())
			case ses.ErrCodeConfigurationSetDoesNotExistException:
				fmt.Println(ses.ErrCodeConfigurationSetDoesNotExistException, aerr.Error())
			default:
				fmt.Println(aerr.Error())
			}
		} else {
			// Print the error, cast err to awserr.Error to get the Code and
			// Message from an error.
			fmt.Println(err.Error())
		}

		return
	}

	fmt.Println("Email Sent to address: " + Recipient)
}
