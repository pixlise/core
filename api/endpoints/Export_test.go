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

	"github.com/pixlise/core/v2/core/awsutil"
)

func Example_registerExportHandlerSunny() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	var exp MockExporter
	exp.downloadReturn = []byte{80, 101, 116, 101, 114, 32, 105, 115, 32, 97, 119, 101, 115, 111, 109, 101}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Exporter = &exp
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/export/files/983561", bytes.NewReader([]byte(`{
		"fileName": "test.zip",
		"quantificationId": "abc123",
		"fileIds": ["quant-maps-csv", "rois"]
	}`)))
	resp := executeRequest(req, apiRouter.Router)
	fmt.Println(resp.Code)

	fmt.Println(exp.datasetID)
	fmt.Println(exp.userID)
	fmt.Println(exp.quantID)
	fmt.Println(exp.fileIDs)

	fmt.Println(resp.Header().Get("Content-Type"))
	fmt.Println(resp.Header().Get("Content-Length"))
	fmt.Println(string(resp.Body.Bytes()))

	// Output:
	// 200
	// 983561
	// 600f2a0806b6c70071d3d174
	// abc123
	// [quant-maps-csv rois]
	// application/octet-stream
	// 16
	// Peter is awesome
}

// Same result as if the POST body is completely empty
func Example_registerExportHandlerMissingFileName() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	var exp MockExporter
	exp.downloadReturn = []byte{80, 101, 116, 101, 114, 32, 105, 115, 32, 97, 119, 101, 115, 111, 109, 101}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Exporter = &exp
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/export/files/983561", bytes.NewReader([]byte(`{
		"fileIds": ["quant-maps-csv", "rois"]
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Print(string(resp.Body.Bytes()))

	// Output:
	// 400
	// File name must end in .zip
}

func Example_registerExportHandlerMissingColumn() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	var exp MockExporter
	exp.downloadReturn = []byte{80, 101, 116, 101, 114, 32, 105, 115, 32, 97, 119, 101, 115, 111, 109, 101}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Exporter = &exp
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/export/files/983561", bytes.NewReader([]byte(`{
		"fileName": "test.zip",
		"quantificationId": "abc123"
	}`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Print(string(resp.Body.Bytes()))

	// Output:
	// 400
	// No File IDs specified, nothing to export
}

func Example_registerExportHandlerBadJSONBody() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	var exp MockExporter
	exp.downloadReturn = []byte{80, 101, 116, 101, 114, 32, 105, 115, 32, 97, 119, 101, 115, 111, 109, 101}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
	svcs.Exporter = &exp
	apiRouter := MakeRouter(svcs)

	req, _ := http.NewRequest("POST", "/export/files/983561", bytes.NewReader([]byte(`{
		"quantificati,`)))
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Print(string(resp.Body.Bytes()))

	// Output:
	// 400
	// unexpected end of JSON input
}
