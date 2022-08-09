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
	"math/rand"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/pixlise/pixlise-go-api/api/services"
	"gitlab.com/pixlise/pixlise-go-api/core/awsutil"
)

// Quantification manual uploads, this has many failure scenarios...
func Example_quantHandler_UploadFails() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567890},
	}
	apiRouter := MakeRouter(svcs)

	// No body
	req, _ := http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte("")))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// No name line
	req, _ = http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte(`Comments=Hello world
CSV
Header line
PMC,Ca_%,livetime,RTT,SCLK,filename
1,5.3,9.9,98765,1234567890,Normal_A
`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// No comment line
	req, _ = http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte(`Name=Hello world
CSV
Header line
PMC,Ca_%,livetime,RTT,SCLK,filename
1,5.3,9.9,98765,1234567890,Normal_A
`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// No/bad CSV line
	req, _ = http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte(`Name=Hello world
Comments=The test
Header line
PMC,Ca_%,livetime,RTT,SCLK,filename
1,5.3,9.9,98765,1234567890,Normal_A
`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Missing header line
	req, _ = http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte(`Name=Hello world
Comments=The test
CSV
PMC,Ca_%,livetime,RTT,SCLK,filename
1,5.3,9.9,98765,1234567890,Normal_A
`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Missing PMC column
	req, _ = http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte(`Name=Hello world
Comments=The test
CSV
Header line
Ca_%,livetime,RTT,SCLK,filename
5.3,9.9,98765,1234567890,Normal_A
`)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)
	/*
	   	// PMC not the first column
	   	req, _ = http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte(`Name=Hello world
	   Comments=The test
	   CSV
	   Header line
	   Ca_%,PMC,livetime,RTT,SCLK,filename
	   5.3,1,9.9,98765,1234567890,Normal_A
	   `)))
	   	resp = executeRequest(req, apiRouter.Router)

	   	fmt.Println(resp.Code)
	   	fmt.Println(resp.Body)
	*/
	// Output:
	// 400
	// Bad upload format. Expecting format:
	// Name=The quant name
	// Comments=The comments\nWith new lines\nEncoded like so
	// CSV
	// <csv title line>
	// <csv column headers>
	// csv rows
	//
	// 400
	// Bad upload format. Expecting format:
	// Name=The quant name
	// Comments=The comments\nWith new lines\nEncoded like so
	// CSV
	// <csv title line>
	// <csv column headers>
	// csv rows
	//
	// 400
	// Bad upload format. Expecting format:
	// Name=The quant name
	// Comments=The comments\nWith new lines\nEncoded like so
	// CSV
	// <csv title line>
	// <csv column headers>
	// csv rows
	//
	// 400
	// Bad upload format. Expecting format:
	// Name=The quant name
	// Comments=The comments\nWith new lines\nEncoded like so
	// CSV
	// <csv title line>
	// <csv column headers>
	// csv rows
	//
	// 400
	// CSV did not contain any _% columns
	//
	// 400
	// CSV missing column: "PMC"
	//
}

// Quantification manual uploads, success scenario
func Example_quantHandler_UploadOK() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// bin, CSV and summary are all uploaded
	expBinBytes := []byte{10, 4, 67, 97, 95, 37, 10, 8, 108, 105, 118, 101, 116, 105, 109, 101, 18, 2, 0, 0, 26, 31, 10, 1, 65, 18, 26, 8, 1, 16, 205, 131, 6, 24, 210, 133, 216, 204, 4, 42, 5, 21, 154, 153, 169, 64, 42, 5, 21, 102, 102, 30, 65, 26, 31, 10, 1, 66, 18, 26, 8, 1, 16, 205, 131, 6, 24, 210, 133, 216, 204, 4, 42, 5, 21, 102, 102, 166, 64, 42, 5, 21, 205, 204, 28, 65}

	//fmt.Println(string(expBinBytes))
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/upload_quant123.bin"), Body: bytes.NewReader(expBinBytes),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/upload_quant123.csv"), Body: bytes.NewReader([]byte(`Header line
PMC, Ca_%, livetime, RTT, SCLK, filename
1, 5.3, 9.9, 98765, 1234567890, Normal_A
1, 5.2, 9.8, 98765, 1234567890, Normal_B
`)),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-upload_quant123.json"), Body: bytes.NewReader([]byte(`{
    "shared": false,
    "params": {
        "pmcsCount": 0,
        "name": "Hello world",
        "dataBucket": "datasets-bucket",
        "datasetPath": "Datasets/rtt-456/dataset.bin",
        "datasetID": "rtt-456",
        "jobBucket": "job-bucket",
        "detectorConfig": "",
        "elements": [
            "Ca"
        ],
        "parameters": "",
        "runTimeSec": 0,
        "coresPerNode": 0,
        "startUnixTime": 1234567890,
        "creator": {
            "name": "Niko Bellic",
            "user_id": "600f2a0806b6c70071d3d174",
            "email": "niko@spicule.co.uk"
        },
        "roiID": "",
        "elementSetID": "",
        "piquantVersion": "N/A",
        "quantMode": "ABManual",
        "comments": "The test",
        "roiIDs": [],
        "command": "map"
    },
    "elements": [
        "Ca"
    ],
    "jobId": "upload_quant123",
    "status": "complete",
    "message": "user-supplied quantification processed",
    "endUnixTime": 1234567890,
    "outputFilePath": "UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications",
    "piquantLogList": []
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
		{},
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"quant123"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567890},
	}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/upload/rtt-456", bytes.NewReader([]byte(`Name=Hello world
Comments=The test
CSV
Header line
PMC, Ca_%, livetime, RTT, SCLK, filename
1, 5.3, 9.9, 98765, 1234567890, Normal_A
1, 5.2, 9.8, 98765, 1234567890, Normal_B
`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// "upload_quant123"
}

func Example_isABQuant() {
	fmt.Printf("%v\n", isABQuant([]string{
		"header",
		"PMC,filename,Ca_%,Ca_err",
		"100,Normal_A,1.2,1.1",
		"100,Normal_B,1.4,1.2",
		"101,Normal_A,1.3,1.2",
		"101,Normal_B,1.6,1.6",
	}, 1))

	fmt.Printf("%v\n", isABQuant([]string{
		"header",
		"PMC,filename,Ca_%,Ca_err",
		"100,Normal_A,1.2,1.1",
		"101,Normal_A,1.3,1.2",
		"100,Normal_B,1.4,1.2",
		"101,Normal_B,1.6,1.6",
	}, 1))

	fmt.Printf("%v\n", isABQuant([]string{
		"header",
		"PMC,filename,Ca_%,Ca_err",
		"100,Normal_Combined,1.2,1.1",
		"101,Normal_Combined,1.3,1.2",
	}, 1))

	// Output:
	// true
	// true
	// false
}
