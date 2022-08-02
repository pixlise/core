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
                "principalId": "AWS:"
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
                "principalId": "AWS:"
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
