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
