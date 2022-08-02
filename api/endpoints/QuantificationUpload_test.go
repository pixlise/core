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
	"math/rand"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/awsutil"
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
        "roiIDs": []
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
