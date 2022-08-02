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
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/awsutil"
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	svcs.TimeStamper = &services.MockTimeNowStamper{
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
