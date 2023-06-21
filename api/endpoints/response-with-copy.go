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

package endpoints

import (
	"bufio"
	"bytes"
	"errors"
	"net"
	"net/http"
	"strconv"
)

// ResponseWriterWithCopy - Acts like a normal http response writer but stores a copy
// of the written bytes/status, so it can be logged/monitored by a middleware component
type responseWriterWithCopy struct {
	RealWriter http.ResponseWriter
	Body       *bytes.Buffer
	Status     int
}

func (w *responseWriterWithCopy) StatusText() string {
	if w.Status == 0 {
		return "OK"
	}
	return strconv.Itoa(w.Status)
}

func (w *responseWriterWithCopy) Header() http.Header {
	return w.RealWriter.Header()
}

func (w *responseWriterWithCopy) Write(p []byte) (int, error) {
	// Write to our own buffer AND the real one
	w.Body.Write(p)
	return w.RealWriter.Write(p)
}

func (w *responseWriterWithCopy) WriteHeader(statusCode int) {
	w.Status = statusCode
	w.RealWriter.WriteHeader(statusCode)
}

func (w *responseWriterWithCopy) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	h, ok := w.RealWriter.(http.Hijacker)
	if !ok {
		return nil, nil, errors.New("hijack not supported")
	}
	return h.Hijack()
}
