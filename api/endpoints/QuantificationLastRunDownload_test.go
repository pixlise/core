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

package endpoints

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
	"gitlab.com/pixlise/pixlise-go-api/core/pixlUser"
)

func Example_quant_LastRun_Stream_OK() {
	const summaryJSON = `{
   "dataset_id": "590340",
   "group": "groupie",
   "drive_id": 292,
   "site_id": 1,
   "target_id": "?",
   "sol": "0",
   "rtt": 590340,
   "sclk": 0,
   "context_image": "MCC-234.png",
   "location_count": 446,
   "data_file_size": 2699388,
   "context_images": 1,
   "normal_spectra": 882,
   "dwell_spectra": 0,
   "bulk_spectra": 2,
   "max_spectra": 2,
   "pseudo_intensities": 441,
   "detector_config": "PIXL"
}`

	quantResultBytes := []byte{60, 113, 117, 97, 110, 116, 62}

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/590340/LastPiquantOutput/quant/output_data.csv"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			ContentLength: aws.Int64(int64(len(quantResultBytes))),
			Body:          ioutil.NopCloser(bytes.NewReader(quantResultBytes)), // return some printable chars so easier to compare in Output comment
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Permissions: map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	// Should fail, datasets.json fails to download
	req, _ := http.NewRequest("GET", "/quantification/last/download/590340/quant/output", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	// Make sure the response is the right kind...
	fmt.Println(resp.HeaderMap["Content-Disposition"])
	fmt.Println(resp.HeaderMap["Cache-Control"])
	fmt.Println(resp.HeaderMap["Content-Length"])
	fmt.Println(resp.Body)

	// Should fail, invalid command
	req, _ = http.NewRequest("GET", "/quantification/last/download/590340/rpmcalc/output", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	// Make sure the response is the right kind...
	fmt.Println(resp.HeaderMap["Content-Disposition"])
	fmt.Println(resp.HeaderMap["Cache-Control"])
	fmt.Println(resp.HeaderMap["Content-Length"])
	fmt.Println(resp.Body)

	// Should fail, invalid file type
	req, _ = http.NewRequest("GET", "/quantification/last/download/590340/quant/runcost", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	// Make sure the response is the right kind...
	fmt.Println(resp.HeaderMap["Content-Disposition"])
	fmt.Println(resp.HeaderMap["Cache-Control"])
	fmt.Println(resp.HeaderMap["Content-Length"])
	fmt.Println(resp.Body)

	// Should succeed
	req, _ = http.NewRequest("GET", "/quantification/last/download/590340/quant/output", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	// Make sure the response is the right kind...
	fmt.Println(resp.HeaderMap["Content-Disposition"])
	fmt.Println(resp.HeaderMap["Cache-Control"])
	fmt.Println(resp.HeaderMap["Content-Length"])
	fmt.Println(resp.Body)

	// Output:
	// 404
	// []
	// []
	// []
	// 590340 not found
	//
	// 400
	// []
	// []
	// []
	// Invalid request
	//
	// 400
	// []
	// []
	// []
	// Invalid request
	//
	// 200
	// [attachment; filename="output_data.csv"]
	// [max-age=604800]
	// [7]
	// <quant>
}
