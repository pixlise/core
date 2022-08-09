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
