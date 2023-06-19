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

package api

import (
	"encoding/json"
	"net/http"

	"gopkg.in/square/go-jose.v2/jwt"
)

type MockJWTValidator struct {
}

func (v *MockJWTValidator) ValidateRequest(r *http.Request) (*jwt.JSONWebToken, error) {
	//var j jwt.JSONWebToken

	return nil, nil
}
func (v *MockJWTValidator) Claims(r *http.Request, token *jwt.JSONWebToken, values ...interface{}) error {
	//m := (values[0]).(*map[string]interface{}) //map[string]interface{}{}
	//m["https://pixlise.org/username"] = "12345"
	//fmt.Printf("MockJWTValidator first value: %v", m)

	//values= append(values, m)
	b := []byte(`{"https://pixlise.org/username":"12345", "sub": "myuserid"}`)
	for _, d := range values {
		if err := json.Unmarshal(b, d); err != nil {
			return err
		}
	}
	return nil
}
