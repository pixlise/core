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

// api - package for containing "core" API things, which are reusable
// in building any API for our platform. These should not contain
// specific PIXLISE API business logic
package api

import (
	"encoding/json"
	"fmt"
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
	m := (values[0]).(*map[string]interface{}) //map[string]interface{}{}

	//m["https://pixlise.org/username"] = "12345"

	fmt.Printf("%v", m)
	//values= append(values, m)
	b := []byte(`{"https://pixlise.org/username":"12345", "sub": "myuserid"}`)
	for _, d := range values {
		if err := json.Unmarshal(b, d); err != nil {
			return err
		}
	}
	return nil
}
