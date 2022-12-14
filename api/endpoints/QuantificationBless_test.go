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
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/timestamper"
)

func Example_quantHandler_BlessShared1stQuant() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Getting the quant summary to verify it exists
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job2.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/blessed-quant.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"shared": false,
	"params": {
		"pmcsCount": 93,
		"name": "my test quant",
		"dataBucket": "dev-pixlise-data",
		"datasetPath": "Datasets/rtt-456/5x5dataset.bin",
		"datasetID": "rtt-456",
		"jobBucket": "dev-pixlise-piquant-jobs",
		"detectorConfig": "PIXL",
		"elements": [
			"Sc",
			"Cr"
		],
		"parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
		"runTimeSec": 120,
		"coresPerNode": 6,
		"startUnixTime": 1589948988,
		"creator": {
			"name": "peternemere",
			"user_id": "600f2a0806b6c70071d3d174"
		},
		"roiID": "ZcH49SYZ",
		"elementSetID": ""
	},
	"elements": ["Sc", "Cr"],
	"jobId": "job2",
	"status": "complete",
	"message": "Nodes ran: 1",
	"endUnixTime": 1589949035,
	"outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	"piquantLogList": [
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
	]
}`))),
		},
		nil,
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/blessed-quant.json"), Body: bytes.NewReader([]byte(`{
    "history": [
        {
            "version": 1,
            "blessedAt": 1234567890,
            "userId": "600f2a0806b6c70071d3d174",
            "userName": "Niko Bellic",
            "jobId": "job2"
        }
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567890},
	}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/bless/rtt-456/shared-job2", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_quantHandler_BlessSharedQuantV4() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Getting the quant summary to verify it exists
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job2.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/blessed-quant.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"shared": false,
	"params": {
		"pmcsCount": 93,
		"name": "my test quant",
		"dataBucket": "dev-pixlise-data",
		"datasetPath": "Datasets/rtt-456/5x5dataset.bin",
		"datasetID": "rtt-456",
		"jobBucket": "dev-pixlise-piquant-jobs",
		"detectorConfig": "PIXL",
		"elements": [
			"Sc",
			"Cr"
		],
		"parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
		"runTimeSec": 120,
		"coresPerNode": 6,
		"startUnixTime": 1589948988,
		"creator": {
			"name": "peternemere",
			"user_id": "600f2a0806b6c70071d3d174"
		},
		"roiID": "ZcH49SYZ",
		"elementSetID": ""
	},
	"elements": ["Sc", "Cr"],
	"jobId": "job2",
	"status": "complete",
	"message": "Nodes ran: 1",
	"endUnixTime": 1589949035,
	"outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	"piquantLogList": [
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
	]
}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"history": [
		{
			"version": 1,
			"blessedAt": 1234567090,
			"userId": "600f2a0806b6c70071d3d174",
			"userName": "Niko Bellic",
			"jobId": "jobAAA"
		},
		{
			"version": 3,
			"blessedAt": 1234567690,
			"userId": "555555",
			"userName": "Michael Da Santa",
			"jobId": "jobCCC"
		},
		{
			"version": 2,
			"blessedAt": 1234567490,
			"userId": "600f2a0806b6c70071d3d174",
			"userName": "Niko Bellic",
			"jobId": "jobBBB"
		}
	]
}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/blessed-quant.json"), Body: bytes.NewReader([]byte(`{
    "history": [
        {
            "version": 1,
            "blessedAt": 1234567090,
            "userId": "600f2a0806b6c70071d3d174",
            "userName": "Niko Bellic",
            "jobId": "jobAAA"
        },
        {
            "version": 3,
            "blessedAt": 1234567690,
            "userId": "555555",
            "userName": "Michael Da Santa",
            "jobId": "jobCCC"
        },
        {
            "version": 2,
            "blessedAt": 1234567490,
            "userId": "600f2a0806b6c70071d3d174",
            "userName": "Niko Bellic",
            "jobId": "jobBBB"
        },
        {
            "version": 4,
            "blessedAt": 1234567890,
            "userId": "600f2a0806b6c70071d3d174",
            "userName": "Niko Bellic",
            "jobId": "job2"
        }
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567890},
	}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/bless/rtt-456/shared-job2", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

// This is different, because it has to implicitly do a share of the quantification!
func Example_quantHandler_BlessUserQuant() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Getting the quant summary to verify it exists
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			// Share verifies that summary exists
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job2.json"),
		},
		{
			// Post sharing, bless checks that shared summary exists
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job2.json"),
		},
		{
			// Getting previous blessed list
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/blessed-quant.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"shared": false,
	"params": {
		"pmcsCount": 93,
		"name": "my test quant",
		"dataBucket": "dev-pixlise-data",
		"datasetPath": "Datasets/rtt-456/5x5dataset.bin",
		"datasetID": "rtt-456",
		"jobBucket": "dev-pixlise-piquant-jobs",
		"detectorConfig": "PIXL",
		"elements": [
			"Sc",
			"Cr"
		],
		"parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
		"runTimeSec": 120,
		"coresPerNode": 6,
		"startUnixTime": 1589948988,
		"creator": {
			"name": "peternemere",
			"user_id": "600f2a0806b6c70071d3d174"
		},
		"roiID": "ZcH49SYZ",
		"elementSetID": ""
	},
	"elements": ["Sc", "Cr"],
	"jobId": "job2",
	"status": "complete",
	"message": "Nodes ran: 1",
	"endUnixTime": 1589949035,
	"outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	"piquantLogList": [
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
	]
	}`))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"shared": true,
	"params": {
		"pmcsCount": 93,
		"name": "my test quant",
		"dataBucket": "dev-pixlise-data",
		"datasetPath": "Datasets/rtt-456/5x5dataset.bin",
		"datasetID": "rtt-456",
		"jobBucket": "dev-pixlise-piquant-jobs",
		"detectorConfig": "PIXL",
		"elements": [
			"Sc",
			"Cr"
		],
		"parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
		"runTimeSec": 120,
		"coresPerNode": 6,
		"startUnixTime": 1589948988,
		"creator": {
			"name": "peternemere",
			"user_id": "600f2a0806b6c70071d3d174"
		},
		"roiID": "ZcH49SYZ",
		"elementSetID": ""
	},
	"elements": ["Sc", "Cr"],
	"jobId": "job2",
	"status": "complete",
	"message": "Nodes ran: 1",
	"endUnixTime": 1589949035,
	"outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	"piquantLogList": [
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
		"https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
	]
}`))),
		},
		nil,
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/blessed-quant.json"), Body: bytes.NewReader([]byte(`{
    "history": [
        {
            "version": 1,
            "blessedAt": 1234567890,
            "userId": "600f2a0806b6c70071d3d174",
            "userName": "Niko Bellic",
            "jobId": "job2"
        }
    ]
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	// Copying object (the sharing operation)
	mockS3.ExpCopyObjectInput = []s3.CopyObjectInput{
		{
			Bucket:     aws.String(UsersBucketForUnitTest),
			Key:        aws.String("UserContent/shared/rtt-456/Quantifications/job2.bin"),
			CopySource: aws.String(UsersBucketForUnitTest + "/UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2.bin"),
		},
		{
			Bucket:     aws.String(UsersBucketForUnitTest),
			Key:        aws.String("UserContent/shared/rtt-456/Quantifications/summary-job2.json"),
			CopySource: aws.String(UsersBucketForUnitTest + "/UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job2.json"),
		},
		{
			Bucket:     aws.String(UsersBucketForUnitTest),
			Key:        aws.String("UserContent/shared/rtt-456/Quantifications/job2.csv"),
			CopySource: aws.String(UsersBucketForUnitTest + "/UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2.csv"),
		},
	}
	mockS3.QueuedCopyObjectOutput = []*s3.CopyObjectOutput{
		{},
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234567890},
	}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/quantification/bless/rtt-456/job2", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}
