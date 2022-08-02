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

// Mainly so we don't get a bunch of errors for not using field names in StatusError{}
func MakeStatusError(code int, err error) StatusError {
	return StatusError{
		Code: code,
		Err:  err,
	}
}
