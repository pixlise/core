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
	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/pixlUser"
)

// Testing querying and saving multi-quant z-stacks
const zstackS3Path = "UserContent/600f2a0806b6c70071d3d174/dataset123/" + filepaths.MultiQuantZStackFile
const zStackFile = `{
    "roiZStack": [
        {
            "roiID": "roi123",
            "quantificationID": "quantOne"
        },
        {
            "roiID": "roi456",
            "quantificationID": "quantTwo"
        }
    ]
}`

func Example_quantHandler_ZStackLoad() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(zstackS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(zstackS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(zstackS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(zstackS3Path),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("xSomething invalid"))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(zStackFile))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// File not in S3, should return empty
	req, _ := http.NewRequest("GET", "/quantification/combine-list/dataset123", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File in S3 empty, should return empty
	req, _ = http.NewRequest("GET", "/quantification/combine-list/dataset123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File contains invalid json, should return error
	req, _ = http.NewRequest("GET", "/quantification/combine-list/dataset123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File valid, should be OK
	req, _ = http.NewRequest("GET", "/quantification/combine-list/dataset123", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "roiZStack": []
	// }
	//
	// 200
	// {
	//     "roiZStack": []
	// }
	//
	// 500
	// invalid character 'x' looking for beginning of value
	//
	// 200
	// {
	//     "roiZStack": [
	//         {
	//             "roiID": "roi123",
	//             "quantificationID": "quantOne"
	//         },
	//         {
	//             "roiID": "roi456",
	//             "quantificationID": "quantTwo"
	//         }
	//     ]
	// }
}

// Super simple because we overwrite the file each time!
func Example_quantHandler_ZStackSave() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/datasetThatDoesntExist/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/datasetNoPermission/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/dataset123/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "datasetNoPermission",
				"group": "GroupA",
				"drive_id": 292,
				"site_id": 1,
				"target_id": "?",
				"sol": "0",
				"rtt": 456,
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
			 }`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
				"dataset_id": "dataset123",
				"group": "GroupB",
				"drive_id": 292,
				"site_id": 1,
				"target_id": "?",
				"sol": "0",
				"rtt": 456,
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
			 }`))),
		},
	}

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(zstackS3Path), Body: bytes.NewReader([]byte(zStackFile)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)

	mockUser := pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Permissions: map[string]bool{
			"access:GroupB": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}

	apiRouter := MakeRouter(svcs)

	// Invalid contents, fail
	req, _ := http.NewRequest("POST", "/quantification/combine-list/dataset123", bytes.NewReader([]byte("Something invalid")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Bad dataset ID, fail
	req, _ = http.NewRequest("POST", "/quantification/combine-list/datasetThatDoesntExist", bytes.NewReader([]byte(zStackFile)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// No permissions for dataset, fail
	req, _ = http.NewRequest("POST", "/quantification/combine-list/datasetNoPermission", bytes.NewReader([]byte(zStackFile)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Valid, success
	req, _ = http.NewRequest("POST", "/quantification/combine-list/dataset123", bytes.NewReader([]byte(zStackFile)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Request body invalid
	//
	// 400
	// datasetThatDoesntExist not found
	//
	// 400
	// dataset datasetNoPermission not permitted
	//
	// 200
}
