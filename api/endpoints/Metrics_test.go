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
	"net/http"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/timestamper"
)

func Example_registerMetricsHandlerTest() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	// NOTE: PUT expected JSON needs to have spaces not tabs
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{
		{
			Bucket: aws.String(UsersBucketForUnitTest),
			Key:    aws.String("Activity/2023-03-16/metric-button-600f2a0806b6c70071d3d174-1678938381.json"),
			Body:   bytes.NewReader([]byte(`{"name": "something", "counter": 3, "comment": "lala"}`)),
		},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		{},
	}
	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1678938381},
	}
	apiRouter := MakeRouter(svcs)

	postItem := `{"name": "something", "counter": 3, "comment": "lala"}`

	// POST without ID should fail
	req, _ := http.NewRequest("POST", "/metrics", bytes.NewReader([]byte(postItem)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(string(resp.Body.Bytes()))

	req, _ = http.NewRequest("POST", "/metrics/button", bytes.NewReader([]byte(postItem)))
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(string(resp.Body.Bytes()))

	// Output:
	// 404
	// 404 page not found
	//
	// 200
	//
}
