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
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pixlise/core/v4/api/dataimport"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/utils"
)

func HandleRequest(ctx context.Context, event awsutil.Event) (string, error) {
	fmt.Printf("Data Importer Lambda version: %v\n", services.ApiVersion)

	configBucket := os.Getenv("CONFIG_BUCKET")
	datasetBucket := os.Getenv("DATASETS_BUCKET")
	manualBucket := os.Getenv("MANUAL_BUCKET")
	envName := os.Getenv("ENVIRONMENT_NAME")
	mongoSecret := os.Getenv("DB_SECRET_NAME") // Used to be hard coded to: "pixlise/docdb/masteruser"

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

	// Turn off date+time prefixing of log msgs, we have timestamps captured in other ways
	log.SetFlags(0)

	// Normally we'd only expect event.Records to be of length 1...
	worked := 0
	iLog := &logger.StdOutLogger{}
	iLog.SetLogLevel(logger.LogInfo)

	for _, record := range event.Records {
		// Print this to stdout - not that useful, won't be in the log file, but lambda cloudwatch log should have it
		// and it'll be useful for initial debugging
		fmt.Printf("ImportForTrigger: \"%v\"\n", record.SNS.Message)

		wd, err := os.Getwd()
		if err != nil {
			fmt.Printf("Failed to get working dir: %v\n", err)
		} else {
			fmt.Printf("Working dir: %v\n", wd)

			err = os.Chdir(os.TempDir())
			if err != nil {
				fmt.Printf("Failed to change to temp dir: %v\n", err)
			}

			freeBytes, err := utils.GetDiskAvailableBytes()
			if err != nil {
				fmt.Printf("Failed to read disk free space: %v\n", err)
			} else {
				fmt.Printf("Disk free space: %v\n", freeBytes)
			}

			err = os.Chdir(wd)
			if err != nil {
				fmt.Printf("Failed to change to working dir: %v\n", err)
			}
		}

		mongoClient, _, err := mongoDBConnection.Connect(sess, mongoSecret, iLog)
		if err != nil {
			log.Fatal(err)
		}

		// Get handle to the DB
		dbName := mongoDBConnection.GetDatabaseName("pixlise", envName)
		db := mongoClient.Database(dbName)

		result, err := dataimport.ImportForTrigger([]byte(record.SNS.Message), configBucket, datasetBucket, manualBucket, db, iLog, remoteFS)
		if err != nil {

			if len(result.WorkingDir) > 0 {
				clearWorkingDir(result.WorkingDir)
			}

			return "", err
		}

		clearWorkingDir(result.WorkingDir)

		// TODO: Send some kind of SNS notification to the API so it can directly show notifications to connected user sessions
		// What do we do about multiple API instances though? Should all APIs receive it?

		if err != nil {
			return "", err
		} else {
			worked++
		}
	}

	return fmt.Sprintf("Imported %v records", worked), nil
}

func clearWorkingDir(workingDir string) {
	// Delete the working directory here, there's no point leaving it on a lambda machine, we can't debug it
	// but if this code ran elsewhere we wouldn't delete it, to have something to look at
	if len(workingDir) > 0 {
		removeErr := os.RemoveAll(workingDir)
		if removeErr != nil {
			fmt.Printf("Failed to remove working dir: \"%v\". Error: %v\n", workingDir, removeErr)
		} else {
			fmt.Printf("Removed working dir: \"%v\"\n", workingDir)
		}
	}
}

func main() {
	os.Mkdir("/tmp/profile", 0750) // Not sure what this is for, permissions are read/executable for owner/group. Perhaps some profiling tool Tom used a while back
	lambda.Start(HandleRequest)
}
