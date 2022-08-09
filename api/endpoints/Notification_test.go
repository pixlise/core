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
	"time"

	"github.com/pixlise/core/core/notifications"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/core/awsutil"
)

const emptyUserJSON = `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[],"hints":[],"uinotifications":[]},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"","data_collection":"unknown"}}`

const userJSON = `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[{"name":"topic c","config":{"method":{"ui":true,"sms":false,"email":false}}},{"name":"topic d","config":{"method":{"ui":true,"sms":false,"email":false}}}],"hints":[],"uinotifications":[]},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"","data_collection":"unknown"}}`

const hintJSON = `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[{"name":"topic c","config":{"method":{"ui":true,"sms":false,"email":false}}},{"name":"topic d","config":{"method":{"ui":true,"sms":false,"email":false}}}],"hints":["hint c","hint d"],"uinotifications":[]},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"","data_collection":"unknown"}}`

const userSMSEMAILJSON = `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[{"name":"topic c","config":{"method":{"ui":true,"sms":true,"email":true}}},{"name":"topic d","config":{"method":{"ui":true,"sms":true,"email":true}}}],"hints":[],"uinotifications":[]},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"","data_collection":"unknown"}}`

const userJSONNotification = `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[{"name":"topic c","config":{"method":{"ui":true,"sms":true,"email":true}}},{"name":"topic d","config":{"method":{"ui":true,"sms":true,"email":true}}}],"hints":[],"uinotifications":[{"topic":"test-data-source","message":"New Data Source Available","timestamp":"2021-02-01T01:01:01.000Z","userid":"600f2a0806b6c70071d3d174"}]},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"","data_collection":"unknown"}}`

func Example_subscriptions_empty() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"), Body: bytes.NewReader([]byte(emptyUserJSON)),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	setTestAuth0Config(&svcs)

	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/notification/subscriptions", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	// Output:
	// ensure-valid: 200
	// {
	//     "topics": []
	// }
}

func Example_subscriptions() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(userJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(userJSON))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{

			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"), Body: bytes.NewReader([]byte(userJSON)),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	setTestAuth0Config(&svcs)

	apiRouter := MakeRouter(svcs)

	jsonstr := `{"topics": [{
			"name": "topic c",
			"config": {
				"method": {
					"ui": true,
					"sms": false,
					"email": false
				}
			}
		}, {
			"name": "topic d",
			"config": {
				"method": {
					"ui": true,
					"sms": false,
					"email": false
				}
			}
		}]}`
	req, _ := http.NewRequest("POST", "/notification/subscriptions", bytes.NewReader([]byte(jsonstr)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	req, _ = http.NewRequest("GET", "/notification/subscriptions", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	// Output:
	// ensure-valid: 200
	// {
	//     "topics": [
	//         {
	//             "name": "topic c",
	//             "config": {
	//                 "method": {
	//                     "ui": true,
	//                     "sms": false,
	//                     "email": false
	//                 }
	//             }
	//         },
	//         {
	//             "name": "topic d",
	//             "config": {
	//                 "method": {
	//                     "ui": true,
	//                     "sms": false,
	//                     "email": false
	//                 }
	//             }
	//         }
	//     ]
	// }
	// ensure-valid: 200
	// {
	//     "topics": [
	//         {
	//             "name": "topic c",
	//             "config": {
	//                 "method": {
	//                     "ui": true,
	//                     "sms": false,
	//                     "email": false
	//                 }
	//             }
	//         },
	//         {
	//             "name": "topic d",
	//             "config": {
	//                 "method": {
	//                     "ui": true,
	//                     "sms": false,
	//                     "email": false
	//                 }
	//             }
	//         }
	//     ]
	// }
}

func Example_alerts_empty() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// Expecting 1 get for users file
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
	}

	// No file!
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)

	obj := notifications.UINotificationObj{
		Topic:     "test-data-source",
		Message:   "New Data Source Available",
		Timestamp: time.Time{},
		UserID:    "600f2a0806b6c70071d3d174",
	}
	svcs.Notifications.AddNotification(obj)

	setTestAuth0Config(&svcs)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/notification/alerts", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	// Output:
	// ensure-valid: 200
	// []
}

func Example_alerts() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	const userClearedNotification = `{"userid":"600f2a0806b6c70071d3d174","notifications":{"topics":[{"name":"topic c","config":{"method":{"ui":true,"sms":true,"email":true}}},{"name":"topic d","config":{"method":{"ui":true,"sms":true,"email":true}}}],"hints":[],"uinotifications":[]},"userconfig":{"name":"Niko Bellic","email":"niko@spicule.co.uk","cell":"","data_collection":"unknown"}}`

	// Expecting 2 gets for users file
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
	}

	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		// Simulate returning the user file with notifications in it
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(userJSONNotification))),
		},
		// Second get receives the notification-cleared file
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(userClearedNotification))),
		},
	}

	// Expecting a put for the cleared user file
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("UserContent/notifications/600f2a0806b6c70071d3d174.json"), Body: bytes.NewReader([]byte(userClearedNotification)),
		},
	}

	// Simulate returning ok for put
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)

	obj := notifications.UINotificationObj{
		Topic:     "test-data-source",
		Message:   "New Data Source Available",
		Timestamp: time.Time{},
		UserID:    "600f2a0806b6c70071d3d174",
	}
	svcs.Notifications.AddNotification(obj)

	setTestAuth0Config(&svcs)
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/notification/alerts", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	// Output:
	// ensure-valid: 200
	// [
	//     {
	//         "topic": "test-data-source",
	//         "message": "New Data Source Available",
	//         "timestamp": "2021-02-01T01:01:01Z",
	//         "userid": "600f2a0806b6c70071d3d174"
	//     }
	// ]
	// ensure-valid: 200
	// []
}

func Example_hints_empty() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// User requesting their hints
	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
	}

	// We're saying there's no hint file in S3
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		nil,
	}

	// Expect API to upload a hints file
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"), Body: bytes.NewReader([]byte(emptyUserJSON)),
		},
	}

	// Mocking empty OK response from put call
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	setTestAuth0Config(&svcs)

	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("GET", "/notification/hints", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	// Output:
	// ensure-valid: 200
	// {
	//     "hints": []
	// }
}

func Example_hints() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	mockS3.ExpGetObjectInput = []s3.GetObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"),
		},
	}
	mockS3.QueuedGetObjectOutput = []*s3.GetObjectOutput{
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(hintJSON))),
		},
		{
			Body: ioutil.NopCloser(bytes.NewReader([]byte(hintJSON))),
		},
	}

	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest), Key: aws.String("/UserContent/notifications/600f2a0806b6c70071d3d174.json"), Body: bytes.NewReader([]byte(hintJSON)),
		},
	}

	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
	setTestAuth0Config(&svcs)

	apiRouter := MakeRouter(svcs)

	jsonstr := `{
				"hints": [      
					"hint c",
					"hint d"
				]}`
	req, _ := http.NewRequest("POST", "/notification/hints", bytes.NewReader([]byte(jsonstr)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	req, _ = http.NewRequest("GET", "/notification/hints", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Printf("ensure-valid: %v\n", resp.Code)
	fmt.Printf("%v", resp.Body)

	// Output:
	// ensure-valid: 200
	// {
	//     "hints": [
	//         "hint c",
	//         "hint d"
	//     ]
	// }
	// ensure-valid: 200
	// {
	//     "hints": [
	//         "hint c",
	//         "hint d"
	//     ]
	// }
}
