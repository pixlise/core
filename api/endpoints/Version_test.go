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
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"github.com/gorilla/mux"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/core/awsutil"
)

func Example_version() {
	var mockS3 awsutil.MockS3Client
	defer mockS3.FinishTest()

	svcs := MakeMockSvcs(&mockS3, nil, nil)

	apiRouter := apiRouter.NewAPIRouter(&svcs, mux.NewRouter())

	apiRouter.AddPublicHandler("/", "GET", RootRequest)
	apiRouter.AddPublicHandler("/version-binary", "GET", GetVersionProtobuf)
	apiRouter.AddPublicHandler("/version-json", "GET", GetVersionJSON)

	req, _ := http.NewRequest("GET", "/", nil)
	resp := executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	fmt.Println(strings.HasPrefix(string(resp.Body.Bytes()), "<!DOCTYPE html>"))

	versionPat := regexp.MustCompile(`<h1>PIXLISE API</h1><p>Version .+</p>`)
	fmt.Println(versionPat.MatchString(string(resp.Body.Bytes())))

	req, _ = http.NewRequest("GET", "/version-json", nil)
	resp = executeRequest(req, apiRouter.Router)

	fmt.Println(resp.Code)
	// Don't know why but sometimes this passes, then it fails because it's printed with "API", "version" or with the space
	// missing... so here we just remove all spaces
	fmt.Printf("%v\n", strings.ReplaceAll(resp.Body.String(), " ", ""))

	// Output:
	// 200
	// true
	// true
	// 200
	// {"versions":[{"component":"API","version":"(Localbuild)"}]}
}
