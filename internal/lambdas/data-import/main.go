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
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/pixlise/core/core/awsutil"
	"os"
)

type DatasourceEvent struct {
	Inpath         string `json:"inpath"`
	Rangespath     string `json:"rangespath"`
	Outpath        string `json:"outpath"`
	DatasetID      string `json:"datasetid"`
	DetectorConfig string `json:"detectorconfig"`
}

//var artifactDataSourceBucket = os.Getenv("DATASETS_BUCKET")

//var artifactPreProcessedBucket = os.Getenv("PREPROCESS_BUCKET") //"artifactsstack-artifactspreprocesseddatasourcespi-9h8o5px7rqk7"
//var artifactManualUploadBucket = os.Getenv("MANUAL_BUCKET") //"artifactsstack-artifactsmanualuploaddatasourcespi-1m9y4zu1x9vud"
//var configBucket = os.Getenv("CONFIG_BUCKET")

func getConfigBucket() string {
	return os.Getenv("CONFIG_BUCKET")
}

func getManualBucket() string {
	return os.Getenv("MANUAL_BUCKET")
}

func getDatasourceBucket() string {
	return os.Getenv("DATASETS_BUCKET")
}

var envBuckets = []string{
	"devstack-persistencepixlisedata4f446ecf-1corom7nbx3uv",
	"stagingstack-persistencepixlisedata4f446ecf-118o0uwwb176b",
	"prodstack-persistencepixlisedata4f446ecf-m36oehuca7uc",
}

//{
//  "inpath": "pixl.zip",
//  "rangespath": "configs/StandardPseudoIntensities.csv",
//  "outpath": "/tmp/",
//  "datasetid": "pixl_data_drive_dir_structure",
//  "detectorconfig": "PIXL"
//}
///

var tmpprefix = ""
var localUnzipPath = ""
var localInputPath = ""
var localArchivePath = ""
var localRangesCSVPath = ""

type StructKeys struct {
	Dir string
	Log string
}
type APISnsMessage struct {
	Key StructKeys `json:"datasetaddons"`
}

func HandleRequest(ctx context.Context, event awsutil.Event) (string, error) {
	setupLocalPaths()
	var makeLog = true
	// Init a logger for this job

	fmt.Printf("Unzip Path: %v \n", localUnzipPath)
	fmt.Printf("Input Path: %v \n", localInputPath)
	fmt.Printf("Archive Path: %v \n", localArchivePath)
	fmt.Printf("Ranges Path: %v \n", localRangesCSVPath)

	// If a targetbucket is defined it will copy the datasource to that bucket.
	// Otherwise it will use the envBuckets to seed the datasets.
	defer os.RemoveAll(tmpprefix)
	for _, record := range event.Records {
		if record.EventSource == "aws:s3" {
			return processS3(makeLog, record)
		} else if record.EventSource == "aws:sns" {
			return processSns(makeLog, record)
		}
	}
	return fmt.Sprintf("----- DONE -----\n"), nil
}

func main() {
	lambda.Start(HandleRequest)
}
