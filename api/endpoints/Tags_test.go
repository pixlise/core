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
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/timestamper"
)

const tagsUserS3Path = "UserContent/shared/Tags.json"

func Example_tagHandler_List() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(tagsUserS3Path),
		},
	}

	// By minifying response, don't have to worry about whitespace or tab differences
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(minifyJSON(`
				{
					"test-id": {
						"id": "test-id",
						"name": "new_tag_test",
						"creator": {
							"name": "Ryan Stonebraker",
							"user_id": "6227d96292150a0069117483",
							"email": "ryan.a.stonebraker@jpl.nasa.gov"
						},
						"dateCreated": 1670623334,
						"type": "expression",
						"datasetID": "123456"
					}
				}`)))),
		},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	apiRouter := MakeRouter(svcs)

	// User requests list of all tags
	req, _ := http.NewRequest("GET", "/tags/123456", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(minifyJSON(resp.Body.String()))

	// Output:
	// 200
	// {"test-id":{"id":"test-id","name":"new_tag_test","creator":{"name":"Ryan Stonebraker","user_id":"6227d96292150a0069117483","email":"ryan.a.stonebraker@jpl.nasa.gov"},"dateCreated":1670623334,"type":"expression","datasetID":"123456"}}
}

func Example_tagHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(tagsUserS3Path),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(`{}`))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(tagsUserS3Path), Body: bytes.NewReader([]byte(standardizeJSON(`{
				"test-new-id": {
					"id": "test-new-id",
					"name": "testing_post_item",
					"creator": {
						"name": "Niko Bellic",
						"user_id": "600f2a0806b6c70071d3d174",
						"email": "niko@spicule.co.uk"
					},
					"dateCreated": 1670623334,
					"type": "expression",
					"datasetID": "123456"
				}
			}`))),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	var idGen MockIDGenerator
	idGen.ids = []string{"test-new-id"}
	svcs := MakeMockSvcs(&mockS3, &idGen, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1670623334},
	}

	apiRouter := MakeRouter(svcs)

	postItem := minifyJSON(`{
		"id": "test-new-id",
		"name": "testing_post_item",
		"dateCreated": 1670623334,
		"type": "expression",
		"datasetID": "123456"
	}`)

	// Posts a new tag
	req, _ := http.NewRequest("POST", "/tags/123456", bytes.NewReader([]byte(postItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// {
	//     "id": "test-new-id"
	// }
}

func Example_tagHandler_Delete() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(tagsUserS3Path),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(tagsUserS3Path),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(standardizeJSON(`{
				"test-new-id": {
					"id": "test-new-id",
					"name": "testing_post_item",
					"creator": {
						"name": "Niko Bellic",
						"user_id": "600f2a0806b6c70071d3d174",
						"email": "niko@spicule.co.uk"
					},
					"dateCreated": 1670623334,
					"type": "expression",
					"datasetID": "123456"
				},
				"some-other-id": {
					"id": "some-other-id",
					"name": "some_tag",
					"creator": {
						"name": "Niko Bellic",
						"user_id": "600f2a0806b6c70071d3d174",
						"email": "niko@spicule.co.uk"
					},
					"dateCreated": 1670623334,
					"type": "expression",
					"datasetID": "123456"
				}
			}`)))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(standardizeJSON(`{
				"test-other-creator-id": {
					"id": "test-other-creator-id",
					"name": "testing_post_item",
					"creator": {
						"name": "Some Other User",
						"user_id": "1234567",
						"email": "email@email.com"
					},
					"dateCreated": 1670623334,
					"type": "expression",
					"datasetID": "123456"
				}
			}`)))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String(tagsUserS3Path), Body: bytes.NewReader([]byte(standardizeJSON(`{
				"some-other-id": {
					"id": "some-other-id",
					"name": "some_tag",
					"creator": {
						"name": "Niko Bellic",
						"user_id": "600f2a0806b6c70071d3d174",
						"email": "niko@spicule.co.uk"
					},
					"dateCreated": 1670623334,
					"type": "expression",
					"datasetID": "123456"
				}
			}`))),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)

	apiRouter := MakeRouter(svcs)

	// Tries to delete a tag the user owns and only deletes that tag from the tag list
	req, _ := http.NewRequest("DELETE", "/tags/123456/test-new-id", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Tries to delete a tag the user doesn't own, should fail
	req, _ = http.NewRequest("DELETE", "/tags/123456/test-other-creator-id", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	//
	// 401
	// test-other-creator-id not owned by 600f2a0806b6c70071d3d174
}
