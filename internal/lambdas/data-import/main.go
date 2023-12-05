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
	"github.com/pixlise/core/v3/api/dataimport"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/mongoDBConnection"
)

func HandleRequest(ctx context.Context, event awsutil.Event) (string, error) {
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
	logger := &logger.StdOutLogger{}
	for _, record := range event.Records {
		mongoClient, err := mongoDBConnection.Connect(sess, mongoSecret, logger)
		if err != nil {
			log.Fatal(err)
		}

		// Get handle to the DB
		dbName := mongoDBConnection.GetDatabaseName("pixlise", envName)
		db := mongoClient.Database(dbName)

		// Print this to stdout - not that useful, won't be in the log file, but lambda cloudwatch log should have it
		// and it'll be useful for initial debugging
		fmt.Printf("ImportForTrigger: \"%v\"\n", record.SNS.Message)

		result, err := dataimport.ImportForTrigger([]byte(record.SNS.Message), envName, configBucket, datasetBucket, manualBucket, db, logger, remoteFS)
		if err != nil {
			return "", err
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

func main() {
	os.Mkdir("/tmp/profile", 0750)
	lambda.Start(HandleRequest)
}
