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
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/pixlUser"
)

const UrlStreamDownloadIndicator = "download"

// If it's a handler that streams a file from S3 to the client, use this
type ApiHandlerStreamParams struct {
	Svcs       *services.APIServices
	UserInfo   pixlUser.UserInfo
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
		var userInfo pixlUser.UserInfo
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
					err = api.MakeNotFoundError(name)
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
						log.Printf("Download of \"%s\" complete. Wrote %v bytes\n", name, bytesWritten)
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
		var userInfo pixlUser.UserInfo
		userInfo, err = h.APIServices.JWTReader.GetUserInfo(r)

		if err == nil {
			// Get the S3 object and file name we're streaming
			var result *s3.GetObjectOutput
			var name string
			result, name, err = h.Stream(ApiHandlerStreamParams{h.APIServices, userInfo, pathParams, r.Header})
			if err != nil {
				if h.FS.IsNotFoundError(err) {
					err = api.MakeNotFoundError(name)
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
					err = fmt.Errorf("Error copying file to the http response %s", err.Error())
				} else {
					log.Printf("Download of \"%s\" complete. Wrote %v bytes", name, bytesWritten)
				}
			}
		}
	}

	if err != nil {
		logHandlerErrors(err, h.APIServices.Log, w, r)
	}
}
