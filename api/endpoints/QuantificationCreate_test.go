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
