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
	"log"
	"os"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// AWS SDK Utils
// According to: https://docs.aws.amazon.com/sdk-for-go/api/aws/session/
// Sessions are safe to use concurrently as long as the Session is not being modified.
// So we generally get the session on startup and pass it around elsewhere...

// GetSession - returns an AWS session
func GetSession() (*session.Session, error) {
	region, regionPresent := os.LookupEnv("AWS_REGION")
	if regionPresent {
		log.Printf("Initializing AWS session for AWS_REGION (%s)", region)
		return GetSessionWithRegion(region)
	}
	region, defaultRegionPresent := os.LookupEnv("AWS_DEFAULT_REGION")
	if defaultRegionPresent {
		log.Printf("Initializing AWS session for AWS_DEFAULT_REGION (%s)", region)
		return GetSessionWithRegion(region)
	}
	log.Printf("Initializing default AWS session")
	return session.NewSession()
}

// GetSessionWithRegion - Can specify am S3 region, returns an AWS session
func GetSessionWithRegion(region string) (*session.Session, error) {
	sess, err := session.NewSession(&aws.Config{Region: aws.String(region)})
	if err != nil {
		return nil, err
	}
	return sess, nil
}

// GetS3 - returns an S3 session
func GetS3(sess *session.Session) (s3iface.S3API, error) {
	svc := s3.New(sess)
	return svc, nil
}
