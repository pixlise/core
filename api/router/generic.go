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

package apiRouter

import (
	"net/http"

	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/jwtparser"
)

// If all else fails, use this. Is the most generic handler, passes in request & response writer like raw ServeHTTP
// but also passed the parsed user info & path params
type ApiHandlerGenericParams struct {
	Svcs       *services.APIServices
	UserInfo   jwtparser.JWTUserInfo
	PathParams map[string]string
	Writer     http.ResponseWriter
	Request    *http.Request
}
type ApiHandlerGenericFunc func(ApiHandlerGenericParams) error
type ApiHandlerGeneric struct {
	*services.APIServices
	Handler ApiHandlerGenericFunc
}

func (h ApiHandlerGeneric) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParams := makePathParams(h.APIServices, r)

	userInfo, err := h.APIServices.JWTReader.GetUserInfo(r)
	if err == nil {
		err = h.Handler(ApiHandlerGenericParams{h.APIServices, userInfo, pathParams, w, r})
	} else {
		err = errorwithstatus.MakeBadRequestError(err)
	}

	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
	}
}
