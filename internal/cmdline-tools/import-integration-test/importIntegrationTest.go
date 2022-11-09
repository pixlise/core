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
	"flag"
	"fmt"
	"log"
	"math/rand"
	"os"
	"path"
	"time"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/fileaccess"
)

const testDatasetID = "000000001"                          // an ID (RTT) that wouldn't ever come from OCS
const zipName = testDatasetID + "-21-10-2022-15-37-00.zip" // Found in ./test-data/

func main() {
	rand.Seed(time.Now().UnixNano())

	var rawBucket = flag.String("raw_bucket", "", "Raw bucket that we simulate zip files landing in to trigger import")
	var datasetBucket = flag.String("dataset_bucket", "", "Dataset bucket where we expect files to appear")

	flag.Parse()
	/*
		if len(os.Args) != 3 {
			fmt.Println("Arguments: raw_bucket, dataset_bucket")
			fmt.Println("  Where:")
			//fmt.Println("  - environment name is one of [dev, staging, prod] OR a review environment name (eg review-env-blah, so without -api.review at the end)")
			fmt.Println("  - raw_bucket - ")
			fmt.Println("  - dataset_bucket - ")
			fmt.Println("NOTE: Environment variables for accessing S3 must be configured too!")
			os.Exit(1)
		}

		// Check arguments
		//var environment = os.Args[1]

		//fmt.Println("Running integration test for env: " + environment)

		var rawBucket = os.Args[1]
		var datasetBucket = os.Args[2]
	*/
	if len(*rawBucket) <= 0 {
		log.Fatalln("raw_bucket not specified")
	}
	if len(*datasetBucket) <= 0 {
		log.Fatalln("dataset_bucket not specified")
	}

	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("%v\n", err)
		os.Exit(1)
	}

	remoteFS := fileaccess.MakeS3Access(svc)
	localFS := fileaccess.FSAccess{}

	// Dataset generation process that we're testing:
	// 1. Zip file appears in raw bucket
	// 2. This should trigger an SNS message to importer lambda
	// 3. Lambda runs and generates a dataset
	// 4. Copies zip file to datasets bucket /Archive
	// 5. Copies dataset files to datasets bucket /Datasets
	// 6. Copies dataset summary to datasets bucket /DatasetSummaries

	// We trigger the process by putting a zip into raw bucket, then ensure
	// the other steps happen, finally, clear out any evidence that we ran

	showSubHeading("Setting up for test...")

	err = clearTestDatasets(remoteFS, *rawBucket, *datasetBucket, testDatasetID)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	// Copy a zip file into raw data bucket
	zipData, err := localFS.ReadObject("./test-data", zipName)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	showSubHeading("Triggering dataset import...")

	err = remoteFS.WriteObject(*rawBucket, zipName, zipData)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	showSubHeading("Waiting before checking it worked...")

	// Now we poll for the dataset files to appear. If they don't in the timeout period
	// we give up and say it failed
	timeoutSec := 300
	pollIntervalSec := 10
	pollStartSec := 30 // wait this long before starting polling

	sleepTimeSec := pollStartSec
	timeSleptSec := 0

	datasetsPath := path.Join(filepaths.RootDatasets, testDatasetID)
	filesFound := []string{}

	for timeSleptSec < timeoutSec {
		time.Sleep(time.Duration(sleepTimeSec) * time.Second)

		// Poll for files
		fmt.Println("  Checking dataset files for " + testDatasetID + " exist...")
		filesFound, err = remoteFS.ListObjects(*datasetBucket, datasetsPath)
		if err != nil {
			log.Fatalf("Failed to poll for dataset creation finished\n")
		}

		// If we have at least one file...
		if len(filesFound) > 0 {
			fmt.Println("  Dataset files detected, checking import complete...")
			break
		}

		timeSleptSec += sleepTimeSec
		sleepTimeSec = pollIntervalSec
	}

	// Now that we found something, sleep a bit more just in case it's still writing out some files
	time.Sleep(time.Duration(2) * time.Second)

	// Ensure we have all the files we think should exist
	name := printTestStart("Check dataset created")

	// At this point:
	// - The zip file should be in archive
	archivePath := path.Join(filepaths.RootArchive, zipName)
	exists, err := remoteFS.ObjectExists(*datasetBucket, archivePath)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	if !exists {
		err = fmt.Errorf("File not found: %v", archivePath)
	} else {
		// - Dataset files should exist
		expDatasetFiles := map[string]bool{
			"Datasets/000000001/PCW_0138_0679216324_000RCM_N00518120000000010077075J02.png": true,
			"Datasets/000000001/PCW_0138_0679216527_000RCM_N00518120000000010078075J02.png": true,
			"Datasets/000000001/PCW_0138_0679223931_000RCM_N00518120000000010304075J02.png": true,
			"Datasets/000000001/PCW_0138_0679224147_000RCM_N00518120000000010306075J02.png": true,
			"Datasets/000000001/dataset.bin":                                                true,
			"Datasets/000000001/diffraction-db.bin":                                         true,
			"Datasets/000000001/summary.json":                                               true,
		}

		// We expect there to be a dataset.bin, diffraction-db.bin and summary.json at least!
		foundCount := 0
		for _, fileName := range filesFound {
			_, isExp := expDatasetFiles[fileName]
			if !isExp {
				err = fmt.Errorf("Unexpected dataset file found: %v", fileName)
			} else {
				foundCount++
			}
		}

		if foundCount != len(expDatasetFiles) {
			err = fmt.Errorf("Not all expected dataset output were found")
		} else {
			// - Dataset summary should exist
			summaryPath := filepaths.GetDatasetSummaryFilePath(testDatasetID)
			exists, err := remoteFS.ObjectExists(*datasetBucket, summaryPath)
			if err != nil {
				log.Fatalf("%v\n", err)
			}

			if !exists {
				err = fmt.Errorf("File not found: %v", archivePath)
			}
		}
	}

	printTestResult(err, name)

	/*
		// Wait for all
		fmt.Println("\n---------------------------------------------------------")
		now := time.Now().Format(timeFormat)
		fmt.Printf(" %v  STARTING quantifications, will wait for them to complete...\n", now)
		fmt.Printf("---------------------------------------------------------\n\n")
	*/

	showSubHeading("Clearing generated dataset...")
	err = clearTestDatasets(remoteFS, *rawBucket, *datasetBucket, testDatasetID)
	if err != nil {
		log.Fatalf("%v\n", err)
	}

	fmt.Println("\n==============================")

	if len(failedTestNames) == 0 {
		fmt.Println("PASSED All Tests!")
		os.Exit(0)
	}

	fmt.Println("FAILED One or more tests:")
	for _, name := range failedTestNames {
		fmt.Printf("- %v\n", name)
	}
	os.Exit(1)
}

func clearTestDatasets(fs fileaccess.FileAccess, rawBucket string, datasetBucket string, datasetID string) error {
	// Delete from raw bucket
	err := deleteFile(fs, rawBucket, zipName)
	if err != nil {
		return err
	}

	// Delete from archive
	archivePath := path.Join(filepaths.RootArchive, zipName)
	err = deleteFile(fs, datasetBucket, archivePath)
	if err != nil {
		return err
	}

	// Delete any files from dataset bucket area
	// We get a listing just in case we introduce/change file names and they get stranded by some version in future
	datasetsPath := path.Join(filepaths.RootDatasets, testDatasetID)
	files, err := fs.ListObjects(datasetBucket, datasetsPath)

	// Listing shouldn't fail if there's nothing there
	if err != nil {
		return fmt.Errorf("Failed to list files in s3://%v/%v", datasetBucket, datasetsPath)
	}

	for _, file := range files {
		// Delete each file
		err := deleteFile(fs, datasetBucket, file)
		if err != nil {
			return err
		}
	}

	// Delete dataset summary file
	summaryPath := filepaths.GetDatasetSummaryFilePath(datasetID)
	err = deleteFile(fs, datasetBucket, summaryPath)
	if err != nil {
		return err
	}

	return nil
}
