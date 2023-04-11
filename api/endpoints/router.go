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
	"github.com/gorilla/mux"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/api/services"
)

func MakeRouter(svcs services.APIServices) apiRouter.ApiObjectRouter {
	router := mux.NewRouter() //.StrictSlash(true)
	// Should we use StrictSlash??

	apiRouter := apiRouter.NewAPIRouter(&svcs, router)

	registerVersionHandler(&apiRouter)
	registerDataExpressionHandler(&apiRouter)
	registerDataModuleHandler(&apiRouter)
	registerDatasetHandler(&apiRouter)
	registerElementSetHandler(&apiRouter)
	registerROIHandler(&apiRouter)
	registerTagHandler(&apiRouter)
	registerAnnotationHandler(&apiRouter)
	registerDetectorConfigHandler(&apiRouter)
	registerViewStateHandler(&apiRouter)
	registerExportHandler(&apiRouter)
	registerMetricsHandler(&apiRouter)
	registerQuantificationHandler(&apiRouter)
	registerPiquantHandler(&apiRouter)
	registerUserManagementHandler(&apiRouter)
	registerNotificationHandler(&apiRouter)
	registerTestHandler(&apiRouter)
	registerDiffractionHandler(&apiRouter)
	registerRGBMixHandler(&apiRouter)
	registerLoggerHandler(&apiRouter)

	return apiRouter
}
