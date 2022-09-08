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
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"log"
	"net/http"
	"time"

	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/notifications"

	"github.com/getsentry/sentry-go"
	"github.com/pixlise/core/api/esutil"
	"github.com/pixlise/core/core/pixlUser"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3/s3iface"
	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/logger"
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

type ITimeStamper interface {
	GetTimeNowSec() int64
}

type UnixTimeNowStamper struct {
}

// GetTimeNowSec - Returns unix time now in seconds
func (ts *UnixTimeNowStamper) GetTimeNowSec() int64 {
	return time.Now().Unix()
}

type MockTimeNowStamper struct {
	QueuedTimeStamps []int64
}

// GetTimeNowSec - Returns unix time now in seconds
func (ts *MockTimeNowStamper) GetTimeNowSec() int64 {
	val := ts.QueuedTimeStamps[0]
	ts.QueuedTimeStamps = ts.QueuedTimeStamps[1:]
	return val
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

	// Anything accessing files should use this
	FS fileaccess.FileAccess

	// For Event Logging
	ES esutil.Connection

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
	TimeStamper ITimeStamper

	SecretsManager *secretcache.Cache
}

// InitAPIServices sets up a new APIServices instance
func InitAPIServices(cfg config.APIConfig, jwtReader IJWTReader, idGen IDGenerator, signer URLSigner, exporter ExportZipper, notifications notifications.NotificationManager) APIServices {
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:         cfg.SentryEndpoint,
		Environment: cfg.EnvironmentName,
		Release:     ApiVersion,
	}); err != nil {
		fmt.Printf("Sentry initialization failed: %v\n", err)
	}

	// Get a session for the bucket region
	sessBucket, err := awsutil.GetSessionWithRegion(cfg.AWSBucketRegion)
	if err != nil {
		log.Fatalf("Failed to create AWS session for region: %v. Error: %v", cfg.AWSBucketRegion, err)
	}

	s3svc, err := awsutil.GetS3(sessBucket)
	if err != nil {
		log.Fatalf("Failed to create AWS S3 access for region: %v. Error: %v", cfg.AWSBucketRegion, err)
	}

	fs := fileaccess.MakeS3Access(s3svc)

	sessCW, err := awsutil.GetSessionWithRegion(cfg.AWSCloudwatchRegion)
	if err != nil {
		log.Fatalf("Failed to create AWS session for region: %v. Error: %v", cfg.AWSCloudwatchRegion, err)
	}
	// Init default logger
	ourLogger, err := logger.Init(logger.DefaultGroup, cfg.LogLevel, cfg.EnvironmentName, sessCW)

	if err != nil {
		log.Fatalf("Failed to initialise API logger: %v", err)
	}

	client := esutil.FullFatClient(cfg, ourLogger)
	es, err := esutil.Connect(client, cfg)
	if err != nil {
		ourLogger.Errorf("Failed to connect to Elastic Search: %v", err)
	}

	/* Took this out because it looked like test code that was left in?
	o := esutil.LoggingObject{
		Instance:  "Test",
		Time:      time.Now(),
		Component: "Test Component",
		Message:   "Test Message",
		Params:    nil,
		Environment: "Test",
		User: "5838239847",
	}
	esutil.InsertLogRecord(es, o, ourLogger)
	*/
	secretscache, err := secretcache.New()
	if err != nil {
		ourLogger.Errorf("Failed to bootstrap secrets manager")
	}
	return APIServices{
		Config:         cfg,
		Log:            ourLogger,
		AWSSessionCW:   sessCW,
		FS:             fs,
		S3:             s3svc,
		ES:             es,
		JWTReader:      jwtReader,
		IDGen:          idGen,
		Signer:         signer,
		Exporter:       exporter,
		Notifications:  notifications,
		TimeStamper:    &UnixTimeNowStamper{},
		SecretsManager: secretscache,
	}
}
