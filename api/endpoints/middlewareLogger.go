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
	"bytes"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/logger"
)

// How many chars of request body to display in logs
const bodyTextReqLogLength = 200

// How many chars of resp body to display in logs
const bodyTextRespLogHeadLength = 600

// How many chars of resp body to display in logs
const bodyTextRespLogTailLength = 300

// If req/resp body is longer than the limits, we print this ti show it was cut off
const logSnipIndicator = "\n    ---- >8 -------- >8 -------- >8 -------- >8 ----\n"

// loggingObject - Object used for logging user activity
type loggingObject struct {
	Instance    string
	Time        time.Time
	Component   string
	Message     string
	Response    string
	Version     string
	Params      map[string]interface{}
	Environment string
	User        string
}

type LoggerMiddleware struct {
	*services.APIServices
	JwtValidator jwtparser.JWTInterface
}

func (h *LoggerMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Read the HTTP body. We can log it here if required, and then we pass it into the next in chain
		bodyBytes, err := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		fullReqBodyText := "REQ BODY ERROR"
		reqBodyText := fullReqBodyText

		if err == nil {
			fullReqBodyText = string(bodyBytes)
		}
		if h.Config.LogLevel == logger.LogDebug {
			// We write the whole request, and body to log...
			reqBodyText = fullReqBodyText
			// Display a part of the body
			if len(reqBodyText) > bodyTextReqLogLength {
				reqBodyText = reqBodyText[0:bodyTextReqLogLength] + logSnipIndicator
			}
		}

		// Create a multiwriter, so we can write to the http response AND store it so we can log it
		buf := new(bytes.Buffer)
		w2 := &responseWriterWithCopy{RealWriter: w, Body: buf, Status: 0}

		next.ServeHTTP(w2, r)

		// We only log if we're in debug log level OR we detected an error
		hadError := w2.Status != 0 && w2.Status != http.StatusOK && w2.Status != http.StatusNotModified

		// Write body in debug output, if we can
		contType := w2.RealWriter.Header().Get("Content-Type")
		respBodyTxt := ""
		fullRespBodyText := ""
		// We're not that strict on content types, basically if it's not set it's probably a download, if it is set it's probably
		// text we can log, though octet-stream is definitely a special case we don't want to log.
		// TODO: Improve content types so this check can be made more accurate
		if len(contType) > 0 && contType != "application/octet-stream" { //contType == "application/json" || contType == "application/text" {
			fullRespBodyText = string(buf.Bytes())
			respBodyTxt = fullRespBodyText

			if len(fullRespBodyText) > bodyTextRespLogHeadLength+bodyTextRespLogTailLength {
				respBodyTxt = fullRespBodyText[0:bodyTextRespLogHeadLength] +
					logSnipIndicator +
					fullRespBodyText[len(fullRespBodyText)-bodyTextRespLogTailLength:] +
					"^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^^\n"
			}
		} else {
			respBodyTxt = fmt.Sprintf("Body data length: %v bytes", buf.Len())
		}

		jwtValidator := jwtparser.RealJWTReader{Validator: h.JwtValidator}
		requestingUser, _ := jwtValidator.GetSimpleUserInfo(r)

		//level := logger.LogDebug
		if hadError {
			//level = logger.LogError

			// TODO: Don't think this will ever get called, there is no sentry hub inited for this middleware or above, sentry seems
			// to have been attached to individual handlers in addHandler
			if hub := sentry.GetHubFromContext(r.Context()); hub != nil {
				hub.WithScope(func(scope *sentry.Scope) {
					params := make(map[string]interface{})
					params["method"] = r.Method
					kv := r.URL.Query()

					for i, s := range kv {
						params["queryparam"+i] = s
					}

					for name, values := range r.Header {
						// Loop over all values for the name.
						vals := strings.Join(values, "; ")
						params[name] = vals
					}
					for k, v := range params {
						scope.SetExtra(k, v)
					}
					hub.CaptureMessage("Error detected in http request")
				})
			}
			msg := fmt.Sprintf("API returned %v for %v \"%v %v\", query params: %v. Requesting user id: \"%v\", name: \"%v\". Response body: \"%v\"",
				w2.Status,
				r.Method,
				r.Host,
				r.URL,
				r.URL.Query(),
				requestingUser.UserID,
				requestingUser.Name,
				respBodyTxt,
			)
			// This always showed an exception for errors.errorString or whatever, what we have here is a message to print to sentry...
			//sentry.CaptureException(errors.New(msg))

			// Filter out common spam errors that we don't want to know about. For example, getting 500 on an API call due
			// to client disconnecting while downloading. This resulted in heaps of spam that must only be caused by someone
			// closing their tab/laptop during downloading stuff.
			if w2.Status == 500 && (strings.Contains(respBodyTxt, "connection reset by peer") || strings.Contains(respBodyTxt, "broken pipe")) ||
				w2.Status == 404 && r.Method == "GET" && strings.Contains(r.URL.String(), "/logger/fetch/") {
				h.Log.Errorf("Decided error is spam, not sending to SENTRY. Message: %v", msg)
			} else {
				sentry.CaptureMessage(msg)
			}
		}

		// Don't log requests to / as some load balancer seems to be doing this constantly, so we lose all other logs
		// in the sea of requests to /
		// Also, don't log all requests to notification alerts, because this endpoint is being polled by PIXLISE
		// so we'd lose other logs in a sea of requests to it.
		if r.URL.Path != "/" {
			kv := r.URL.Query()
			params := make(map[string]interface{})
			params["method"] = r.Method
			for i, s := range kv {
				params["queryparam"+i] = s
			}

			for name, values := range r.Header {
				// Loop over all values for the name.
				vals := strings.Join(values, "; ")
				params[name] = vals
			}
			/*
				track := false

				if val, ok := h.Notifications.GetTrack(requestingUser.UserID); ok {
					track = val
				} else {
					user, err := h.APIServices.Users.GetUser(requestingUser.UserID)
					if err != nil {
						h.Notifications.SetTrack(requestingUser.UserID, false)
						return
					}

					if user.Config.DataCollection != "unknown" && user.Config.DataCollection != "false" {
						track = true
						h.Notifications.SetTrack(requestingUser.UserID, true)
					}
				}
				if track && len(contType) > 0 && (contType == "application/json" || strings.HasPrefix(contType, "text")) {
					go func() {
						// Read time & id this way so it's mockable
						currentTime := time.Unix(h.TimeStamper.GetTimeNowSec(), 0).UTC()
						datestamp := currentTime.Format("2006-01-02")

						id := h.IDGen.GenObjectID()

						o := loggingObject{
							Time:        currentTime,
							Component:   r.URL.Path,
							Message:     fullReqBodyText,
							Response:    fullRespBodyText,
							Params:      params,
							Environment: h.APIServices.Config.EnvironmentName,
							User:        requestingUser.UserID,
						}

						writePath := path.Join(filepaths.RootUserActivity, datestamp, id+".json")
						err := h.FS.WriteJSON(h.Config.UsersBucket, writePath, o)
						if err != nil {
							h.Log.Errorf("Failure to write to activity bucket: %v", err)
						}
					}()

					if hadError || h.Config.LogLevel == logger.LogDebug {
						h.Log.Printf(level, "Request: %v (%v), body: %v\nResponse status: %v, body: %v", r.URL, r.Method, reqBodyText, w2.StatusText(), respBodyTxt)
					}
				}*/
		}
	})
}
