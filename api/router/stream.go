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
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/jwtparser"
)

const UrlStreamDownloadIndicator = "download"

// If it's a handler that streams a file from S3 to the client, use this
type ApiHandlerStreamParams struct {
	Svcs       *services.APIServices
	UserInfo   jwtparser.JWTUserInfo
	PathParams map[string]string
	Headers    http.Header
}
type ApiStreamHandlerFunc func(ApiHandlerStreamParams) (*s3.GetObjectOutput, string, error)
type ApiCacheControlledStreamHandlerFunc func(ApiHandlerStreamParams) (*s3.GetObjectOutput, string, string, string, int, error)
type ApiStreamFromS3Handler struct {
	*services.APIServices
	Stream ApiStreamHandlerFunc
}
type ApiCacheControlledStreamFromS3Handler struct {
	*services.APIServices
	Stream ApiCacheControlledStreamHandlerFunc
}

func (h ApiCacheControlledStreamFromS3Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParams := makePathParams(h.APIServices, r)

	var err error
	if r.Method != "GET" {
		err = errors.New("Stream must be GET")
	}

	if err == nil {
		isStreamPath := false
		pathbits := strings.Split(r.URL.Path, "/")
		// If an intermediate part contains indicates this is a stream GET...
		for c, bit := range pathbits {
			// NOTE: c > 1 because [0] is "", [1] is the root path, eg /dataset, so we want to check after there
			if c > 1 && c < len(pathbits)-1 && bit == UrlStreamDownloadIndicator {
				isStreamPath = true
				break
			}
		}

		if !isStreamPath {
			err = errors.New("Stream has invalid path")
		}
	}

	if err == nil {
		var userInfo jwtparser.JWTUserInfo
		userInfo, err = h.APIServices.JWTReader.GetUserInfo(r)

		if err == nil {
			// Get the S3 object and file name we're streaming
			var result *s3.GetObjectOutput
			var name string
			var etag string
			var status int
			var lastmodified string
			result, name, etag, lastmodified, status, err = h.Stream(ApiHandlerStreamParams{h.APIServices, userInfo, pathParams, r.Header})
			if err != nil {
				if h.FS.IsNotFoundError(err) {
					err = errorwithstatus.MakeNotFoundError(name)
				}
			}

			if err == nil {
				// Set up to stream this S3 object to caller
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
				//w.Header().Set("Cache-Control", "no-store")

				if etag == "" {
					w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%v", downloadCacheMinMaxAgeSec))
				} else {
					w.Header().Set("Etag", etag)
					//w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%v", downloadCacheMaxAgeSec))
					//w.Header().Set("Vary", "Accept-Encoding")
				}
				if lastmodified != "" {
					w.Header().Set("last-modified", lastmodified)
				}

				if status != 0 {
					w.WriteHeader(status)
				}

				if result != nil {
					w.Header().Set("Content-Length", fmt.Sprintf("%v", *result.ContentLength))
					var bytesWritten int64
					bytesWritten, err = io.Copy(w, result.Body)
					if err != nil {
						err = fmt.Errorf("Error copying file to the http response %s", err.Error())
					} else {
						h.APIServices.Log.Debugf("Download of \"%s\" complete. Wrote %v bytes", name, bytesWritten)
					}
				}
			}
		}
	}

	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
	}
}
func (h ApiStreamFromS3Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	pathParams := makePathParams(h.APIServices, r)

	var err error
	if r.Method != "GET" {
		err = errors.New("Stream must be GET")
	}

	if err == nil {
		isStreamPath := false
		pathbits := strings.Split(r.URL.Path, "/")
		// If an intermediate part contains indicates this is a stream GET...
		for c, bit := range pathbits {
			// NOTE: c > 1 because [0] is "", [1] is the root path, eg /dataset, so we want to check after there
			if c > 1 && c < len(pathbits)-1 && bit == UrlStreamDownloadIndicator {
				isStreamPath = true
				break
			}
		}

		if !isStreamPath {
			err = errors.New("Stream has invalid path")
		}
	}

	if err == nil {
		var userInfo jwtparser.JWTUserInfo
		userInfo, err = h.APIServices.JWTReader.GetUserInfo(r)

		if err == nil {
			// Get the S3 object and file name we're streaming
			var result *s3.GetObjectOutput
			var name string
			result, name, err = h.Stream(ApiHandlerStreamParams{h.APIServices, userInfo, pathParams, r.Header})
			if err != nil {
				if h.FS.IsNotFoundError(err) {
					err = errorwithstatus.MakeNotFoundError(name)
				}
			}

			if err == nil {
				// Set up to stream this S3 object to caller
				w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
				//w.Header().Set("Cache-Control", "no-store")
				w.Header().Set("Cache-Control", fmt.Sprintf("max-age=%v", downloadCacheMaxAgeSec))
				if result.ContentLength != nil {
					w.Header().Set("Content-Length", fmt.Sprintf("%v", *result.ContentLength))
				}

				var bytesWritten int64
				bytesWritten, err = io.Copy(w, result.Body)
				if err != nil {
					err = fmt.Errorf("error copying file to the http response %s", err.Error())
				} else {
					h.APIServices.Log.Debugf("Download of \"%s\" complete. Wrote %v bytes", name, bytesWritten)
				}
			}
		}
	}

	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
	}
}
