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

import (
	"bytes"
	"fmt"
	"net/http"

	"github.com/pixlise/core/core/awsutil"
)

func Example_registerExportHandlerSunny() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()
	var exp MockExporter
	exp.downloadReturn = []byte{80, 101, 116, 101, 114, 32, 105, 115, 32, 97, 119, 101, 115, 111, 109, 101}

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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

	svcs := MakeMockSvcs(&mockS3, nil, nil, nil, nil)
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
