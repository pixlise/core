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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/quantModel"
	"github.com/pixlise/core/v3/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

func Test_quantHandler_AdminList(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			// Peter found
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "5de45d85ca40070f421a3a34"},
					{"Notifications", bson.D{
						{"Topics", bson.A{}},
					}},
					{"Config", bson.D{
						{"Name", "Peter N"},
						{"Email", "peter@pixlise.org"},
						{"Cell", ""},
						{"DataCollection", "unknown"},
					}},
				},
			),
			// Tom not found
			mtest.CreateCursorResponse(
				1,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
			),
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.NextBatch,
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		// Listing lists files in user dir, share dir, then gets jobs.json and forms a single list of job summaries to send back
		mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
			{
				Bucket: aws.String(jobBucketForUnitTest), Prefix: aws.String("JobSummaries/"),
			},
		}
		mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
			{
				Contents: []*s3.Object{
					{Key: aws.String("JobSummaries/rtt-456-jobs.json")},
					{Key: aws.String("JobSummaries/rtt-456/shouldnt/behere.pmcs")},
					{Key: aws.String("JobSummaries/rtt-456-tmp.txt")},
					{Key: aws.String("JobSummaries/rtt-111-jobs.json")},
					{Key: aws.String("JobSummaries/rtt-222-jobs.json")},
					{Key: aws.String("JobSummaries/rtt-123-jobs.json")},
					{Key: aws.String("JobSummaries/rtt-123/job2/params.json")},
					{Key: aws.String("JobSummaries/jobs.json")},
				},
			},
		}

		mockS3.ExpGetObjectInput = []s3.GetObjectInput{
			{
				Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobSummaries/rtt-456-jobs.json"),
			},
			{
				Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobSummaries/rtt-111-jobs.json"),
			},
			{
				Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobSummaries/rtt-222-jobs.json"),
			},
			{
				Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobSummaries/rtt-123-jobs.json"),
			},
		}
		mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
    "job1": {
        "shared": false,
        "params": {
            "pmcsCount": 313,
            "name": "alllll",
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
                "name": "peternemere",
                "user_id": "5de45d85ca40070f421a3a34",
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
        "status": "error",
        "message": "Error when converting combined CSV to PIXLISE bin format: Failed to determine detector ID from filename column: ",
        "endUnixTime": 1592968196,
        "outputFilePath": "",
        "piquantLogList": null
    },
    "job2": {
        "shared": false,
        "params": {
            "pmcsCount": 1769,
            "name": "ase12",
            "dataBucket": "dev-pixlise-data",
            "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
            "datasetID": "rtt-456",
            "jobBucket": "dev-pixlise-piquant-jobs",
            "detectorConfig": "PIXL",
            "elements": [
                "Ru",
                "Cr"
            ],
            "parameters": "-q,pPIETXCFsr -b,0,12,60,910,2800,16",
            "runTimeSec": 120,
            "coresPerNode": 6,
            "startUnixTime": 1591826489,
            "creator": {
                "name": "tom",
                "user_id": "5e3b3bc480ee5c191714d6b7",
                "email": ""
            },
            "roiID": "",
            "elementSetID": "",
            "quantMode": "AB"
        },
        "jobId": "job2",
        "status": "nodes_running",
        "message": "Node count: 1, Files/Node: 1770",
        "endUnixTime": 0,
        "outputFilePath": "",
        "piquantLogList": null
    }
}`))),
			},
			nil,
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`bad json`))),
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"job7": {
        "shared": false,
		"params": {
			"pmcsCount": 93,
			"name": "in progress quant",
			"dataBucket": "dev-pixlise-data",
			"datasetPath": "Datasets/rtt-123/5x5dataset.bin",
			"datasetID": "rtt-123",
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
				"user_id": "600f2a0806b6c70071d3d174",
                "email": ""
			},
			"roiID": "ZcH49SYZ",
			"elementSetID": ""
		},
		"jobId": "job7",
		"status": "complete",
		"message": "Something about the nodes",
        "endUnixTime": 1592968196,
        "outputFilePath": "",
        "piquantLogList": null
	}
}`))),
			},
		}
		// NOTE: job2 and job7 was missing elements, because we introduced this later, checking that API still puts in empty list always

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/quantification", nil)
		resp := executeRequest(req, apiRouter.Router)

		if resp.Code != 200 {
			t.Errorf("Invalid resp code: %v", resp.Code)
		}

		// Order of response items is unpredictable, just check that they all exist
		respSummaries := []quantModel.JobSummaryItem{}

		getBody, _ := ioutil.ReadAll(resp.Body)
		err := json.Unmarshal(getBody, &respSummaries)
		if err != nil {
			t.Error("Failed to read response")
		}

		if len(respSummaries) != 3 {
			t.Errorf("Expected 3 summaries, got: %v", len(respSummaries))
		}

		// Check each one
		expJob1 := `{
    "shared": false,
    "params": {
        "pmcsCount": 313,
        "name": "alllll",
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
            "name": "Peter N",
            "user_id": "5de45d85ca40070f421a3a34",
            "email": "peter@pixlise.org"
        },
        "roiID": "",
        "elementSetID": "",
        "piquantVersion": "3.0.3",
        "quantMode": "Combined",
        "comments": "",
        "roiIDs": []
    },
    "elements": [
        "Sc",
        "Cr"
    ],
    "jobId": "job1",
    "status": "error",
    "message": "Error when converting combined CSV to PIXLISE bin format: Failed to determine detector ID from filename column: ",
    "endUnixTime": 1592968196,
    "outputFilePath": "",
    "piquantLogList": null
}`
		expJob2 := `{
    "shared": false,
    "params": {
        "pmcsCount": 1769,
        "name": "ase12",
        "dataBucket": "dev-pixlise-data",
        "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
        "datasetID": "rtt-456",
        "jobBucket": "dev-pixlise-piquant-jobs",
        "detectorConfig": "PIXL",
        "elements": [
            "Ru",
            "Cr"
        ],
        "parameters": "-q,pPIETXCFsr -b,0,12,60,910,2800,16",
        "runTimeSec": 120,
        "coresPerNode": 6,
        "startUnixTime": 1591826489,
        "creator": {
            "name": "tom",
            "user_id": "5e3b3bc480ee5c191714d6b7",
            "email": ""
        },
        "roiID": "",
        "elementSetID": "",
        "piquantVersion": "",
        "quantMode": "AB",
        "comments": "",
        "roiIDs": []
    },
    "elements": [],
    "jobId": "job2",
    "status": "nodes_running",
    "message": "Node count: 1, Files/Node: 1770",
    "endUnixTime": 0,
    "outputFilePath": "",
    "piquantLogList": null
}`
		expJob3 := `{
    "shared": false,
    "params": {
        "pmcsCount": 93,
        "name": "in progress quant",
        "dataBucket": "dev-pixlise-data",
        "datasetPath": "Datasets/rtt-123/5x5dataset.bin",
        "datasetID": "rtt-123",
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
            "name": "Peter N",
            "user_id": "5de45d85ca40070f421a3a34",
            "email": "peter@pixlise.org"
        },
        "roiID": "ZcH49SYZ",
        "elementSetID": "",
        "piquantVersion": "",
        "quantMode": "",
        "comments": "",
        "roiIDs": []
    },
    "elements": [],
    "jobId": "job7",
    "status": "complete",
    "message": "Something about the nodes",
    "endUnixTime": 1592968196,
    "outputFilePath": "",
    "piquantLogList": null
}`

		for c, item := range respSummaries {
			itemJ, err := json.MarshalIndent(item, "", utils.PrettyPrintIndentForJSON)
			if err != nil {
				t.Errorf("Failed to read summary[%v]", c)
			}

			summaryStr := string(itemJ)

			if item.JobID == "job1" {
				if summaryStr != expJob1 {
					t.Errorf("Summary [%v] is wrong, expected: %v\ngot: %v\n", c, expJob1, summaryStr)
				}
			} else if item.JobID == "job2" {
				if summaryStr != expJob2 {
					t.Errorf("Summary [%v] is wrong, expected: %v\ngot: %v\n", c, expJob2, summaryStr)
				}
			} else if item.JobID == "job3" {
				if summaryStr != expJob3 {
					t.Errorf("Summary [%v] is wrong, expected: %v\ngot: %v\n", c, expJob3, summaryStr)
				}
			}
		}
	})
}

func Test_quantHandler_List(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "600f2a0806b6c70071d3d174"},
					{"Notifications", bson.D{
						{"Topics", bson.A{}},
					}},
					{"Config", bson.D{
						{"Name", "Peter N"},
						{"Email", "peter@pixlise.org"},
						{"Cell", ""},
						{"DataCollection", "unknown"},
					}},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()
		mockS3.AllowGetInAnyOrder = true

		// Listing lists files in user dir, share dir, then gets datasets jobs.json and forms a single list of job summaries to send back
		mockS3.ExpListObjectsV2Input = []s3.ListObjectsV2Input{
			{
				Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-"),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Prefix: aws.String("UserContent/shared/rtt-456/Quantifications/summary-"),
			},
		}
		mockS3.QueuedListObjectsV2Output = []*s3.ListObjectsV2Output{
			{
				Contents: []*s3.Object{
					{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job1.json")},
					//{Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-something.txt")},
				},
			},
			{
				Contents: []*s3.Object{
					{Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job2.json")},
				},
			},
		}

		mockS3.ExpGetObjectInput = []s3.GetObjectInput{
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job1.json"),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job2.json"),
			},
			{
				Bucket: aws.String(jobBucketForUnitTest), Key: aws.String("JobSummaries/rtt-456-jobs.json"),
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
			"user_id": "600f2a0806b6c70071d3d174",
            "email": ""
		},
		"roiID": "ZcH49SYZ",
		"elementSetID": ""
	},
	"jobId": "job1",
	"status": "complete",
	"message": "Nodes ran: 1",
	"endUnixTime": 1589949035,
	"outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	"piquantLogList": [
		"node00001_stdout.log",
		"node00001_threads.log"
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
			"user_id": "600f2a0806b6c70071d3d174",
            "email": ""
		},
		"roiID": "ZcH49SYZ",
		"elementSetID": ""
	},
	"jobId": "job2",
	"status": "complete",
	"message": "Nodes ran: 1",
	"endUnixTime": 1589949035,
	"outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	"piquantLogList": [
		"node00001_stdout.log",
		"node00001_threads.log"
	]
}`))),
			},
			// Only job 5 should be sent out, job 3 is completed, job 4 is another dataset
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(`{
	"job3": {
		"params": {
			"pmcsCount": 93,
			"name": "in progress quant",
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
				"user_id": "600f2a0806b6c70071d3d174",
                "email": ""
			},
			"roiID": "ZcH49SYZ",
			"elementSetID": ""
		},
		"jobId": "job3",
		"status": "complete",
		"message": "Something about the nodes"
	},
	"job5": {
		"params": {
			"pmcsCount": 93,
			"name": "in progress quant",
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
				"user_id": "600f2a0806b6c70071d3d174",
                "email": ""
			},
			"roiID": "ZcH49SYZ",
			"elementSetID": ""
		},
		"elements": ["Cr03", "Sc"],
		"jobId": "job5",
		"status": "nodes_running",
		"message": "Something about the nodes"
	},
	"job1": {
		"params": {
			"pmcsCount": 93,
			"name": "in progress quant",
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
				"user_id": "600f2a0806b6c70071d3d174",
                "email": ""
			},
			"roiID": "ZcH49SYZ",
			"elementSetID": ""
		},
		"jobId": "job1",
		"status": "nodes_running",
		"message": "Not done yet"
	}
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
			"jobId": "job2"
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

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest("GET", "/quantification/rtt-456", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "summaries": [
        {
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
                    "name": "Peter N",
                    "user_id": "600f2a0806b6c70071d3d174",
                    "email": "peter@pixlise.org"
                },
                "roiID": "ZcH49SYZ",
                "elementSetID": "",
                "piquantVersion": "",
                "quantMode": "",
                "comments": "",
                "roiIDs": []
            },
            "elements": [],
            "jobId": "job1",
            "status": "complete",
            "message": "Nodes ran: 1",
            "endUnixTime": 1589949035,
            "outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
            "piquantLogList": [
                "node00001_stdout.log",
                "node00001_threads.log"
            ]
        },
        {
            "shared": false,
            "params": {
                "pmcsCount": 93,
                "name": "in progress quant",
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
                    "name": "Peter N",
                    "user_id": "600f2a0806b6c70071d3d174",
                    "email": "peter@pixlise.org"
                },
                "roiID": "ZcH49SYZ",
                "elementSetID": "",
                "piquantVersion": "",
                "quantMode": "",
                "comments": "",
                "roiIDs": []
            },
            "elements": [
                "Cr03",
                "Sc"
            ],
            "jobId": "job5",
            "status": "nodes_running",
            "message": "Something about the nodes",
            "endUnixTime": 0,
            "outputFilePath": "",
            "piquantLogList": null
        },
        {
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
                    "name": "Peter N",
                    "user_id": "600f2a0806b6c70071d3d174",
                    "email": "peter@pixlise.org"
                },
                "roiID": "ZcH49SYZ",
                "elementSetID": "",
                "piquantVersion": "",
                "quantMode": "",
                "comments": "",
                "roiIDs": []
            },
            "elements": [],
            "jobId": "shared-job2",
            "status": "complete",
            "message": "Nodes ran: 1",
            "endUnixTime": 1589949035,
            "outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
            "piquantLogList": [
                "node00001_stdout.log",
                "node00001_threads.log"
            ]
        }
    ],
    "blessedQuant": {
        "version": 3,
        "blessedAt": 1234567690,
        "userId": "555555",
        "userName": "Michael Da Santa",
        "jobId": "shared-job2"
    }
}
`)
	})
}

func Test_quantHandler_Get(t *testing.T) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		// User name lookup
		mongoMockedResponses := []primitive.D{
			mtest.CreateCursorResponse(
				0,
				"userdatabase-unit_test.users",
				mtest.FirstBatch,
				bson.D{
					{"Userid", "600f2a0806b6c70071d3d174"},
					{"Notifications", bson.D{
						{"Topics", bson.A{}},
					}},
					{"Config", bson.D{
						{"Name", "Peter N"},
						{"Email", "peter@pixlise.org"},
						{"Cell", ""},
						{"DataCollection", "unknown"},
					}},
				},
			),
		}

		mt.AddMockResponses(mongoMockedResponses...)

		const summaryJSON = `{
   "dataset_id": "rtt-456",
   "group": "the-group",
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
}`

		const publicDatasetsJSON = `{
   "rtt-456": {
      "dataset_id": "rtt-456",
      "public": false,
      "public_release_utc_time_sec": 0,
      "sol": ""
   }
}`
		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		mockS3.ExpGetObjectInput = []s3.GetObjectInput{
			{
				Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
			},
			{
				Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job1.json"), // 4
			},
			{
				Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"), // 1
			},
			{
				Bucket: aws.String("config-bucket"), Key: aws.String("PixliseConfig/datasets-auth.json"), // 1
			},
			{
				Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"), // 2
			},
			{
				Bucket: aws.String("config-bucket"), Key: aws.String("PixliseConfig/datasets-auth.json"), // 2
			},
			{
				Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"), // 3
			},
			{
				Bucket: aws.String("config-bucket"), Key: aws.String("PixliseConfig/datasets-auth.json"), // 2
			},
		}
		mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
			},
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
						"user_id": "600f2a0806b6c70071d3d174",
						"email": ""
					},
					"roiID": "ZcH49SYZ",
					"elementSetID": "",
					"quantMode": "AB"
				},
				"jobId": "job1",
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
				Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))), // 1
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(publicDatasetsJSON))), // 1
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))), // 2
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(publicDatasetsJSON))), // 2
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))), // 3
			},
			{
				Body: ioutil.NopCloser(bytes.NewReader([]byte(publicDatasetsJSON))), // 2
			},
		}

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")
		mockUser := pixlUser.UserInfo{
			Name:   "Niko Bellic",
			UserID: "600f2a0806b6c70071d3d174",
			Permissions: map[string]bool{
				"read:data-analysis": true,
				"access:the-group":   true,
			},
		}
		svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
		apiRouter := MakeRouter(svcs)

		// File found, OK
		req, _ := http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 200, `{
    "summary": {
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
                "name": "Peter N",
                "user_id": "600f2a0806b6c70071d3d174",
                "email": "peter@pixlise.org"
            },
            "roiID": "ZcH49SYZ",
            "elementSetID": "",
            "piquantVersion": "",
            "quantMode": "AB",
            "comments": "",
            "roiIDs": []
        },
        "elements": [],
        "jobId": "job1",
        "status": "complete",
        "message": "Nodes ran: 1",
        "endUnixTime": 1589949035,
        "outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
        "piquantLogList": [
            "https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
            "https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
        ]
    },
    "url": "https:///quantification/download/rtt-456/job1"
}
`)

		mockUser = pixlUser.UserInfo{
			Name:   "Niko Bellic",
			UserID: "600f2a0806b6c70071d3d174",
			Permissions: map[string]bool{
				"read:data-analysis": true,
				"access:wrong-group": true,
			},
		}
		svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
		apiRouter = MakeRouter(svcs)

		// Dataset summary file wrong group, ERROR - NO ACCESS
		req, _ = http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 403, `dataset rtt-456 not permitted
`)

		// Dataset summary has different group, ACCESS DENIED
		req, _ = http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 500, `failed to verify dataset group permission
`)
		mockUser = pixlUser.UserInfo{
			Name:   "Niko Bellic",
			UserID: "600f2a0806b6c70071d3d174",
			Permissions: map[string]bool{
				"read:data-analysis": true,
				"access:the-group":   true,
			},
		}
		svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
		apiRouter = MakeRouter(svcs)

		// Failed to parse summary JSON, ERROR
		req, _ = http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
		resp = executeRequest(req, apiRouter.Router)

		checkResult(t, resp, 500, `failed to verify dataset group permission
`)
	})
}

func Example_quant_Stream_OK() {
	const summaryJSON = `{
   "dataset_id": "590340",
   "group": "groupie",
   "drive_id": 292,
   "site_id": 1,
   "target_id": "?",
   "sol": "0",
   "rtt": 590340,
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
}`

	datasetBytes := []byte{60, 113, 117, 97, 110, 116, 62}

	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/590340/Quantifications/job-7.bin"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			ContentLength: aws.Int64(int64(len(datasetBytes))),
			Body:          ioutil.NopCloser(bytes.NewReader(datasetBytes)), // return some printable chars so easier to compare in Output comment
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Permissions: map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/quantification/download/590340/job-7", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	// Make sure the response is the right kind...
	fmt.Println(resp.HeaderMap["Content-Disposition"])
	fmt.Println(resp.HeaderMap["Cache-Control"])
	fmt.Println(resp.HeaderMap["Content-Length"])
	fmt.Println(resp.Body)

	// Output:
	// 200
	// [attachment; filename="job-7.bin"]
	// [max-age=604800]
	// [7]
	// <quant>
}

func Example_quantHandler_Stream_404() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Permissions: map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/quantification/download/590340/job-7", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 404
	// 590340 not found
}

func Example_quantHandler_Stream_BadSummary() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/590340/summary.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Permissions: map[string]bool{
			"access:groupie": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/quantification/download/590340/job-7", nil) // Should return empty list, datasets.json fails to download
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 500
	// failed to verify dataset group permission
}
