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

package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
)

func checkFileDownload(JWT string, url string) ([]byte, error) {
	getReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	getReq.Header.Set("Authorization", "Bearer "+JWT)

	getResp, err := http.DefaultClient.Do(getReq)
	if err != nil {
		return nil, err
	}
	defer getResp.Body.Close()
	body, err := ioutil.ReadAll(getResp.Body)
	if err != nil {
		return nil, err
	}

	if getResp.Status != "200 OK" {
		return nil, fmt.Errorf("Download status fail: %v, response: %v", getResp.Status, string(body))
	}

	headerBytes, err := strconv.Atoi(getResp.Header["Content-Length"][0])
	if err != nil {
		return nil, fmt.Errorf("Failed to read content-length: %v", err)
	}

	//comparing bytes of body and content length header
	if headerBytes != len(body) {
		return nil, fmt.Errorf("Content-length: %v does not match downloaded size: %v", headerBytes, len(body))
	}

	return body, nil
}
