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

package errorwithstatus

import (
	"fmt"
	"net/http"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Neater error handling

// See:
// https://blog.questionable.services/article/http-handler-error-handling-revisited/
// https://golang.org/pkg/net/http/#Handler
// https://github.com/gorilla/mux

// Error represents a handler error. It provides methods for a HTTP status
// code and embeds the built-in error interface.
type Error interface {
	error
	Status() int
}

// api.StatusError represents an error with an associated HTTP status code.
type StatusError struct {
	Code int
	Err  error
}

// Allows api.StatusError to satisfy the error interface.
func (se StatusError) Error() string {
	return se.Err.Error()
}

// Status - Returns our HTTP status code.
func (se StatusError) Status() int {
	return se.Code
}

// Some common errors
func MakeNotFoundError(ID string) StatusError {
	return StatusError{
		Code: http.StatusNotFound,
		Err:  fmt.Errorf("%v not found", ID),
	}
}

func MakeBadRequestError(err error) StatusError {
	return StatusError{
		Code: http.StatusBadRequest,
		Err:  err,
	}
}

func MakeUnauthorisedError(err error) StatusError {
	return StatusError{
		Code: http.StatusUnauthorized,
		Err:  err,
	}
}

// Mainly so we don't get a bunch of errors for not using field names in StatusError{}
func MakeStatusError(code int, err error) StatusError {
	return StatusError{
		Code: code,
		Err:  err,
	}
}
