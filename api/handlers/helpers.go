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

package handlers

import (
	"fmt"
	"net/http"

	"github.com/gorilla/mux"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/logger"
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
