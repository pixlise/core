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

package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	mongoDBConnection "github.com/pixlise/core/v2/core/mongo"
	"github.com/pixlise/core/v2/core/timestamper"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	apiNotifications "github.com/pixlise/core/v2/core/notifications"
	"github.com/pixlise/core/v2/data-import/importer"
	"github.com/pixlise/core/v2/data-import/importtime"
)

func HandleRequest(ctx context.Context, event awsutil.Event) (string, error) {
	configBucket := os.Getenv("CONFIG_BUCKET")
	datasetBucket := os.Getenv("DATASETS_BUCKET")
	manualBucket := os.Getenv("MANUAL_BUCKET")
	envName := os.Getenv("ENVIRONMENT_NAME")

	sess, err := awsutil.GetSession()
	if err != nil {
		return "", err
	}

	svc, err := awsutil.GetS3(sess)
	if err != nil {
		return "", err
	}

	// Was used to try to diagnose issues, but not needed as it appears /tmp is empty when we start!
	/*
		// Print contents of /tmp directory... AWS reuses nodes, we may have left files in the past
		localFS := fileaccess.FSAccess{}
		tmpFiles, err := localFS.ListObjects("/tmp", "")
		if err == nil {
			for c, tmpFile := range tmpFiles {
				fmt.Printf("%v: %v\n", c+1, tmpFile)
			}
		} else {
			fmt.Printf("Failed to list tmp files: %v\n", err)
		}
	*/

	remoteFS := fileaccess.MakeS3Access(svc)

	// Normally we'd only expect event.Records to be of length 1...
	worked := 0
	for _, record := range event.Records {
		mongoclient := connectMongo(&logger.StdOutLogger{})

		notificationStack, err := apiNotifications.MakeNotificationStack(mongoclient, envName, &timestamper.UnixTimeNowStamper{}, &logger.StdOutLogger{}, []string{})
		if err != nil {
			return "", err
		}
		if record.SNS.Subject == "TestNotifications" {
			runTest(mongoclient, notificationStack, record)
		} else {
			// Print this to stdout - not that useful, won't be in the log file, but lambda cloudwatch log should have it
			// and it'll be useful for initial debugging
			fmt.Printf("ImportForTrigger: \"%v\"\n", record.SNS.Message)

			result, err := importer.ImportForTrigger([]byte(record.SNS.Message), envName, configBucket, datasetBucket, manualBucket, nil, remoteFS)
			defer result.Logger.Close()

			if len(result.WhatChanged) > 0 {
				err := triggerNotifications(
					configBucket,
					result.DatasetTitle,
					remoteFS,
					result.IsUpdate,
					result.WhatChanged,
					notificationStack,
					result.Logger,
				)

				if err != nil {
					result.Logger.Errorf("ImportForTrigger triggerNotifications had an error: \"%v\"\n", err)
				}
			}

			// Delete the working directory here, there's no point leaving it on a lambda machine, we can't debug it
			// but if this code ran elsewhere we wouldn't delete it, to have something to look at
			if len(result.WorkingDir) > 0 {
				removeErr := os.RemoveAll(result.WorkingDir)
				if removeErr != nil {
					fmt.Printf("Failed to remove working dir: \"%v\". Error: %v\n", result.WorkingDir, removeErr)
				} else {
					fmt.Printf("Removed working dir: \"%v\"\n", result.WorkingDir)
				}
			}

			if err != nil {
				return "", err
			} else {
				worked++
			}
		}
	}
	return fmt.Sprintf("Imported %v records", worked), nil

}

func main() {
	os.Mkdir("/tmp/profile", 0750)
	lambda.Start(HandleRequest)
}

func triggerNotifications(
	configBucket string,
	datasetName string,
	fs fileaccess.FileAccess,
	update bool,
	updatetype string,
	notificationStack apiNotifications.NotificationManager,
	jobLog logger.ILogger) error {
	/*var pinusers = []string{"tom.barber@jpl.nasa.gov", "scott.davidoff@jpl.nasa.gov",
	"adrian.e.galvin@jpl.nasa.gov", "peter.nemere@qut.edu.au"}*/
	if notificationStack == nil {
		return errors.New("Notification Stack is empty, this is a success notification")
	}
	var err error

	template := make(map[string]interface{})
	template["datasourcename"] = datasetName

	lastImportUnixSec, err := importtime.GetDatasetImportUnixTimeSec(fs, configBucket, datasetName)

	// Print an error if we got one, but this can always continue...
	if err != nil {
		jobLog.Errorf("%v", err)
	}

	lastImportTime := time.Unix(int64(lastImportUnixSec), 0)
	if time.Since(lastImportTime).Minutes() < 60 {
		jobLog.Infof("Skipping notification send - one was sent recently")
	} else {
		if update {
			jobLog.Infof("Dispatching notification for updated datasource")

			template["subject"] = fmt.Sprintf("Datasource %v Processing Complete", template["datasourcename"])
			err = notificationStack.SendAllDataSource(fmt.Sprintf("dataset-%v-updated", updatetype), template, nil, true, "dataset-updated")
		} else {
			jobLog.Infof("Dispatching notification for new datasource")

			template["subject"] = fmt.Sprintf("Datasource %v Processing Complete", "")
			err = notificationStack.SendAllDataSource("new-dataset-available", template, nil, true, "")
		}
	}

	tsSaveErr := importtime.SaveDatasetImportUnixTimeSec(fs, jobLog, configBucket, datasetName, int(time.Now().Unix()))

	if tsSaveErr != nil {
		jobLog.Errorf(tsSaveErr.Error())
	}

	// Also write out
	if err != nil {
		jobLog.Errorf(err.Error())
	}
	return err
}

func connectMongo(ourLogger logger.ILogger) *mongo.Client {
	var mongoClient *mongo.Client
	var err error
	// Connect to mongo
	mongoSecret := os.Getenv("DB_SECRET_NAME")
	if len(mongoSecret) > 0 {
		// Remote is configured, connect to it
		mongoConnectionInfo, err := mongoDBConnection.GetMongoConnectionInfoFromSecretCache("pixlise/docdb/masteruser")
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
			err2 := fmt.Errorf("Failed connect to remote mongo: %v", err)
			ourLogger.Errorf("%v", err2)
			log.Fatalf("%v", err)
		}

	} else {
		// Connect to local mongo
		mongoClient, err = mongoDBConnection.ConnectToLocalMongoDB(ourLogger)
		if err != nil {
			err2 := fmt.Errorf("Failed connect to local mongo: %v", err)
			ourLogger.Errorf("%v", err2)
			log.Fatalf("%v", err)
		}
	}
	return mongoClient

}

func runTest(mongoclient *mongo.Client, notificationStack *apiNotifications.NotificationStack, record awsutil.Record) (string, error) {
	err := mongoclient.Ping(context.Background(), readpref.Primary())
	if err != nil {
		return "", err
	}

	template := make(map[string]interface{})
	template["datasourcename"] = "Test dataset name"
	template["subject"] = fmt.Sprintf("Datasource %v Processing Complete", "")
	err = notificationStack.SendAll("Test Datasource Email", template, []string{record.SNS.Message}, false)
	if err != nil {
		return "", err
	}
	return "", nil
}
