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
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"gitlab.com/pixlise/pixlise-go-api/api/filepaths"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	datasetModel "gitlab.com/pixlise/pixlise-go-api/core/dataset"
	"gitlab.com/pixlise/pixlise-go-api/core/fileaccess"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
)

func updateDatasets(fs fileaccess.FileAccess, s3Bucket string, log logger.ILogger) error {
	log.Infof("Requesting file listing from: %v", s3Bucket)

	allPaths, err := fs.ListObjects(s3Bucket, filepaths.RootDatasets+"/")
	if err != nil {
		return err
	}

	summaryPaths := []string{}
	for _, k := range allPaths {
		if strings.HasSuffix(k, filepaths.DatasetSummaryFileName) {
			summaryPaths = append(summaryPaths, k)
		}
	}

	log.Infof("Got %v paths. Requesting %v summary files...", len(allPaths), len(summaryPaths))

	summaries := []datasetModel.SummaryFileData{}
	for _, k := range summaryPaths {
		var summary datasetModel.SummaryFileData
		err := fs.ReadJSON(s3Bucket, k, &summary, false)
		if err != nil {
			log.Errorf("Failed to read dataset summary %v: %v", s3Bucket, err)
			continue
		}
		summaries = append(summaries, summary)
	}

	mapped := datasetModel.DatasetConfig{
		Datasets: summaries,
	}

	// Done without a call to WriteJSON because we have less indentation here... mainly as a space save
	fileContents, err := json.MarshalIndent(mapped, "", " ")
	if err != nil {
		return err
	}

	datasetsPath := filepaths.GetDatasetListPath()
	log.Infof("Returning data to %v %v. List of dataset IDs:", s3Bucket, datasetsPath)
	for c, summary := range summaries {
		log.Infof("  %v: %v", c+1, summary.DatasetID)
	}
	return fs.WriteObject(s3Bucket, datasetsPath, fileContents)
}

func handler(ctx context.Context, s3Event events.S3Event) error {
	stdLog := logger.StdOutLogger{}
	errCount := 0

	for _, record := range s3Event.Records {
		s3ev := record.S3
		bucket := s3ev.Bucket.Name
		//fmt.Printf("[%s - %s] Bucket = %s, Key = %s \n", record.EventSource, record.EventTime, s3.Bucket.Name, s3.Object.Key)

		sess, err := awsutil.GetSession()
		if err != nil {
			return err
		}
		s3svc, err := awsutil.GetS3(sess)
		if err != nil {
			return err
		}

		fs := fileaccess.MakeS3Access(s3svc)
		err = updateDatasets(fs, bucket, stdLog)

		if err != nil {
			// Don't stop here!
			stdLog.Errorf("updateDatasets FAILED for bucket: s3://%v. Error: %v.", bucket, err)
			errCount++
		}
	}

	if errCount > 0 {
		return fmt.Errorf("updateDatasets failed for %v paths", errCount)
	}

	return nil
}

func main() {
	rand.Seed(time.Now().UnixNano())
	lambda.Start(handler)
	/*
		sess, _ := awsutil.GetSession()
		s3svc, _ := awsutil.GetS3(sess)
		fs := fileaccess.MakeS3Access(s3svc)
		stdLog := logger.StdOutLogger{}
		updateDatasets(fs, "/prodstack-persistencepixlisedata4f446ecf-m36oehuca7uc", stdLog)
	*/
}
