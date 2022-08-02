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
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/core/detector"
	"github.com/pixlise/core/core/piquant"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Getting detector config
// PIXLISE downloads a detect-specific config file when a dataset is opened. This contains settings
// that are relevant to that piece of detector hardware. It also contains a list of PIQUANT config
// versions, so when PIQUANT is called on, the user can pick from versions of the configuration files

// Config we send over wire (containing the PIQUANT config versions)
type detectorConfigWire struct {
	*detector.DetectorConfig
	PIQUANTConfigVersions []string `json:"piquantConfigVersions"`
}

func registerDetectorConfigHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "detector-config"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadPiquantConfig), detectorConfigGet)
}

func detectorConfigGet(params handlers.ApiHandlerParams) (interface{}, error) {
	// It's a get, we don't care about the body...

	// Using path params, work out path
	configName := params.PathParams[idIdentifier]

	// Download config & return it
	cfg, err := detector.ReadDetectorConfig(params.Svcs, configName)
	if err != nil {
		return nil, err
	}

	// Get a list of PIQUANT config versions too
	versions := piquant.GetPiquantConfigVersions(params.Svcs, configName)

	// Set the versions
	result := detectorConfigWire{
		&cfg,
		versions,
	}

	return &result, nil
}
