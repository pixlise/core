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

package dataimport

import (
	"fmt"
)

// Trigger for a manual dataset regeneration (user clicks save button on dataset edit page)
func Example_decodeImportTrigger_Manual() {
	trigger := `{
	"datasetID": "189137412",
	"jobID": "dataimport-zmzddoytch2krd7n"
}`

	sourceBucket, sourceFilePath, datasetID, jobId, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, jobId, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: "189137412"
	// Job: "dataimport-zmzddoytch2krd7n"
	// Err: "<nil>"
}

// Trigger from when a new zip arrives from the pipeline
func Example_decodeImportTrigger_OCS() {
	trigger := `{
    "Records": [
        {
            "eventVersion": "2.1",
            "eventSource": "aws:s3",
            "awsRegion": "us-east-1",
            "eventTime": "2022-09-16T09:10:28.417Z",
            "eventName": "ObjectCreated:CompleteMultipartUpload",
            "userIdentity": {
                "principalId": "AWS:AIDA6AOWGDOHF37MOKWLS"
            },
            "requestParameters": {
                "sourceIPAddress": "81.154.57.137"
            },
            "responseElements": {
                "x-amz-request-id": "G3QWWT0BAYKP81QK",
                "x-amz-id-2": "qExUWHHDE1nL+UP3zim1XA7FIXRUoKxlIrJt/7ULAtn08/+EvRCt4sChLhCGEqMo7ny4CU/KufMNmOcyZsDPKGWHT2ukMbo+"
            },
            "s3": {
                "s3SchemaVersion": "1.0",
                "configurationId": "OTBjMjZmYzAtYThlOC00OWRmLWIwMzUtODkyZDk0YmRhNzkz",
                "bucket": {
                    "name": "prodpipeline-rawdata202c7bd0-o40ktu17o2oj",
                    "ownerIdentity": {
                        "principalId": "AP902Y0PI20DF"
                    },
                    "arn": "arn:aws:s3:::prodpipeline-rawdata202c7bd0-o40ktu17o2oj"
                },
                "object": {
                    "key": "189137412-07-09-2022-10-07-57.zip",
                    "size": 54237908,
                    "eTag": "b21ebca14f67255be1cd28c01d494508-7",
                    "sequencer": "0063243D6858D568F0"
                }
            }
        }
    ]
}`

	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger([]byte(trigger))

	// NOTE: we're only checking the length of the log ID because it's a timestamp+random chars. Other code has this stubbed out but here it's probably sufficient
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nLog Str Len: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, len(logID), err)

	// Output:
	// Source Bucket: "prodpipeline-rawdata202c7bd0-o40ktu17o2oj"
	// Source file: "189137412-07-09-2022-10-07-57.zip"
	// Dataset: "189137412"
	// Log Str Len: "43"
	// Err: "<nil>"
}

// Trigger from when a new zip arrives from the pipeline
func Example_decodeImportTrigger_OCS2() {
	trigger := `{
    "Records": [
        {
            "eventVersion": "2.1",
            "eventSource": "aws:s3",
            "awsRegion": "us-east-1",
            "eventTime": "2022-09-25T14:33:49.456Z",
            "eventName": "ObjectCreated:Put",
            "userIdentity": {
                "principalId": "AWS:AIDA6AOWGDOHF37MOKWLS"
            },
            "requestParameters": {
                "sourceIPAddress": "3.12.95.94"
            },
            "responseElements": {
                "x-amz-request-id": "K811ZDJ52EYBJ8P2",
                "x-amz-id-2": "R7bGQ2fOjvSZHkHez700w3wRVpn32nmr6jVPVYhKtNE2c2KYOmgm9hjmOA5WSQFh8faLRe6fHAmANKSTNRhwCq7Xgol0DgX4"
            },
            "s3": {
                "s3SchemaVersion": "1.0",
                "configurationId": "OTBjMjZmYzAtYThlOC00OWRmLWIwMzUtODkyZDk0YmRhNzkz",
                "bucket": {
                    "name": "prodpipeline-rawdata202c7bd0-o40ktu17o2oj",
                    "ownerIdentity": {
                        "principalId": "AP902Y0PI20DF"
                    },
                    "arn": "arn:aws:s3:::prodpipeline-rawdata202c7bd0-o40ktu17o2oj"
                },
                "object": {
                    "key": "197329413-25-09-2022-14-33-39.zip",
                    "size": 1388,
                    "eTag": "932bda7d32c05d90ecc550d061862994",
                    "sequencer": "00633066CD68A4BF43"
                }
            }
        }
    ]
}`

	sourceBucket, sourceFilePath, datasetID, jobID, err := decodeImportTrigger([]byte(trigger))

	// NOTE: we're only checking the length of the log ID because it's a timestamp+random chars. Other code has this stubbed out but here it's probably sufficient
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob Str Len: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, len(jobID), err)

	// Output:
	// Source Bucket: "prodpipeline-rawdata202c7bd0-o40ktu17o2oj"
	// Source file: "197329413-25-09-2022-14-33-39.zip"
	// Dataset: "197329413"
	// Job Str Len: "43"
	// Err: "<nil>"
}

// Trigger from when a new zip arrives from the pipeline but pipeline stores it in a subdir of the bucket
func Example_decodeImportTrigger_OCS3() {
	trigger := `{
    "Records": [
        {
            "eventVersion": "2.1",
            "eventSource": "aws:s3",
            "awsRegion": "us-east-1",
            "eventTime": "2022-09-25T14:33:49.456Z",
            "eventName": "ObjectCreated:Put",
            "userIdentity": {
                "principalId": "AWS:AIDA6AOWGDOHF37MOKWLS"
            },
            "requestParameters": {
                "sourceIPAddress": "3.12.95.94"
            },
            "responseElements": {
                "x-amz-request-id": "K811ZDJ52EYBJ8P2",
                "x-amz-id-2": "R7bGQ2fOjvSZHkHez700w3wRVpn32nmr6jVPVYhKtNE2c2KYOmgm9hjmOA5WSQFh8faLRe6fHAmANKSTNRhwCq7Xgol0DgX4"
            },
            "s3": {
                "s3SchemaVersion": "1.0",
                "configurationId": "OTBjMjZmYzAtYThlOC00OWRmLWIwMzUtODkyZDk0YmRhNzkz",
                "bucket": {
                    "name": "prodpipeline-rawdata202c7bd0-o40ktu17o2oj",
                    "ownerIdentity": {
                        "principalId": "AP902Y0PI20DF"
                    },
                    "arn": "arn:aws:s3:::prodpipeline-rawdata202c7bd0-o40ktu17o2oj"
                },
                "object": {
                    "key": "data/197329413-25-09-2022-14-33-39.zip",
                    "size": 1388,
                    "eTag": "932bda7d32c05d90ecc550d061862994",
                    "sequencer": "00633066CD68A4BF43"
                }
            }
        }
    ]
}`

	sourceBucket, sourceFilePath, datasetID, jobID, err := decodeImportTrigger([]byte(trigger))

	// NOTE: we're only checking the length of the log ID because it's a timestamp+random chars. Other code has this stubbed out but here it's probably sufficient
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob Str Len: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, len(jobID), err)

	// Output:
	// Source Bucket: "prodpipeline-rawdata202c7bd0-o40ktu17o2oj"
	// Source file: "data/197329413-25-09-2022-14-33-39.zip"
	// Dataset: "197329413"
	// Job Str Len: "43"
	// Err: "<nil>"
}

func Example_decodeImportTrigger_ManualBadMsg() {
	trigger := `{
	"weird": "message"
}`

	sourceBucket, sourceFilePath, datasetID, jobID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, jobID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Job: ""
	// Err: "Unexpected or no message type embedded in triggering SNS message"
}

func Example_decodeImportTrigger_ManualBadDatasetID() {
	trigger := `{
	"datasetID": "",
	"jobID": "dataimport-zmzddoytch2krd7n"
}`

	sourceBucket, sourceFilePath, datasetID, jobID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, jobID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Job: ""
	// Err: "Failed to find dataset ID in reprocess trigger"
}

func Example_decodeImportTrigger_ManualBadLogID() {
	trigger := `{
		"datasetID": "qwerty"
	}`

	sourceBucket, sourceFilePath, datasetID, jobID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, jobID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Job: ""
	// Err: "Failed to find job ID in reprocess trigger"
}

func Example_decodeImportTrigger_OCS_Error() {
	trigger := `{
		"Records": []
}`
	sourceBucket, sourceFilePath, datasetID, jobID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, jobID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Job: ""
	// Err: "Unexpected or no message type embedded in triggering SNS message"
}

func Example_decodeImportTrigger_OCS_BadEventType() {
	trigger := `{
    "Records": [
        {
            "eventVersion": "2.1",
            "eventSource": "aws:sqs",
            "awsRegion": "us-east-1",
            "eventTime": "2022-09-16T09:10:28.417Z",
            "eventName": "ObjectCreated:CompleteMultipartUpload",
            "userIdentity": {
                "principalId": "AWS:AIDA6AOWGDOHF37MOKWLS"
            },
            "requestParameters": {
                "sourceIPAddress": "81.154.57.137"
            },
            "responseElements": {
                "x-amz-request-id": "G3QWWT0BAYKP81QK",
                "x-amz-id-2": "qExUWHHDE1nL+UP3zim1XA7FIXRUoKxlIrJt/7ULAtn08/+EvRCt4sChLhCGEqMo7ny4CU/KufMNmOcyZsDPKGWHT2ukMbo+"
            }
        }
    ]
}`

	sourceBucket, sourceFilePath, datasetID, jobID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nJob: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, jobID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Job: ""
	// Err: "Failed to decode dataset import trigger: Failed to decode sqs body to an S3 event: unexpected end of JSON input"
}
