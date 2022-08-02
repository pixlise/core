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
)

func Example_quantHandler_DeleteUserJobNotExist() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job1.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Non-existant job, ERROR
	req, _ := http.NewRequest("DELETE", "/quantification/rtt-456/job1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// job1 not found
}

// TODO: test job cancellation if/when implemented

func Example_quantHandler_DeleteUserJob() {
	rand.Seed(time.Now().UnixNano())
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.AllowDeleteInAnyOrder = true

	// Listing log files for each deletion
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2-logs/"),
		},
	}

	// First has logs, second has none (old style quant), third has 1 file
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2-logs/log001.txt")},
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2-logs/log002.txt")},
				{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2-logs/log001_another.txt")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job2.json"),
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
	}

	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job2.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2.bin"),
		},
		{
			Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobStatus/rtt-456/job2-status.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2.csv"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2-logs/log001.txt"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2-logs/log002.txt"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/job2-logs/log001_another.txt"),
		},
	}

	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
		{},
		{},
		{},
		{},
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Existant job, OK
	req, _ := http.NewRequest("DELETE", "/quantification/rtt-456/job2", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}

func Example_quantHandler_DeleteSharedJobNotExists() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job1.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Non-existant shared job, ERROR
	req, _ := http.NewRequest("DELETE", "/quantification/rtt-456/shared-job1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// job1 not found
}

func Example_quantHandler_DeleteSharedJob() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.AllowDeleteInAnyOrder = true

	// Listing log files for each deletion
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/shared/rtt-456/Quantifications/job3-logs/"),
		},
	}

	// First has logs, second has none (old style quant), third has 1 file
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("UserContent/shared/rtt-456/Quantifications/job3-logs/log001.txt")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job3.json"),
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
	"jobId": "job3",
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
	}

	mockS3.ExpDeleteObjectInput = []s3.DeleteObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job3.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/job3.bin"),
		},
		{
			Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobStatus/rtt-456/job3-status.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/job3.csv"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/job3-logs/log001.txt"),
		},
	}

	mockS3.QueuedDeleteObjectOutput = []*s3.DeleteObjectOutput{
		{},
		{},
		{},
		{},
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// Existant shared job, OK
	req, _ := http.NewRequest("DELETE", "/quantification/rtt-456/shared-job3", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
}
