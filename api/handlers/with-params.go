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

package handlers

import (
	"net/http"

	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/pixlUser"
)

// If returning JSON, use this
type ApiHandlerParams struct {
	Svcs       *services.APIServices
	UserInfo   pixlUser.UserInfo
	PathParams map[string]string
	Request    *http.Request
}
type ApiHandlerFunc func(ApiHandlerParams) (interface{}, error)

type ApiHandlerJSON struct {
	*services.APIServices
	Handler ApiHandlerFunc
}

func (h ApiHandlerJSON) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParams := makePathParams(h.APIServices, r)

	userInfo, err := h.APIServices.JWTReader.GetUserInfo(r)
	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
		return
	}

	resp, err := h.Handler(ApiHandlerParams{h.APIServices, userInfo, pathParams, r})

	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
		return
	}

	// Save result as JSON
	api.ToJSON(w, resp)
}
