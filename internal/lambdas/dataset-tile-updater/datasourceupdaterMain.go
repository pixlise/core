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
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/core/awsutil"
	datasetModel "github.com/pixlise/core/core/dataset"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/logger"
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
