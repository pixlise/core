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

package main

import (
	"bytes"
	"fmt"
	"io/ioutil"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/logger"
)

func printHelp(s1 string, s2 string, e error) {
	fmt.Printf("%v|%v|'%v'\n", s1, s2, e)
}
func Example_decodeJobStatusPath() {
	printHelp(decodeJobStatusPath("JobStatus/rtt-123/job1-status.json"))  // actual valid possibilty
	printHelp(decodeJobStatusPath("/JobStatus/rtt-123/job1-status.json")) // / craziness
	// fails:
	printHelp(decodeJobStatusPath(""))
	printHelp(decodeJobStatusPath("/"))
	printHelp(decodeJobStatusPath("JobStatus"))
	printHelp(decodeJobStatusPath("JobStatus/rtt-123/something.txt"))
	printHelp(decodeJobStatusPath("JobStatus//something.txt"))
	printHelp(decodeJobStatusPath("JobStatus/rtt-123/-status.json"))
	printHelp(decodeJobStatusPath("Jobs/some/thing.json"))
	printHelp(decodeJobStatusPath("/Jobs/some/thing.json"))
	printHelp(decodeJobStatusPath("Jobs/rtt-123/job1-status.json"))
	printHelp(decodeJobStatusPath("/Jobs/rtt-123/job1-status.json"))

	// Output:
	// rtt-123|job1|'<nil>'
	// rtt-123|job1|'<nil>'
	// ||'Failed to parse path: '
	// ||'Failed to parse path: /'
	// ||'Failed to parse path: JobStatus'
	// ||'Unexpected file name in path: something.txt, full path path: JobStatus/rtt-123/something.txt'
	// ||'Failed to parse path: JobStatus//something.txt'
	// ||'Unexpected file name in path: -status.json, full path path: JobStatus/rtt-123/-status.json'
	// ||'Unexpected start to monitoring path: Jobs, full path path: Jobs/some/thing.json'
	// ||'Unexpected start to monitoring path: Jobs, full path path: /Jobs/some/thing.json'
	// ||'Unexpected start to monitoring path: Jobs, full path path: Jobs/rtt-123/job1-status.json'
	// ||'Unexpected start to monitoring path: Jobs, full path path: /Jobs/rtt-123/job1-status.json'
}

func Example_regenJobSummaryListBucketFail() {
	const jobBucket = "dev-pixlise-piquant-jobs"
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	l := logger.NullLogger{}

	// Listing returns an error
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(jobBucket), Prefix: aws.String("JobStatus/data123/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{nil}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(regenJobSummary(fs, jobBucket, filepaths.RootJobStatus+"/data123/jobID333-status.json", l))

	// Output:
	// Returning error from ListObjectsV2
}

func Example_regenJobSummaryBadTrigger() {
	const jobBucket = "dev-pixlise-piquant-jobs"
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	l := logger.NullLogger{}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(regenJobSummary(fs, jobBucket, filepaths.RootJobStatus+"/data123/readme.txt", l))

	// Output:
	// Unexpected file name in path: readme.txt, full path path: JobStatus/data123/readme.txt
}

func Example_regenJobSummaryErrorsGettingFiles() {
	const jobBucket = "dev-pixlise-piquant-jobs"
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.AllowGetInAnyOrder = true
	l := logger.NullLogger{}

	// Listing returns 1 item, get status returns error, check that it still requests 2nd item, 2nd item will fail to parse
	// but the func should still upload a blank jobs.json
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(jobBucket), Prefix: aws.String("JobStatus/datasetID0001/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("JobStatus/datasetID0001/abc-123-status.json")},
				{Key: aws.String("JobStatus/datasetID0001/file.txt")},
				{Key: aws.String("JobStatus/datasetID0001/abc-123/params.json")},
				{Key: aws.String("JobStatus/datasetID0001/abc-456-status.json")},
				{Key: aws.String("JobStatus/datasetID0001/abc-456/node1.json")},
				{Key: aws.String("JobStatus/datasetID0001/abc-456/params.json")},
				{Key: aws.String("JobStatus/datasetID0001/abc-456/output/combined.csv")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobStatus/datasetID0001/abc-123-status.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobStatus/datasetID0001/abc-456-status.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobData/datasetID0001/abc-123/params.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobData/datasetID0001/abc-456/params.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobSummaries/datasetID0001-jobs.json"), Body: bytes.NewReader([]byte("{}")),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(regenJobSummary(fs, jobBucket, filepaths.RootJobStatus+"/datasetID0001/job33-status.json", l))

	// Output:
	// <nil>
}

func Example_regenJobSummaryOneJobItem() {
	const jobBucket = "dev-pixlise-piquant-jobs"
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.AllowGetInAnyOrder = true
	l := logger.NullLogger{}

	// Listing returns 1 item, get status returns error, requests second and third item
	// and properly combines the json files into a jobs.json
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(jobBucket), Prefix: aws.String("JobStatus/datasetID123/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("JobStatus/datasetID123/abc-123-status.json")},
				{Key: aws.String("JobStatus/datasetID123/abc-123-node1.json")},
				{Key: aws.String("JobStatus/datasetID123/abc-123/params.json")},
				{Key: aws.String("JobStatus/datasetID123/abc-456-status.json")},
				{Key: aws.String("JobStatus/datasetID123/abc-456/node1.json")},
				{Key: aws.String("JobStatus/datasetID123/abc-456-params.json")},
				{Key: aws.String("JobStatus/datasetID123/abc-456/output/combined.csv")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobStatus/datasetID123/abc-123-status.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobStatus/datasetID123/abc-456-status.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobData/datasetID123/abc-123/params.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobData/datasetID123/abc-456/params.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"jobId": "7ecf8bb3-a369-42d5-914a-80a99e64b3a8",
	"status": "complete",
	"message": "Nodes ran: 1, quantification saved to: Quantifications/SOL-00001/Experiment-00002/test z-Ca,Ti,Fe,Al,Mg,Si.bin",
	"endUnixTime": 0,
	"outputFilePath": "",
	"piquantLogList": null
}`))),
		},
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"pmcs": [
		3559,
		3749,
		3750,
		3751,
		3752,
		3940,
		3941,
		3942,
		3943,
		3944,
		3945
	],
	"name": "test err 6",
	"dataBucket": "dev-pixlise-data",
	"datasetPath": "Downloads/SOL-00001/Experiment-00002/5x11dataset.bin",
	"datasetID": "rtt-123",
	"jobBucket": "dev-pixlise-piquant-jobs",
	"detectorConfig": "PIXL",
	"elements": [
		"Mg",
		"Al",
		"Ca",
		"Ti",
		"Fe",
		"Si"
	],
	"parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
	"runTimeSec": 120,
	"coresPerNode": 6,
	"startUnixTime": 1586237347,
	"creator": {
		"name": "Mickey Mouse",
		"user_id": "user-888"
	},
	"roiID": "roi111",
	"elementSetID": "elem222",
	"piquantVersion": "1.2.3",
	"quantMode": "AB"
}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobSummaries/datasetID123-jobs.json"), Body: bytes.NewReader([]byte(`{
    "abc-456": {
        "shared": false,
        "params": {
            "pmcsCount": 11,
            "name": "test err 6",
            "dataBucket": "dev-pixlise-data",
            "datasetPath": "Downloads/SOL-00001/Experiment-00002/5x11dataset.bin",
            "datasetID": "rtt-123",
            "jobBucket": "dev-pixlise-piquant-jobs",
            "detectorConfig": "PIXL",
            "elements": [
                "Mg",
                "Al",
                "Ca",
                "Ti",
                "Fe",
                "Si"
            ],
            "parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
            "runTimeSec": 120,
            "coresPerNode": 6,
            "startUnixTime": 1586237347,
            "creator": {
                "name": "Mickey Mouse",
                "user_id": "user-888",
                "email": ""
            },
            "roiID": "roi111",
            "elementSetID": "elem222",
            "piquantVersion": "1.2.3",
            "quantMode": "AB",
            "comments": "",
            "roiIDs": []
        },
        "elements": [],
        "jobId": "7ecf8bb3-a369-42d5-914a-80a99e64b3a8",
        "status": "complete",
        "message": "Nodes ran: 1, quantification saved to: Quantifications/SOL-00001/Experiment-00002/test z-Ca,Ti,Fe,Al,Mg,Si.bin",
        "endUnixTime": 0,
        "outputFilePath": "",
        "piquantLogList": null
    }
}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(regenJobSummary(fs, jobBucket, filepaths.RootJobStatus+"/datasetID123/job45-status.json", l))

	// Output:
	// <nil>
}

func Example_regenJobSummaryOneJobItemParamFail() {
	const jobBucket = "dev-pixlise-piquant-jobs"
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.AllowGetInAnyOrder = true
	l := logger.NullLogger{}

	// Listing returns 1 item, get status returns error, requests second and third item
	// third item returns status error, empty jobs.json is uploaded
	mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
		{
			Bucket: aws.String(jobBucket), Prefix: aws.String("JobStatus/datasetID999/"),
		},
	}
	mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
		{
			Contents: []*s3.Object{
				{Key: aws.String("JobStatus/datasetID999/abc-123-status.json")},
				{Key: aws.String("JobStatus/datasetID999/abc-123/node1.json")},
				{Key: aws.String("JobStatus/datasetID999/abc-123/params.json")},
				{Key: aws.String("JobStatus/datasetID999/abc-456-status.json")},
				{Key: aws.String("JobStatus/datasetID999/abc-456/node1.json")},
				{Key: aws.String("JobStatus/datasetID999/abc-456/params.json")},
				{Key: aws.String("JobStatus/datasetID999/abc-456/output/combined.csv")},
			},
		},
	}

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobStatus/datasetID999/abc-123-status.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobStatus/datasetID999/abc-456-status.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobData/datasetID999/abc-123/params.json"),
		},
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobData/datasetID999/abc-456/params.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"jobId": "7ecf8bb3-a369-42d5-914a-80a99e64b3a8",
	"status": "complete",
	"message": "Nodes ran: 1, quantification saved to: Quantifications/SOL-00001/Experiment-00002/test z-Ca,Ti,Fe,Al,Mg,Si.bin",
	"endUnixTime": 0,
	"outputFilePath": "",
	"piquantLogList": null
}`))),
		},
		nil,
		nil,
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(jobBucket), Key: aws.String("JobSummaries/datasetID999-jobs.json"), Body: bytes.NewReader([]byte("{}")),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	fs := fileaccess.MakeS3Access(&mockS3)
	fmt.Println(regenJobSummary(fs, jobBucket, filepaths.RootJobStatus+"/datasetID999/job8-status.json", l))

	// Output:
	// <nil>
}
