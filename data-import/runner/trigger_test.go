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

package runner

import (
	"fmt"
)

// Trigger for a manual dataset regeneration (user clicks save button on dataset edit page)
func Example_decodeImportTrigger_Manual() {
	trigger := `{
	"datasetaddons": {
		"dir": "dataset-addons/189137412/custom-meta.json",
		"log": "dataimport-zmzddoytch2krd7n"
	}
}`

	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nLog: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, logID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: "189137412"
	// Log: "dataimport-zmzddoytch2krd7n"
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

func Example_decodeImportTrigger_ManualBadPath() {
	trigger := `{
	"datasetaddons": {
		"dir": "dataset-addons/readme.txt",
		"log": "dataimport-zmzddoytch2krd7n"
	}
}`

	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nLog: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, logID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Log: ""
	// Err: "Failed to find dataset ID from path: dataset-addons/readme.txt"
}

func Example_decodeImportTrigger_ManualErrors() {
	trigger := `{
	"datasetaddons": {
		"dir": "dataset-a
}`

	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nLog: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, logID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Log: ""
	// Err: "Failed to decode dataset addon trigger: invalid character '\n' in string literal"
}

func Example_decodeImportTrigger_OCS_Error() {
	trigger := `{
		"Records": []
}`
	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nLog: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, logID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Log: ""
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

	sourceBucket, sourceFilePath, datasetID, logID, err := decodeImportTrigger([]byte(trigger))
	fmt.Printf("Source Bucket: \"%v\"\nSource file: \"%v\"\nDataset: \"%v\"\nLog: \"%v\"\nErr: \"%v\"\n", sourceBucket, sourceFilePath, datasetID, logID, err)

	// Output:
	// Source Bucket: ""
	// Source file: ""
	// Dataset: ""
	// Log: ""
	// Err: "Failed to decode dataset import trigger: Failed to decode sqs body to an S3 event: unexpected end of JSON input"
}
