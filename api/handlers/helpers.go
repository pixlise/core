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
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/logger"
)

const HostParamName = "hostname"

// Helper functions for the above handlers
func makePathParams(svcs *services.APIServices, r *http.Request) map[string]string {
	// Get path params
	pathParams := mux.Vars(r)
	queries := r.URL.Query()
	for q, v := range queries {
		if len(v) > 0 {
			pathParams[q] = v[0] // we ignore subsequent ones
		}
	}

	// Set the host name in case anything needs it
	// TODO: get the host name in some sane way
	if svcs.Config.EnvironmentName == "local" {
		pathParams[HostParamName] = "http://" + r.Host
	} else {
		pathParams[HostParamName] = "https://" + r.Host
	}

	return pathParams
}

func logHandlerErrors(err error, log logger.ILogger, w http.ResponseWriter, r *http.Request) {
	switch e := err.(type) {
	case api.Error:
		// We can retrieve the status here and write out a specific
		// HTTP status code.
		log.Errorf("Request: %v (%v), Result: status=%v, error=%v", r.URL, r.Method, e.Status(), e)
		http.Error(w, e.Error(), e.Status())
	default:
		log.Errorf("Request: %v (%v), Result: status=%v, error=%v", r.URL, r.Method, http.StatusInternalServerError, e)

		// Any error types we don't specifically look out for default
		// to serving a HTTP 500
		http.Error(w, fmt.Sprintf("%v", e), http.StatusInternalServerError)
	}
}
