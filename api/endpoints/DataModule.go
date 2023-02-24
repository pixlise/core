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
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// DataModules - storing/retrieving modules for expressions to call

func registerDataModuleHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "data-module"

	// Listing
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), dataModuleList)
	// Getting an individual module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), dataModuleGet)
	// Adding a new module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), dataModulePost)
	// Adding a new version for a module
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), dataModulePut)
	// NOTE: you cannot delete a module!
}

func dataModuleList(params handlers.ApiHandlerParams) (interface{}, error) {
	return nil, nil
}

func dataModuleGet(params handlers.ApiHandlerParams) (interface{}, error) {
	return nil, nil
}

func dataModulePost(params handlers.ApiHandlerParams) (interface{}, error) {
	return nil, nil
}

func dataModulePut(params handlers.ApiHandlerParams) (interface{}, error) {
	return nil, nil
}
