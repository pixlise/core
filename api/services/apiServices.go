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

/*
Services used by API endpoint handlers and other bits of code. This is a collection
of common things needed by code, such as:
  - Access to an instance of logger
  - AWS S3 API
  - The current Mongo DB connection
  - Facilities to send user notifications
  - API configuration

among others
*/
package services

import (
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/timestamper"
	"go.mongodb.org/mongo-driver/mongo"
)

// NOTE: these 2 vars are set during compilation in CI build (see Makefile)
var ApiVersion string
var GitHash string

// This defines some generic interfaces that are used by a lot of the API code. Instead
// of using a bunch of global variables we pass around this services object and other
// code has access to a logger, random string generator etc.
// This comes in very useful when writing unit tests, since we can mock these interfaces

type APIServices struct {
	// Configuration read in on startup
	Config config.APIConfig

	// Default logger
	Log logger.ILogger

	// Anything talking to S3 should use this
	S3 s3iface.S3API

	// AWS SNS - At time of writing used for triggering Data Import lambda (it gets triggered by SNS
	// from iSDS pipeline, so we also trigger it the same way) and sending email alerts
	SNS awsutil.SNSInterface

	// AWS SQS - At time of writing used for triggering Image Coreg importing lambda
	SQS awsutil.SQSInterface

	// Anything accessing files should use this
	FS fileaccess.FileAccess

	// Validation of JWT tokens
	JWTReader jwtparser.IJWTReader

	// ID generator
	IDGen idgen.IDGenerator

	// Timestamp retriever - so can be mocked for unit tests
	TimeStamper timestamper.ITimeStamper

	// Our mongo db connection
	MongoDB *mongo.Database

	// And how we connected to it (so we can run mongodump later if needed)
	MongoDetails mongoDBConnection.MongoConnectionDetails

	Notifier INotifier

	// The unique identifier of this API instance (so we can log/debug issues that are cross-instance!)
	InstanceId string
}
