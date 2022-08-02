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
	return services.ApiVersion
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
		result.Components = append(result.Components, ComponentVersion{
			Component:        "PIQUANT",
			Version:          piquantVersion,
			BuildUnixTimeSec: 0,
		})
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
