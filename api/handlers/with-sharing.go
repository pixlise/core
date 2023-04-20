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
	"errors"
	"net/http"

	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/pixlUser"
)

// If it's a share function, use this. Enforces that method is only POST
type ApiSharingHandler struct {
	*services.APIServices
	Share ApiHandlerFunc
}

func (h ApiSharingHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParams := makePathParams(h.APIServices, r)

	var err error
	if r.Method != "POST" {
		err = errors.New("Share must be POST")
	} else {
		var userInfo pixlUser.UserInfo
		userInfo, err = h.APIServices.JWTReader.GetUserInfo(r)

		if err == nil {
			resp, errShare := h.Share(ApiHandlerParams{h.APIServices, userInfo, pathParams, r})
			err = errShare
			if err == nil {
				api.ToJSON(w, resp)
			}
		}
	}

	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
	}
}
