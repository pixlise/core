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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"

	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/expressions/modules"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// DataModules - storing/retrieving modules for expressions to call

const idVersion = "version"

func registerDataModuleHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "data-module"

	// Listing
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermPublic), dataModuleList)
	// Getting an individual module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier, idVersion), apiRouter.MakeMethodPermission("GET", permission.PermPublic), dataModuleGet)
	// Adding a new module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), dataModulePost)
	// Adding a new version for a module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), dataModulePut)
	// NOTE: you cannot delete a module!
}

func dataModuleList(params handlers.ApiHandlerParams) (interface{}, error) {
	filteredModules := modules.DataModuleWireLookup{}

	isPublicUser := !params.UserInfo.Permissions[permission.PermReadDataAnalysis]
	allModules, err := params.Svcs.Expressions.ListModules(true)
	if err != nil {
		return filteredModules, err
	}

	if isPublicUser {
		publicObjectsAuth, err := permission.GetPublicObjectsAuth(params.Svcs.FS, params.Svcs.Config.ConfigBucket, isPublicUser)
		if err != nil {
			return nil, err
		}

		// Filter out any modules that are not public
		for _, mod := range allModules {
			isModPublic, err := permission.CheckIsObjectInPublicSet(publicObjectsAuth.Modules, mod.ID)
			if err != nil {
				return nil, err
			}

			if isModPublic {
				fmt.Println("MOD IS PUBLIC, ADDING", mod.ID)
				filteredModules[mod.ID] = mod
			}
		}
	} else {
		// No filtering needed
		filteredModules = allModules
	}

	return filteredModules, nil
}

func dataModuleGet(params handlers.ApiHandlerParams) (interface{}, error) {
	modID := params.PathParams[idIdentifier]
	version := params.PathParams[idVersion]

	isPublicUser := !params.UserInfo.Permissions[permission.PermReadDataAnalysis]
	if isPublicUser {
		isModulePublic, err := permission.CheckIsObjectPublic(params.Svcs.FS, params.Svcs.Config.ConfigBucket, permission.PublicObjectModule, modID)
		if err != nil {
			return nil, err
		}

		if !isModulePublic {
			return nil, api.MakeBadRequestError(errors.New("module is not public"))
		}
	}

	var ver *modules.SemanticVersion

	if len(version) > 0 {
		verParsed, err := modules.SemanticVersionFromString(version)
		if err != nil {
			return nil, api.MakeBadRequestError(fmt.Errorf("Invalid version specified: %v", err))
		}
		ver = &verParsed
	}

	return params.Svcs.Expressions.GetModule(modID, ver, true)
}

func dataModulePost(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	// A new DOI is published to Zenodo if the "publish_doi" query parameter is true
	publishDOI := params.Request.URL.Query().Get("publish_doi") == "true"

	var req modules.DataModuleInput
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	if !modules.IsValidModuleName(req.Name) {
		return modules.DataModuleSpecificVersionWire{}, api.MakeBadRequestError(fmt.Errorf("Invalid module name: %v. Must only contain letters, under-score, numbers (but not in first character), and be less than 20 characters long.", req.Name))
	}

	if len(req.SourceCode) <= 0 {
		return modules.DataModuleSpecificVersionWire{}, api.MakeBadRequestError(errors.New("Source code field cannot be empty"))
	}

	return params.Svcs.Expressions.CreateModule(req, params.UserInfo, publishDOI)
}

func dataModulePut(params handlers.ApiHandlerParams) (interface{}, error) {
	modID := params.PathParams[idIdentifier]

	// A new DOI is published to Zenodo if the "publish_doi" query parameter is true
	publishDOI := params.Request.URL.Query().Get("publish_doi") == "true"

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req modules.DataModuleVersionInput
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	return params.Svcs.Expressions.AddModuleVersion(modID, req, publishDOI)
}
