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

package services

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"go.mongodb.org/mongo-driver/mongo"

	expressionStorage "github.com/pixlise/core/v2/core/expressions"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/notifications"
	"github.com/pixlise/core/v2/core/timestamper"
	"github.com/pixlise/core/v2/core/utils"

	"github.com/getsentry/sentry-go"
	"github.com/pixlise/core/v2/core/pixlUser"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/aws/aws-sdk-go/service/sns"
	"github.com/pixlise/core/v2/api/config"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/logger"
	mongoDBConnection "github.com/pixlise/core/v2/core/mongo"
)

// NOTE: these 2 vars are set during compilation in gitlab CI build (see Makefile)
var ApiVersion string
var GitHash string

// This defines some generic interfaces that are used by a lot of the API code. Instead
// of using a bunch of global variables we pass around this services object and other
// code has access to a logger, random string generator etc.
// This comes in very useful when writing unit tests, since we can mock these interfaces

// IJWTReader - User ID getter from HTTP request
type IJWTReader interface {
	GetUserInfo(*http.Request) (pixlUser.UserInfo, error)
}

// IDGenerator - Generates ID strings
type IDGenerator interface {
	GenObjectID() string
}

// ExportZipper - Interface for creating an export zip file
type ExportZipper interface {
	MakeExportFilesZip(*APIServices, string, string, string, string, string, []string, []string) ([]byte, error)
	//MakeExportImagesZip(*APIServices, string, string, string, string, string, string, []string) ([]byte, error)
}

// URLSigner - Generates AWS S3 signed URLs
type URLSigner interface {
	GetSignedURL(s3iface.S3API, string, string, time.Duration) (string, error)
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////

// APIServices contains any services that HTTP handlers would want to use, like logging/config reading
type APIServices struct {
	// Configuration read in on startup
	Config config.APIConfig

	// Default logger
	Log logger.ILogger

	// This is configured on startup to talk to the configured AWSCloudwatchRegion
	AWSSessionCW *session.Session

	// Anything talking to S3 should use this
	S3 s3iface.S3API

	SNS awsutil.SNSInterface

	// Anything accessing files should use this
	FS fileaccess.FileAccess

	// Validation of JWT tokens
	JWTReader IJWTReader

	// ID generator
	IDGen IDGenerator

	// URL signer for S3
	Signer URLSigner

	// Zip File Generator
	Exporter ExportZipper

	// Notification Handler
	Notifications notifications.NotificationManager

	// Timestamp retriever - so can be mocked for unit tests
	TimeStamper timestamper.ITimeStamper

	// Our mongo db connection
	Mongo *mongo.Client

	// "User DB"
	Users pixlUser.UserDetailsLookup

	// "Expression DB"
	Expressions expressionStorage.ExpressionDB
}

// InitAPIServices sets up a new APIServices instance
func InitAPIServices(cfg config.APIConfig, jwtReader IJWTReader, idGen IDGenerator, signer URLSigner, exporter ExportZipper) APIServices {
	// Get a session for the bucket region
	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("Failed to create AWS session. Error: %v", err)
	}

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("Failed to create AWS S3 service. Error: %v", err)
	}

	fs := fileaccess.MakeS3Access(s3svc)

	// Init default logger - if we're local, we just output to stdout
	// NOTE: we contain multiple streams for the one application in the one log group. Here we define
	// a log group for the API for this environment, and other parts of the code that deal with logging will write
	// there also
	var ourLogger logger.ILogger
	if cfg.EnvironmentName == "local" {
		ourLogger = &logger.StdOutLogger{}
	} else {
		ourLogger, err = logger.InitCloudWatchLogger(
			sess,
			"/api/"+cfg.EnvironmentName,
			// Startup date/time, but with randomness after so it's likely unique
			fmt.Sprintf("%v (%v)", time.Now().Format("02-Jan-2006 15-04-05"), utils.RandStringBytesMaskImpr(8)),
			cfg.LogLevel,
			30, // Log retention for 30 days
			3,  // Send logs every 3 seconds in batches
		)

		if err != nil {
			log.Fatalf("Failed to initialise API logger: %v", err)
		}
	}

	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.SentryEndpoint,
		Environment: cfg.EnvironmentName,
		Release:     ApiVersion,
	}); err != nil {
		ourLogger.Errorf("Sentry initialization failed: %v", err)
	}

	snsSvc := sns.New(sess)

	var mongoClient *mongo.Client

	// Connect to mongo
	if len(cfg.MongoSecret) > 0 {
		// Remote is configured, connect to it
		mongoConnectionInfo, err := mongoDBConnection.GetMongoConnectionInfoFromSecretCache(sess, cfg.MongoSecret)
		if err != nil {
			err2 := fmt.Errorf("failed to read mongo DB connection info from secrets cache: %v", err)
			ourLogger.Errorf("%v", err2)
			log.Fatalf("%v", err)
		}

		mongoClient, err = mongoDBConnection.ConnectToRemoteMongoDB(
			mongoConnectionInfo.Host,
			mongoConnectionInfo.Username,
			mongoConnectionInfo.Password,
			ourLogger,
		)
		if err != nil {
			err2 := fmt.Errorf("failed connect to remote mongo: %v", err)
			ourLogger.Errorf("%v", err2)
			log.Fatalf("%v", err)
		}

	} else {
		// Connect to local mongo
		mongoClient, err = mongoDBConnection.ConnectToLocalMongoDB(ourLogger)
		if err != nil {
			err2 := fmt.Errorf("failed connect to local mongo: %v", err)
			ourLogger.Errorf("%v", err2)
			log.Fatalf("%v", err)
		}
	}

	return APIServices{
		Config:       cfg,
		Log:          ourLogger,
		AWSSessionCW: sess,
		FS:           fs,
		S3:           s3svc,
		SNS:          awsutil.RealSNS{SNS: snsSvc},
		JWTReader:    jwtReader,
		IDGen:        idGen,
		Signer:       signer,
		Exporter:     exporter,
		TimeStamper:  &timestamper.UnixTimeNowStamper{},
		Mongo:        mongoClient,
	}
}
