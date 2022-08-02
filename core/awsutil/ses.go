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
