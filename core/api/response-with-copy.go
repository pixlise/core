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

// Contains "core"-level helper code to build APIs. These should
// be reusable in building any API for our platform, and should
// not contain specific PIXLISE API business logic
package api

import (
	"bytes"
	"net/http"
	"strconv"
)

// ResponseWriterWithCopy - Acts like a normal http response writer but stores a copy
// of the written bytes/status, so it can be logged/monitored by a middleware component
type ResponseWriterWithCopy struct {
	RealWriter http.ResponseWriter
	Body       *bytes.Buffer
	Status     int
}

func (w *ResponseWriterWithCopy) StatusText() string {
	if w.Status == 0 {
		return "OK"
	}
	return strconv.Itoa(w.Status)
}

func (w *ResponseWriterWithCopy) Header() http.Header {
	return w.RealWriter.Header()
}

func (w *ResponseWriterWithCopy) Write(p []byte) (int, error) {
	// Write to our own buffer AND the real one
	w.Body.Write(p)
	return w.RealWriter.Write(p)
}

func (w *ResponseWriterWithCopy) WriteHeader(statusCode int) {
	w.Status = statusCode
	w.RealWriter.WriteHeader(statusCode)
}
