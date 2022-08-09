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
	"errors"
	"fmt"
	"strings"

	"github.com/pixlise/core/api/handlers"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/piquant"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Getting component versions

// ComponentVersion is getting versions of stuff in API, public because it's used in integration test
type ComponentVersion struct {
	Component        string `json:"component"`
	Version          string `json:"version"`
	BuildUnixTimeSec int32  `json:"build-unix-time-sec"`
}

// ComponentVersionsGetResponse is wrapper of above
type ComponentVersionsGetResponse struct {
	Components []ComponentVersion `json:"components"`
}

func getAPIVersion() string {
	if len(services.ApiVersion) <= 0 {
		return "N/A - Local build"
	}

	ver := services.ApiVersion
	if len(services.GitHash) > 8 {
		ver += "-" + services.GitHash[0:8]
	}

	return ver
}

func registerVersionHandler(router *apiRouter.ApiObjectRouter) {
	// User goes to root of API, returns HTML
	router.AddPublicHandler("/", "GET", rootRequest)

	// User requesting version as JSON
	router.AddPublicHandler("/version", "GET", componentVersionsGet)
}

func componentVersionsGet(params handlers.ApiHandlerGenericPublicParams) error {
	var result ComponentVersionsGetResponse

	result.Components = []ComponentVersion{
		{
			Component:        "API",
			Version:          getAPIVersion(),
			BuildUnixTimeSec: 0,
		},
	}

	piquantVersion := ""

	ver, err := piquant.GetPiquantVersion(params.Svcs)
	if err == nil {
		parts := strings.Split(ver.Version, "/")
		if len(parts) > 0 {
			piquantVersion = parts[len(parts)-1]
		}
	} else {
		return err
	}

	if len(piquantVersion) > 0 {
		ver := ComponentVersion{
			Component:        "PIQUANT",
			Version:          piquantVersion,
			BuildUnixTimeSec: 0,
		}

		result.Components = append(result.Components, ver)
	} else {
		return errors.New("Failed to determine configured PIQUANT version")
	}

	api.ToJSON(params.Writer, result)
	return nil
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Root request, which also shows version & a logo

func rootRequest(params handlers.ApiHandlerGenericPublicParams) error {
	params.Writer.Header().Add("Content-Type", "text/html")

	var start string = `<!DOCTYPE html>
<html lang="en"><head></head>
<body style="font-family: Arial, Helvetica, sans-serif">
<center>`
	var midtemplate = "<h1>PIXLISE API</h1><p>Version %s</p><p>Git Commit: %s"
	var mid = fmt.Sprintf(midtemplate, getAPIVersion(), services.GitHash)
	var end string = `</p>
</center>
</body>`

	params.Writer.Write([]byte(start + binchicken + mid + end))
	return nil
}
