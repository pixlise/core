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
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/pixlUser"
	"github.com/pixlise/core/core/quantModel"
	"github.com/pixlise/core/core/utils"
)

func Example_quantHandler_AdminList() {
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/quantification", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)

	// Order of response items was unpredictable, print them in alphabetical id order
	respSummaries := []quantModel.JobSummaryItem{}

	getBody, _ := ioutil.ReadAll(resp.Body)
	fmt.Println(json.Unmarshal(getBody, &respSummaries))

	keys := []string{"job1", "job2", "job7"}
	for _, key := range keys {
		for _, item := range respSummaries {
			if item.JobID == key {
				itemJ, _ := json.MarshalIndent(item, "", utils.PrettyPrintIndentForJSON)
				fmt.Println(string(itemJ))
			}
		}
	}

	// Output:
	// 200
	// <nil>
	// {
	//     "shared": false,
	//     "params": {
	//         "pmcsCount": 313,
	//         "name": "alllll",
	//         "dataBucket": "dev-pixlise-data",
	//         "datasetPath": "Datasets/rtt-456/dataset.bin",
	//         "datasetID": "rtt-456",
	//         "jobBucket": "dev-pixlise-piquant-jobs",
	//         "detectorConfig": "PIXL",
	//         "elements": [
	//             "Sc",
	//             "Cr"
	//         ],
	//         "parameters": "-q,pPIETXCFsr -b,0,12,60,910,2800,16",
	//         "runTimeSec": 60,
	//         "coresPerNode": 6,
	//         "startUnixTime": 1592968112,
	//         "creator": {
	//             "name": "peternemere",
	//             "user_id": "5de45d85ca40070f421a3a34",
	//             "email": ""
	//         },
	//         "roiID": "",
	//         "elementSetID": "",
	//         "piquantVersion": "3.0.3",
	//         "quantMode": "Combined",
	//         "comments": "",
	//         "roiIDs": []
	//     },
	//     "elements": [
	//         "Sc",
	//         "Cr"
	//     ],
	//     "jobId": "job1",
	//     "status": "error",
	//     "message": "Error when converting combined CSV to PIXLISE bin format: Failed to determine detector ID from filename column: ",
	//     "endUnixTime": 1592968196,
	//     "outputFilePath": "",
	//     "piquantLogList": null
	// }
	// {
	//     "shared": false,
	//     "params": {
	//         "pmcsCount": 1769,
	//         "name": "ase12",
	//         "dataBucket": "dev-pixlise-data",
	//         "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
	//         "datasetID": "rtt-456",
	//         "jobBucket": "dev-pixlise-piquant-jobs",
	//         "detectorConfig": "PIXL",
	//         "elements": [
	//             "Ru",
	//             "Cr"
	//         ],
	//         "parameters": "-q,pPIETXCFsr -b,0,12,60,910,2800,16",
	//         "runTimeSec": 120,
	//         "coresPerNode": 6,
	//         "startUnixTime": 1591826489,
	//         "creator": {
	//             "name": "tom",
	//             "user_id": "5e3b3bc480ee5c191714d6b7",
	//             "email": ""
	//         },
	//         "roiID": "",
	//         "elementSetID": "",
	//         "piquantVersion": "",
	//         "quantMode": "AB",
	//         "comments": "",
	//         "roiIDs": []
	//     },
	//     "elements": [],
	//     "jobId": "job2",
	//     "status": "nodes_running",
	//     "message": "Node count: 1, Files/Node: 1770",
	//     "endUnixTime": 0,
	//     "outputFilePath": "",
	//     "piquantLogList": null
	// }
	// {
	//     "shared": false,
	//     "params": {
	//         "pmcsCount": 93,
	//         "name": "in progress quant",
	//         "dataBucket": "dev-pixlise-data",
	//         "datasetPath": "Datasets/rtt-123/5x5dataset.bin",
	//         "datasetID": "rtt-123",
	//         "jobBucket": "dev-pixlise-piquant-jobs",
	//         "detectorConfig": "PIXL",
	//         "elements": [
	//             "Sc",
	//             "Cr"
	//         ],
	//         "parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
	//         "runTimeSec": 120,
	//         "coresPerNode": 6,
	//         "startUnixTime": 1589948988,
	//         "creator": {
	//             "name": "peternemere",
	//             "user_id": "600f2a0806b6c70071d3d174",
	//             "email": ""
	//         },
	//         "roiID": "ZcH49SYZ",
	//         "elementSetID": "",
	//         "piquantVersion": "",
	//         "quantMode": "",
	//         "comments": "",
	//         "roiIDs": []
	//     },
	//     "elements": [],
	//     "jobId": "job7",
	//     "status": "complete",
	//     "message": "Something about the nodes",
	//     "endUnixTime": 1592968196,
	//     "outputFilePath": "",
	//     "piquantLogList": null
	// }
}

func Example_quantHandler_List() {
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/quantification/rtt-456", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "summaries": [
	//         {
	//             "shared": false,
	//             "params": {
	//                 "pmcsCount": 93,
	//                 "name": "my test quant",
	//                 "dataBucket": "dev-pixlise-data",
	//                 "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
	//                 "datasetID": "rtt-456",
	//                 "jobBucket": "dev-pixlise-piquant-jobs",
	//                 "detectorConfig": "PIXL",
	//                 "elements": [
	//                     "Sc",
	//                     "Cr"
	//                 ],
	//                 "parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
	//                 "runTimeSec": 120,
	//                 "coresPerNode": 6,
	//                 "startUnixTime": 1589948988,
	//                 "creator": {
	//                     "name": "peternemere",
	//                     "user_id": "600f2a0806b6c70071d3d174",
	//                     "email": ""
	//                 },
	//                 "roiID": "ZcH49SYZ",
	//                 "elementSetID": "",
	//                 "piquantVersion": "",
	//                 "quantMode": "",
	//                 "comments": "",
	//                 "roiIDs": []
	//             },
	//             "elements": [],
	//             "jobId": "job1",
	//             "status": "complete",
	//             "message": "Nodes ran: 1",
	//             "endUnixTime": 1589949035,
	//             "outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	//             "piquantLogList": [
	//                 "node00001_stdout.log",
	//                 "node00001_threads.log"
	//             ]
	//         },
	//         {
	//             "shared": false,
	//             "params": {
	//                 "pmcsCount": 93,
	//                 "name": "in progress quant",
	//                 "dataBucket": "dev-pixlise-data",
	//                 "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
	//                 "datasetID": "rtt-456",
	//                 "jobBucket": "dev-pixlise-piquant-jobs",
	//                 "detectorConfig": "PIXL",
	//                 "elements": [
	//                     "Sc",
	//                     "Cr"
	//                 ],
	//                 "parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
	//                 "runTimeSec": 120,
	//                 "coresPerNode": 6,
	//                 "startUnixTime": 1589948988,
	//                 "creator": {
	//                     "name": "peternemere",
	//                     "user_id": "600f2a0806b6c70071d3d174",
	//                     "email": ""
	//                 },
	//                 "roiID": "ZcH49SYZ",
	//                 "elementSetID": "",
	//                 "piquantVersion": "",
	//                 "quantMode": "",
	//                 "comments": "",
	//                 "roiIDs": []
	//             },
	//             "elements": [
	//                 "Cr03",
	//                 "Sc"
	//             ],
	//             "jobId": "job5",
	//             "status": "nodes_running",
	//             "message": "Something about the nodes",
	//             "endUnixTime": 0,
	//             "outputFilePath": "",
	//             "piquantLogList": null
	//         },
	//         {
	//             "shared": true,
	//             "params": {
	//                 "pmcsCount": 93,
	//                 "name": "my test quant",
	//                 "dataBucket": "dev-pixlise-data",
	//                 "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
	//                 "datasetID": "rtt-456",
	//                 "jobBucket": "dev-pixlise-piquant-jobs",
	//                 "detectorConfig": "PIXL",
	//                 "elements": [
	//                     "Sc",
	//                     "Cr"
	//                 ],
	//                 "parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
	//                 "runTimeSec": 120,
	//                 "coresPerNode": 6,
	//                 "startUnixTime": 1589948988,
	//                 "creator": {
	//                     "name": "peternemere",
	//                     "user_id": "600f2a0806b6c70071d3d174",
	//                     "email": ""
	//                 },
	//                 "roiID": "ZcH49SYZ",
	//                 "elementSetID": "",
	//                 "piquantVersion": "",
	//                 "quantMode": "",
	//                 "comments": "",
	//                 "roiIDs": []
	//             },
	//             "elements": [],
	//             "jobId": "shared-job2",
	//             "status": "complete",
	//             "message": "Nodes ran: 1",
	//             "endUnixTime": 1589949035,
	//             "outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	//             "piquantLogList": [
	//                 "node00001_stdout.log",
	//                 "node00001_threads.log"
	//             ]
	//         }
	//     ],
	//     "blessedQuant": {
	//         "version": 3,
	//         "blessedAt": 1234567690,
	//         "userId": "555555",
	//         "userName": "Michael Da Santa",
	//         "jobId": "shared-job2"
	//     }
	// }
}

func Example_quantHandler_Get() {
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
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job1.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/600f2a0806b6c70071d3d174/rtt-456/Quantifications/summary-job1.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job7.json"),
		},
		{
			Bucket: aws.String(DatasetsBucketForUnitTest), Key: aws.String("Datasets/rtt-456/summary.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/shared/rtt-456/Quantifications/summary-job7.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte("bad json"))),
		},
		nil,
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		nil,
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
			Body: ioutil.NopCloser(bytes.NewReader([]byte(summaryJSON))),
		},
		nil,
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
		"elementSetID": ""
	},
	"elements": ["Sc", "Cr"],
	"jobId": "job7",
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

	//signer := awsutil.MockSigner{[]string{"http://url1.com", "http://url2.com"}}
	//svcs := MakeMockSvcs(&mockS3, nil, &signer)
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	mockUser := pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Permissions: map[string]bool{
			"access:wrong-group": true,
		},
	}
	svcs.JWTReader = MockJWTReader{InfoToReturn: &mockUser}
	apiRouter := MakeRouter(svcs)

	// Dataset summary file wrong group, ERROR - NO ACCESS
	req, _ := http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	mockUser = pixlUser.UserInfo{
		Name:   "Niko Bellic",
		UserID: "600f2a0806b6c70071d3d174",
		Permissions: map[string]bool{
			"access:the-group": true,
		},
	}

	// Dataset summary has different group, ACCESS DENIED
	req, _ = http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Failed to parse summary JSON, ERROR
	req, _ = http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File not found, ERROR
	req, _ = http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// File found, OK
	req, _ = http.NewRequest("GET", "/quantification/rtt-456/job1", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Shared file not found, ERROR
	req, _ = http.NewRequest("GET", "/quantification/rtt-456/shared-job7", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Shared file found, OK
	req, _ = http.NewRequest("GET", "/quantification/rtt-456/shared-job7", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 403
	// dataset rtt-456 not permitted
	//
	// 500
	// failed to verify dataset group permission
	//
	// 404
	// rtt-456 not found
	//
	// 404
	// job1 not found
	//
	// 200
	// {
	//     "summary": {
	//         "shared": false,
	//         "params": {
	//             "pmcsCount": 93,
	//             "name": "my test quant",
	//             "dataBucket": "dev-pixlise-data",
	//             "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
	//             "datasetID": "rtt-456",
	//             "jobBucket": "dev-pixlise-piquant-jobs",
	//             "detectorConfig": "PIXL",
	//             "elements": [
	//                 "Sc",
	//                 "Cr"
	//             ],
	//             "parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
	//             "runTimeSec": 120,
	//             "coresPerNode": 6,
	//             "startUnixTime": 1589948988,
	//             "creator": {
	//                 "name": "peternemere",
	//                 "user_id": "600f2a0806b6c70071d3d174",
	//                 "email": ""
	//             },
	//             "roiID": "ZcH49SYZ",
	//             "elementSetID": "",
	//             "piquantVersion": "",
	//             "quantMode": "AB",
	//             "comments": "",
	//             "roiIDs": []
	//         },
	//         "elements": [],
	//         "jobId": "job1",
	//         "status": "complete",
	//         "message": "Nodes ran: 1",
	//         "endUnixTime": 1589949035,
	//         "outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	//         "piquantLogList": [
	//             "https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
	//             "https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
	//         ]
	//     },
	//     "url": "https:///quantification/download/rtt-456/job1"
	// }
	//
	// 404
	// job7 not found
	//
	// 200
	// {
	//     "summary": {
	//         "shared": false,
	//         "params": {
	//             "pmcsCount": 93,
	//             "name": "my test quant",
	//             "dataBucket": "dev-pixlise-data",
	//             "datasetPath": "Datasets/rtt-456/5x5dataset.bin",
	//             "datasetID": "rtt-456",
	//             "jobBucket": "dev-pixlise-piquant-jobs",
	//             "detectorConfig": "PIXL",
	//             "elements": [
	//                 "Sc",
	//                 "Cr"
	//             ],
	//             "parameters": "-q,pPIETXCFsr -b,0,12,60,910,280,16",
	//             "runTimeSec": 120,
	//             "coresPerNode": 6,
	//             "startUnixTime": 1589948988,
	//             "creator": {
	//                 "name": "peternemere",
	//                 "user_id": "600f2a0806b6c70071d3d174",
	//                 "email": ""
	//             },
	//             "roiID": "ZcH49SYZ",
	//             "elementSetID": "",
	//             "piquantVersion": "",
	//             "quantMode": "",
	//             "comments": "",
	//             "roiIDs": []
	//         },
	//         "elements": [
	//             "Sc",
	//             "Cr"
	//         ],
	//         "jobId": "job7",
	//         "status": "complete",
	//         "message": "Nodes ran: 1",
	//         "endUnixTime": 1589949035,
	//         "outputFilePath": "UserContent/user-1/rtt-456/Quantifications",
	//         "piquantLogList": [
	//             "https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_stdout.log",
	//             "https://dev-pixlise-piquant-jobs.s3.us-east-1.amazonaws.com/Jobs/UC2Bchyz/piquant-logs/node00001.pmcs_threads.log"
	//         ]
	//     },
	//     "url": "https:///quantification/download/rtt-456/shared-job7"
	// }
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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
