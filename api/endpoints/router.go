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
	"github.com/gorilla/mux"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/api/services"
)

func MakeRouter(svcs services.APIServices) apiRouter.ApiObjectRouter {
	router := mux.NewRouter() //.StrictSlash(true)
	// Should we use StrictSlash??

	apiRouter := apiRouter.NewAPIRouter(&svcs, router)

	registerVersionHandler(&apiRouter)
	registerDataExpressionHandler(&apiRouter)
	registerDatasetHandler(&apiRouter)
	registerElementSetHandler(&apiRouter)
	registerROIHandler(&apiRouter)
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
