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

package endpoints

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/pixlUser"
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
