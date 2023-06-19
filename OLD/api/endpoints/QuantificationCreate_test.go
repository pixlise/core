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

// TODO: finish this test... ironic that our most important API endpoint is the one that's not tested! But we do
// unit test many parts of this, writing an overall test didn't seem important/easy

/*
func Example_quantHandler_Post() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	mockS3.ExpPutObjectInput = []s3.PutObjectInput{

		s3.PutObjectInput{
			Bucket: aws.String("job-bucket"), Key: aws.String(quantPath + "/params.json"), Body: bytes.NewReader([]byte(`{
    "pmcs": [
        123,
        456
    ],
    "name": "test_quant",
    "dataBucket": "data-bucket",
    "datasetPath": "....",
    "jobBucket": "job-bucket",
    "detectorConfig": "...",
    "elements": [
        "...."
    ],
    "parameters": "unittests=true",
    "runTimeSec": 100,
    "coresPerNode": 6,
    "startUnixTime": 1589233229
}`))},
		s3.PutObjectInput{
			Bucket: aws.String("job-bucket"), Key: aws.String(quantPath + "/status.json"), Body: bytes.NewReader([]byte(`{
    "jobId": "9790d15a-a7e4-443f-8256-01d44520ee36",
    "status": "starting",
    "message": "Job started",
    "endUnixTime": 0,
    "outputFilePath": "",
    "piquantLogList": null
}`))},
		//	s3.PutObjectInput{
		//			Bucket: aws.String("job-bucket"), Key: aws.String(quantPath+"/status.json"), Body: bytes.NewReader([]byte(`{
		//    "jobId": "9790d15a-a7e4-443f-8256-01d44520ee36",
		//    "status": "preparing_nodes",
		//    "message": "Node count: 1, Cores/Node: 6, PMCs/Node: 3",
		//    "endUnixTime": 0,
		//    "outputFilePath": "",
		//    "piquantLogList": null
		//}`))},
	}
	mockS3.QueuedPutObjectOutput = []*s3.PutObjectOutput{
		&s3.PutObjectOutput{},
		&s3.PutObjectOutput{},
	}
	svcs := MakeMockSvcs(&mockS3, nil, nil)
	router := MakeRouter(svcs)
	req, _ := http.NewRequest("POST", "/quantification/rtt-123", bytes.NewReader([]byte(`{
	"quant_name":"test_quant",
	"dataset_path":"....",
	"pmcs": [123,456],
	"elements":["...."],
	"detectorconfig":"...",
	"parameters":"unittests=true",
	"runtimesec":100
}`)))
	resp := executeRequest(req, router)
	fmt.Println(resp.Code)
	fmt.Println(resp.Body)

	// Output:
	// 200
	// "9790d15a-a7e4-443f-8256-01d44520ee36"
}
*/
