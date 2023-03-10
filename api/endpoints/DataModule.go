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

	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/expressions/modules"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// DataModules - storing/retrieving modules for expressions to call

const idVersion = "version"

func registerDataModuleHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "data-module"

	// Listing
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), dataModuleList)
	// Getting an individual module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier, idVersion), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), dataModuleGet)
	// Adding a new module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), dataModulePost)
	// Adding a new version for a module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), dataModulePut)
	// NOTE: you cannot delete a module!
}

func dataModuleList(params handlers.ApiHandlerParams) (interface{}, error) {
	return params.Svcs.Expressions.ListModules(true)
}

func dataModuleGet(params handlers.ApiHandlerParams) (interface{}, error) {
	modID := params.PathParams[idIdentifier]
	version := params.PathParams[idVersion]

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

	return params.Svcs.Expressions.CreateModule(req, params.UserInfo)
}

func dataModulePut(params handlers.ApiHandlerParams) (interface{}, error) {
	modID := params.PathParams[idIdentifier]

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req modules.DataModuleVersionInput
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	return params.Svcs.Expressions.AddModuleVersion(modID, req)
}
