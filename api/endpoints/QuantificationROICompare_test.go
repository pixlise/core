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
	"math/rand"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/quantModel"
)

//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Multi-quant comparison endpoint

func Example_calculateTotals_Fail_AB() {
	q, err := quantModel.ReadQuantificationFile("./test-data/AB.bin")
	fmt.Printf("%v\n", err)

	result, err := calculateTotals(q, []int{90, 91, 95})

	fmt.Printf("%v|%v\n", result, err)

	// Output:
	// <nil>
	// map[]|Quantification must be for Combined detectors
}

func Example_calculateTotals_Fail_NoPMC() {
	q, err := quantModel.ReadQuantificationFile("./test-data/combined.bin")
	fmt.Printf("%v\n", err)

	result, err := calculateTotals(q, []int{68590, 68591, 68595})

	fmt.Printf("%v|%v\n", result, err)

	// Output:
	// <nil>
	// map[]|Quantification had no valid data for ROI PMCs
}

func Example_calculateTotals_Success() {
	q, err := quantModel.ReadQuantificationFile("./test-data/combined.bin")
	fmt.Printf("%v\n", err)

	result, err := calculateTotals(q, []int{90, 91, 95})

	fmt.Printf("%v|%v\n", result, err)

	// Output:
	// <nil>
	// map[CaO_%:7.5057006 FeO-T_%:10.621034 SiO2_%:41.48377 TiO2_%:0.7424]|<nil>
}

func Example_quantHandler_Comparison_FailReqBody() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{quantIDs":["quant-345", "quant-789"]}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{"quantIDs":[]}`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Request body invalid
	//
	// 400
	// Requested with 0 quant IDs
}

func Example_quantHandler_Comparison_FailRemainingPointsCheck() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{"quantIDs":["quant-345", "quant-789"], "remainingPointsPMCs": [4,6,88]}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	req, _ = http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/RemainingPoints", bytes.NewReader([]byte(`{"quantIDs":["quant-345", "quant-789"]}`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 400
	// Unexpected PMCs supplied for ROI: roi-567
	//
	// 400
	// No PMCs supplied for RemainingPoints ROI
}

func prepROICompareGetCalls() ([]s3.GetObjectInput, []*s3.GetObjectOutput) {
	dsbytes, err := ioutil.ReadFile("./test-data/dataset.bin")
	fmt.Printf("dataset.bin %v\n", err)

	qbytes, err := ioutil.ReadFile("./test-data/combined.bin")
	fmt.Printf("combined.bin %v\n", err)

	s3GetRequests := []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/ROI.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/dataset-123/dataset.bin"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/quant-345.bin"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-quant-345.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/quant-789.bin"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/dataset-123/Quantifications/summary-quant-789.json"),
		},
	}

	s3GetResponses := []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"roi-567": {
		"name": "Dark patch 2",
		"description": "The second dark patch",
		"locationIndexes": [15, 20],
		"creator": { "name": "Peter", "user_id": "u123" }
	},
	"roi-772": {
		"name": "White spot",
		"locationIndexes": [14, 5, 94],
		"creator": { "name": "Tom", "user_id": "u124" }
	}
	}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader(dsbytes)),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader(qbytes)),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"shared": false,
	"params": {
		"pmcsCount": 225,
		"name": "quant test v6 Al \u003e 100%",
		"dataBucket": "devstack-persistencepixlisedata4f446ecf-1corom7nbx3uv",
		"datasetPath": "Datasets/052822532/dataset.bin",
		"datasetID": "052822532",
		"jobBucket": "devstack-persistencepiquantjobs65c7175e-1dg51nw1ye1rk",
		"detectorConfig": "PIXL/PiquantConfigs/v6",
		"elements": [
			"Ca",
			"Ti",
			"Fe",
			"Si"
		],
		"parameters": "",
		"runTimeSec": 60,
		"coresPerNode": 4,
		"startUnixTime": 1629335356,
		"creator": {
			"name": "peternemere",
			"user_id": "5de45d85ca40070f421a3a34",
			"email": "peternemere@gmail.com"
		},
		"roiID": "",
		"elementSetID": "",
		"piquantVersion": "registry.github.com/pixlise/piquant/runner:3.2.8-ALPHA",
		"quantMode": "Combined",
		"comments": ""
	},
	"elements": [
		"CaO",
		"TiO2",
		"FeO-T",
		"SiO2"
	],
	"jobId": "quant-345",
	"status": "complete",
	"message": "Nodes ran: 4",
	"endUnixTime": 1629335518,
	"outputFilePath": "UserContent/5de45d85ca40070f421a3a34/052822532/Quantifications",
	"piquantLogList": [
		"node00001_piquant.log",
		"node00001_stdout.log",
		"node00002_piquant.log",
		"node00002_stdout.log",
		"node00003_piquant.log",
		"node00003_stdout.log",
		"node00004_piquant.log",
		"node00004_stdout.log"
	]
	}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader(qbytes)),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"shared": false,
	"params": {
		"pmcsCount": 225,
		"name": "quant test v6 Al \u003e 100%",
		"dataBucket": "devstack-persistencepixlisedata4f446ecf-1corom7nbx3uv",
		"datasetPath": "Datasets/052822532/dataset.bin",
		"datasetID": "052822532",
		"jobBucket": "devstack-persistencepiquantjobs65c7175e-1dg51nw1ye1rk",
		"detectorConfig": "PIXL/PiquantConfigs/v6",
		"elements": [
			"Ca",
			"Ti",
			"Fe",
			"Si"
		],
		"parameters": "",
		"runTimeSec": 60,
		"coresPerNode": 4,
		"startUnixTime": 1629335356,
		"creator": {
			"name": "peternemere",
			"user_id": "5de45d85ca40070f421a3a34",
			"email": "peternemere@gmail.com"
		},
		"roiID": "",
		"elementSetID": "",
		"piquantVersion": "registry.github.com/pixlise/piquant/runner:3.2.8-ALPHA",
		"quantMode": "Combined",
		"comments": ""
	},
	"elements": [
		"CaO",
		"TiO2",
		"FeO-T",
		"SiO2"
	],
	"jobId": "quant-789",
	"status": "complete",
	"message": "Nodes ran: 4",
	"endUnixTime": 1629335518,
	"outputFilePath": "UserContent/5de45d85ca40070f421a3a34/052822532/Quantifications",
	"piquantLogList": [
		"node00001_piquant.log",
		"node00001_stdout.log",
		"node00002_piquant.log",
		"node00002_stdout.log",
		"node00003_piquant.log",
		"node00003_stdout.log",
		"node00004_piquant.log",
		"node00004_stdout.log"
	]
	}`))),
		},
	}

	return s3GetRequests, s3GetResponses
}

func Example_quantHandler_Comparison_Fail_ROI() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	getRequests, getResponses := prepROICompareGetCalls()

	// Blank out the ROI
	getResponses[0] = nil

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{"quantIDs":["quant-345", "quant-789"]}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// dataset.bin <nil>
	// combined.bin <nil>
	// 404
	// ROI ID roi-567 not found
}

func Example_quantHandler_Comparison_Fail_Dataset() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	getRequests, getResponses := prepROICompareGetCalls()

	// Blank out the dataset
	getResponses[1] = nil

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{"quantIDs":["quant-345", "quant-789"]}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// dataset.bin <nil>
	// combined.bin <nil>
	// 404
	// Failed to download dataset: NoSuchKey: Returning error from GetObject
}

func Example_quantHandler_Fail_QuantFile() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	getRequests, getResponses := prepROICompareGetCalls()

	// Blank out the first quant file
	getResponses[2] = nil

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{"quantIDs":["quant-345", "quant-789"]}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// dataset.bin <nil>
	// combined.bin <nil>
	// 404
	// Failed to download quant quant-345: NoSuchKey: Returning error from GetObject
}

func Example_quantHandler_Fail_QuantSummary() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	getRequests, getResponses := prepROICompareGetCalls()

	// Blank out the first quant summary file
	getResponses[3] = nil

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{"quantIDs":["quant-345", "quant-789"]}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// dataset.bin <nil>
	// combined.bin <nil>
	// 404
	// Failed to download quant summary quant-345: NoSuchKey: Returning error from GetObject
}

func Example_quantHandler_Comparison_OK() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	getRequests, getResponses := prepROICompareGetCalls()

	mockS3.ExpGetObjectInput = getRequests
	mockS3.QueuedGetObjectOutput = getResponses

	mockS3.AllowGetInAnyOrder = true
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/comparison-for-roi/dataset-123/roi-567", bytes.NewReader([]byte(`{"quantIDs":["quant-345", "quant-789"]}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// This'll read PMC 90 and 95 and do the averaging of those... Deliberately different to the calculateTotals tests!

	// Output:
	// dataset.bin <nil>
	// combined.bin <nil>
	// 200
	// {
	//     "roiID": "roi-567",
	//     "quantTables": [
	//         {
	//             "quantID": "quant-345",
	//             "quantName": "quant test v6 Al \u003e 100%",
	//             "elementWeights": {
	//                 "CaO_%": 7.6938,
	//                 "FeO-T_%": 12.49375,
	//                 "SiO2_%": 40.1224,
	//                 "TiO2_%": 0.28710002
	//             }
	//         },
	//         {
	//             "quantID": "quant-789",
	//             "quantName": "quant test v6 Al \u003e 100%",
	//             "elementWeights": {
	//                 "CaO_%": 7.6938,
	//                 "FeO-T_%": 12.49375,
	//                 "SiO2_%": 40.1224,
	//                 "TiO2_%": 0.28710002
	//             }
	//         }
	//     ]
	// }
}
