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

	"gitlab.com/pixlise/pixlise-go-api/api/services"
)

// As with generic handler, but for public API endpoints ONLY
type ApiHandlerGenericPublicParams struct {
	Svcs       *services.APIServices
	PathParams map[string]string
	Writer     http.ResponseWriter
	Request    *http.Request
}
type ApiHandlerGenericPublicFunc func(ApiHandlerGenericPublicParams) error
type ApiHandlerGenericPublic struct {
	*services.APIServices
	Handler ApiHandlerGenericPublicFunc
}

func (h ApiHandlerGenericPublic) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParams := makePathParams(h.APIServices, r)

	err := h.Handler(ApiHandlerGenericPublicParams{h.APIServices, pathParams, w, r})
	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
	}
}
