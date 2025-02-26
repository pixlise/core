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

package awsutil

import "fmt"

func Example_getEventype() {

	var e Event

	s := `{
    "Records": [
        {
            "eventVersion": "2.1",
            "eventSource": "aws:s3",
            "awsRegion": "us-east-1",
            "eventTime": "2022-06-22T14:36:07.988Z",
            "eventName": "ObjectCreated:CompleteMultipartUpload",
            "userIdentity": {
                "principalId": "AWS:AIDA6AOWGDOHF37MOKWLS"
            },
            "requestParameters": {
                "sourceIPAddress": "81.151.138.139"
            },
            "responseElements": {
                "x-amz-request-id": "PN134P5DBY0KJG2G",
                "x-amz-id-2": "bNfJtmP9ASZO++y92UKMgOrnNb2nF2BxG5lpxBj7N+05Iwq7qn+xtitbnifKJR2zQNPUQVN5lyQTTyDEX0ib1Y3t+bs/P9bH"
            },
            "s3": {
                "s3SchemaVersion": "1.0",
                "configurationId": "MTY5MDg4MjMtNGVkZS00MjQyLTlhN2MtZDU0N2RiNTRmNzAx",
                "bucket": {
                    "name": "stagepipeline-rawdata202c7bd0-dmjs9376duys",
                    "ownerIdentity": {
                        "principalId": "AP902Y0PI20DF"
                    },
                    "arn": "arn:aws:s3:::stagepipeline-rawdata202c7bd0-dmjs9376duys"
                },
                "object": {
                    "key": "130089473-08-03-2022-19-24-00.zip",
                    "size": 41407836,
                    "eTag": "b34552c7ddea5f4fd266f0d1d9fa7116-5",
                    "sequencer": "0062B328C0F22C48E1"
                }
            }
        }
    ]
}`
	t := e.getEventType([]byte(s))

	fmt.Printf("%v\n", t)
	// Output:
	// 1
}

func Example_unmarshalJSON() {

	var e Event

	s := `{
    "Records": [
        {
            "eventVersion": "2.1",
            "eventSource": "aws:s3",
            "awsRegion": "us-east-1",
            "eventTime": "2022-06-22T14:36:07.988Z",
            "eventName": "ObjectCreated:CompleteMultipartUpload",
            "userIdentity": {
                "principalId": "AWS:AIDA6AOWGDOHF37MOKWLS"
            },
            "requestParameters": {
                "sourceIPAddress": "81.151.138.139"
            },
            "responseElements": {
                "x-amz-request-id": "PN134P5DBY0KJG2G",
                "x-amz-id-2": "bNfJtmP9ASZO++y92UKMgOrnNb2nF2BxG5lpxBj7N+05Iwq7qn+xtitbnifKJR2zQNPUQVN5lyQTTyDEX0ib1Y3t+bs/P9bH"
            },
            "s3": {
                "s3SchemaVersion": "1.0",
                "configurationId": "MTY5MDg4MjMtNGVkZS00MjQyLTlhN2MtZDU0N2RiNTRmNzAx",
                "bucket": {
                    "name": "stagepipeline-rawdata202c7bd0-dmjs9376duys",
                    "ownerIdentity": {
                        "principalId": "AP902Y0PI20DF"
                    },
                    "arn": "arn:aws:s3:::stagepipeline-rawdata202c7bd0-dmjs9376duys"
                },
                "object": {
                    "key": "130089473-08-03-2022-19-24-00.zip",
                    "size": 41407836,
                    "eTag": "b34552c7ddea5f4fd266f0d1d9fa7116-5",
                    "sequencer": "0062B328C0F22C48E1"
                }
            }
        }
    ]
}`
	e.UnmarshalJSON([]byte(s))

	fmt.Printf("%v\n", e.Records[0].EventSource)
	// Output:
	// aws:s3
}
