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
	"math/rand"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/awsutil"
)

func printMultiLineBody(body string) {
	lines := strings.Split(body, "\\n")
	for _, line := range lines {
		fmt.Println(line)
	}
}

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Multi-quant creation

func Example_quantHandler_MultiQuantCombine_SimpleFails() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "here's a name",
		"description": "combined quants",
		"roiZStack": []
	}`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "here's a name",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Name cannot be empty
	//
	// 400
	// Must reference more than 1 ROI
	//
	// 400
	// Must reference more than 1 ROI
}

const multiCombineJobSummary = `{
	"job1": {
		"shared": false,
		"params": {
			"pmcsCount": 313,
			"name": "in progress",
			"dataBucket": "dev-pixlise-data",
			"datasetPath": "Datasets/rtt-456/dataset.bin",
			"datasetID": "rtt-456",
			"jobBucket": "dev-pixlise-piquant-jobs",
			"detectorConfig": "PIXL",
			"elements": [
				"Sc",
				"Cr"
			],
			"parameters": "-q,pPIETXCFsr -b,0,12,60,910,2800,16",
			"runTimeSec": 60,
			"coresPerNode": 6,
			"startUnixTime": 1592968112,
			"creator": {
				"name": "niko belic",
				"user_id": "600f2a0806b6c70071d3d174",
				"email": ""
			},
			"roiID": "",
			"elementSetID": "",
			"piquantVersion": "3.0.3",
			"quantMode": "Combined"
		},
		"elements": [
			"Sc",
			"Cr"
		],
		"jobId": "job1",
		"status": "nodes_running",
		"message": "Node count: 1, Files/Node: 1770",
		"endUnixTime": 1592968196,
		"outputFilePath": "",
		"piquantLogList": null
	}
}`

const multiCombineUserROIs = `{
	"roi-first": {
		"name": "1st ROI",
		"description": "1st",
		"locationIndexes": [23, 29],
		"creator": { "name": "Peter", "user_id": "u123" }
	},
	"roi-second": {
		"name": "Second ROI",
		"locationIndexes": [22, 25, 29],
		"creator": { "name": "Tom", "user_id": "u124" }
	}
}`

const multiCombineSharedROIs = `{
	"roi-third": {
		"name": "Third ROI (shared)",
		"description": "The third one",
		"locationIndexes": [23, 26],
		"creator": { "name": "Peter", "user_id": "u123" }
	}
}`

func prepCombineGetCalls(quant2File string) ([]s3.GetObjectInput, []*s3.GetObjectOutput) {
	dsbytes, err := ioutil.ReadFile("./test-data/dataset.bin")
	fmt.Printf("dataset %v\n", err)

	q123bytes, err := ioutil.ReadFile("./test-data/combined.bin")
	fmt.Printf("quant1 %v\n", err)

	q456bytes, err := ioutil.ReadFile("./test-data/" + quant2File)
	fmt.Printf("quant2 %v\n", err)

	getRequests := []s3.GetObjectInput{
		// First retrieves the quant summaries that came from uniqueness check, to read the quant name
		{
			Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobSummaries/dataset-123-jobs.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/dataset-123/dataset.bin"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/ROI.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/dataset-123/ROI.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/quant-123.bin"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/quant-456.bin"),
		},
	}

	getResponses := []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(multiCombineJobSummary))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader(dsbytes)),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(multiCombineUserROIs))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(multiCombineSharedROIs))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader(q123bytes)),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader(q456bytes)),
		},
	}

	return getRequests, getResponses
}

func Example_quantHandler_MultiQuantCombine_DuplicateNameWithInProgressQuant() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		// First retrieves the quant summaries that came from uniqueness check, to read the quant name
		{
			Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobSummaries/dataset-123-jobs.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(multiCombineJobSummary))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "in progress",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "roi-second",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "roi-third",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Name already used: in progress
}

func Example_quantHandler_MultiQuantCombine_DatasetFailsToLoad() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
		/*		{
				Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/shared/dataset-123/Quantifications/summary-"),
			},*/
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
		/*		{
				Contents: []*s3.Object{
					{Key: aws.String("UserContent/shared/dataset-123/Quantifications/summary-job2.json")},
				},
			},*/
	}

	getRequests, getResponses := prepCombineGetCalls("combined-3elem.bin")
	getResponses[2] = nil // Dataset bin file

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "new multi",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "roi-second",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "shared-roi-third",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// dataset <nil>
	// quant1 <nil>
	// quant2 <nil>
	// 404
	// Failed to download dataset: NoSuchKey: Returning error from GetObject
}

func Example_quantHandler_MultiQuantCombine_CombineIncompatible() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
		/*		{
				Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/shared/dataset-123/Quantifications/summary-"),
			},*/
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
		/*		{
				Contents: []*s3.Object{
					{Key: aws.String("UserContent/shared/dataset-123/Quantifications/summary-job2.json")},
				},
			},*/
	}

	getRequests, getResponses := prepCombineGetCalls("AB.bin")

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "new multi",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "roi-second",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "shared-roi-third",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// dataset <nil>
	// quant1 <nil>
	// quant2 <nil>
	// 400
	// Detectors don't match other quantifications: quant-456
}

func Example_quantHandler_MultiQuantCombine_UserROIFailsToLoad() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
		/*		{
				Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/shared/dataset-123/Quantifications/summary-"),
			},*/
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
		/*		{
				Contents: []*s3.Object{
					{Key: aws.String("UserContent/shared/dataset-123/Quantifications/summary-job2.json")},
				},
			},*/
	}

	getRequests, getResponses := prepCombineGetCalls("combined-3elem.bin")
	getResponses[3] = nil // User ROI load fail

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "new multi",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "roi-second",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "shared-roi-third",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	printMultiLineBody(resp.Body.String())

	// Output:
	// dataset <nil>
	// quant1 <nil>
	// quant2 <nil>
	// 404
	// Failed to get all ROIs: Failed to find ROI ID: roi-first
}

func Example_quantHandler_MultiQuantCombine_QuantFailsToLoad() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
		/*		{
				Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/shared/dataset-123/Quantifications/summary-"),
			},*/
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
		/*		{
				Contents: []*s3.Object{
					{Key: aws.String("UserContent/shared/dataset-123/Quantifications/summary-job2.json")},
				},
			},*/
	}

	getRequests, getResponses := prepCombineGetCalls("combined-3elem.bin")
	getResponses[5] = nil // First quant load fail

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "new multi",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "roi-second",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "shared-roi-third",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	printMultiLineBody(resp.Body.String())

	// Output:
	// dataset <nil>
	// quant1 <nil>
	// quant2 <nil>
	// 404
	// Failed to download quant quant-123: NoSuchKey: Returning error from GetObject
}

func Example_quantHandler_MultiQuantCombine_ROINotFound() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
		/*		{
				Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/shared/dataset-123/Quantifications/summary-"),
			},*/
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
		/*		{
				Contents: []*s3.Object{
					{Key: aws.String("UserContent/shared/dataset-123/Quantifications/summary-job2.json")},
				},
			},*/
	}

	getRequests, getResponses := prepCombineGetCalls("combined-3elem.bin")
	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "new multi",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "non-existant-roi",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "shared-roi-third",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	printMultiLineBody(resp.Body.String())

	// Output:
	// dataset <nil>
	// quant1 <nil>
	// quant2 <nil>
	// 404
	// Failed to get all ROIs: Failed to find ROI ID: non-existant-roi
}

func Example_quantHandler_MultiQuantCombine_OK() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
	}

	getRequests, getResponses := prepCombineGetCalls("combined-3elem.bin")
	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	expBinBytes := []byte{10, 8, 108, 105, 118, 101, 116, 105, 109, 101, 10, 5, 67, 97, 79, 95, 37, 10, 7, 67, 97, 79, 95, 101, 114, 114, 10, 7, 70, 101, 79, 45, 84, 95, 37, 10, 9, 70, 101, 79, 45, 84, 95, 101, 114, 114, 10, 6, 83, 105, 79, 50, 95, 37, 10, 8, 83, 105, 79, 50, 95, 101, 114, 114, 10, 6, 84, 105, 79, 50, 95, 37, 10, 8, 84, 105, 79, 50, 95, 101, 114, 114, 18, 9, 1, 0, 0, 0, 0, 0, 0, 0, 0, 26, 192, 2, 10, 8, 67, 111, 109, 98, 105, 110, 101, 100, 18, 60, 8, 97, 42, 0, 42, 5, 21, 129, 38, 236, 64, 42, 5, 21, 205, 204, 204, 62, 42, 5, 21, 163, 82, 28, 66, 42, 5, 21, 0, 0, 0, 64, 42, 5, 21, 0, 0, 128, 191, 42, 5, 21, 0, 0, 128, 191, 42, 5, 21, 90, 100, 219, 62, 42, 5, 21, 205, 204, 76, 62, 18, 60, 8, 98, 42, 0, 42, 5, 21, 251, 92, 149, 64, 42, 5, 21, 154, 153, 153, 62, 42, 5, 21, 64, 19, 81, 65, 42, 5, 21, 51, 51, 51, 63, 42, 5, 21, 120, 122, 237, 65, 42, 5, 21, 0, 0, 192, 63, 42, 5, 21, 25, 4, 6, 63, 42, 5, 21, 205, 204, 76, 62, 18, 60, 8, 100, 42, 0, 42, 5, 21, 204, 238, 57, 64, 42, 5, 21, 0, 0, 0, 63, 42, 5, 21, 26, 81, 99, 65, 42, 5, 21, 51, 51, 51, 63, 42, 5, 21, 0, 0, 128, 191, 42, 5, 21, 0, 0, 128, 191, 42, 5, 21, 182, 243, 13, 63, 42, 5, 21, 205, 204, 76, 62, 18, 60, 8, 101, 42, 0, 42, 5, 21, 27, 158, 118, 64, 42, 5, 21, 0, 0, 0, 63, 42, 5, 21, 37, 117, 192, 65, 42, 5, 21, 154, 153, 153, 63, 42, 5, 21, 205, 59, 22, 66, 42, 5, 21, 51, 51, 243, 63, 42, 5, 21, 141, 151, 174, 62, 42, 5, 21, 205, 204, 76, 62, 18, 60, 8, 104, 42, 0, 42, 5, 21, 50, 119, 183, 64, 42, 5, 21, 154, 153, 153, 62, 42, 5, 21, 239, 201, 5, 65, 42, 5, 21, 205, 204, 204, 62, 42, 5, 21, 42, 105, 2, 66, 42, 5, 21, 205, 204, 204, 63, 42, 5, 21, 86, 125, 174, 62, 42, 5, 21, 205, 204, 76, 62}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/multi_combquant123.bin"), Body: bytes.NewReader(expBinBytes),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/multi_combquant123.csv"), Body: bytes.NewReader([]byte(`Combined multi-quantification from quant-123, quant-456
PMC, RTT, SCLK, filename, livetime, CaO_%, CaO_err, FeO-T_%, FeO-T_err, SiO2_%, SiO2_err, TiO2_%, TiO2_err
97, 0, 0, Normal_Combined_roi-second, 0, 7.379700183868408, 0.4000000059604645, 39.0806999206543, 2, -1, -1, 0.4284999966621399, 0.20000000298023224
98, 0, 0, Normal_Combined_roi-first, 0, 4.667600154876709, 0.30000001192092896, 13.06719970703125, 0.699999988079071, 29.684799194335938, 1.5, 0.5235000252723694, 0.20000000298023224
100, 0, 0, Normal_Combined_roi-second, 0, 2.9052000045776367, 0.5, 14.207300186157227, 0.699999988079071, -1, -1, 0.5544999837875366, 0.20000000298023224
101, 0, 0, Normal_Combined_shared-roi-third, 0, 3.8533999919891357, 0.5, 24.057199478149414, 1.2000000476837158, 37.55839920043945, 1.899999976158142, 0.3409999907016754, 0.20000000298023224
104, 0, 0, Normal_Combined_roi-first, 0, 5.73330020904541, 0.30000001192092896, 8.361800193786621, 0.4000000059604645, 32.602699279785156, 1.600000023841858, 0.3407999873161316, 0.20000000298023224
`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-multi_combquant123.json"), Body: bytes.NewReader([]byte(`{
    "shared": false,
    "params": {
        "pmcsCount": 0,
        "name": "new multi",
        "dataBucket": "datasets-bucket",
        "datasetPath": "Datasets/dataset-123/dataset.bin",
        "datasetID": "dataset-123",
        "jobBucket": "job-bucket",
        "detectorConfig": "",
        "elements": [
            "CaO",
            "FeO-T",
            "SiO2",
            "TiO2"
        ],
        "parameters": "",
        "runTimeSec": 0,
        "coresPerNode": 0,
        "startUnixTime": 4234567890,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "roiID": "",
        "elementSetID": "",
        "piquantVersion": "N/A",
        "quantMode": "CombinedMultiQuant",
        "comments": "combined quants",
        "roiIDs": [],
        "command": "map"
    },
    "elements": [
        "CaO",
        "FeO-T",
        "SiO2",
        "TiO2"
    ],
    "jobId": "multi_combquant123",
    "status": "complete",
    "message": "combined-multi quantification processed",
    "endUnixTime": 4234567890,
    "outputFilePath": "UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications",
    "piquantLogList": []
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
	}

	mockS3.AllowGetInAnyOrder = true

	var idGen MockIDGenerator
	idGen.ids = []string{"combquant123"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
		QueuedTimeStamps: []int64{4234567890},
	}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "new multi",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "roi-second",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "shared-roi-third",
				"quantificationID": "quant-123"
			}
		]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	printMultiLineBody(resp.Body.String())

	// Output:
	// dataset <nil>
	// quant1 <nil>
	// quant2 <nil>
	// 200
	// "multi_combquant123"
}

func Example_quantHandler_MultiQuantCombine_SummaryOnly_OK() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Listing quants for name uniqueness check
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-job1.json")},
			},
		},
	}

	getRequests, getResponses := prepCombineGetCalls("combined-3elem.bin")
	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true

	var idGen MockIDGenerator
	idGen.ids = []string{"combquant123"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
		QueuedTimeStamps: []int64{4234567890},
	}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/combine/dataset-123", bytes.NewReader([]byte(`{
		"name": "new multi",
		"description": "combined quants",
		"roiZStack": [
			{
				"roiID": "roi-first",
				"quantificationID": "quant-123"
			},
			{
				"roiID": "roi-second",
				"quantificationID": "quant-456"
			},
			{
				"roiID": "shared-roi-third",
				"quantificationID": "quant-123"
			}
		],
		"summaryOnly": true
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	printMultiLineBody(resp.Body.String())

	// Output:
	// dataset <nil>
	// quant1 <nil>
	// quant2 <nil>
	// 200
	// {
	//     "detectors": [
	//         "Combined"
	//     ],
	//     "weightPercents": {
	//         "CaO": {
	//             "values": [
	//                 0.10906311
	//             ],
	//             "roiIDs": [
	//                 "roi-first",
	//                 "roi-second",
	//                 "shared-roi-third"
	//             ],
	//             "roiNames": [
	//                 "1st ROI",
	//                 "Second ROI",
	//                 "Third ROI (shared)"
	//             ]
	//         },
	//         "FeO-T": {
	//             "values": [
	//                 0.43899643
	//             ],
	//             "roiIDs": [
	//                 "roi-first",
	//                 "roi-second",
	//                 "shared-roi-third"
	//             ],
	//             "roiNames": [
	//                 "1st ROI",
	//                 "Second ROI",
	//                 "Third ROI (shared)"
	//             ]
	//         },
	//         "SiO2": {
	//             "values": [
	//                 0.44375953
	//             ],
	//             "roiIDs": [
	//                 "roi-first",
	//                 "shared-roi-third"
	//             ],
	//             "roiNames": [
	//                 "1st ROI",
	//                 "Third ROI (shared)"
	//             ]
	//         },
	//         "TiO2": {
	//             "values": [
	//                 0.009725777
	//             ],
	//             "roiIDs": [
	//                 "roi-first",
	//                 "roi-second",
	//                 "shared-roi-third"
	//             ],
	//             "roiNames": [
	//                 "1st ROI",
	//                 "Second ROI",
	//                 "Third ROI (shared)"
	//             ]
	//         }
	//     }
	// }
}
