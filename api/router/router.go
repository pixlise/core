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

package apiRouter

import (
	"net/http"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/mux"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	"github.com/pixlise/core/api/services"
)

type MethodPermission struct {
	Method     string
	Permission string
}

func MakeMethodPermission(method string, permission string) MethodPermission {
	return MethodPermission{Method: method, Permission: permission}
}

type RouteMethodPermissions map[string]string

type ApiObjectRouter struct {
	Permissions RouteMethodPermissions
	Svcs        *services.APIServices
	Router      *mux.Router
}

func NewAPIRouter(svcs *services.APIServices, router *mux.Router) ApiObjectRouter {
	return ApiObjectRouter{RouteMethodPermissions{}, svcs, router}
}

func (r *ApiObjectRouter) AddGenericHandler(path string, methodPerm MethodPermission, handleFunc handlers.ApiHandlerGenericFunc) {
	r.addHandler(path, methodPerm, &handlers.ApiHandlerGeneric{APIServices: r.Svcs, Handler: handleFunc})
}

func (r *ApiObjectRouter) AddJSONHandler(path string, methodPerm MethodPermission, handleFunc handlers.ApiHandlerFunc) {
	r.addHandler(path, methodPerm, &handlers.ApiHandlerJSON{APIServices: r.Svcs, Handler: handleFunc})
}

func (r *ApiObjectRouter) AddShareHandler(path string, methodPerm MethodPermission, handleFunc handlers.ApiHandlerFunc) {
	r.addHandler(path, methodPerm, &handlers.ApiSharingHandler{APIServices: r.Svcs, Share: handleFunc})
}

func (r *ApiObjectRouter) AddStreamHandler(path string, methodPerm MethodPermission, handleFunc handlers.ApiStreamHandlerFunc) {
	r.addHandler(path, methodPerm, &handlers.ApiStreamFromS3Handler{APIServices: r.Svcs, Stream: handleFunc})
}

func (r *ApiObjectRouter) AddCacheControlledStreamHandler(path string, methodPerm MethodPermission, handleFunc handlers.ApiCacheControlledStreamHandlerFunc) {
	r.addHandler(path, methodPerm, &handlers.ApiCacheControlledStreamFromS3Handler{APIServices: r.Svcs, Stream: handleFunc})
}

func (r *ApiObjectRouter) AddPublicHandler(path string, method string, handleFunc handlers.ApiHandlerGenericPublicFunc) {
	r.addHandler(path, MethodPermission{method, permission.PermPublic}, &handlers.ApiHandlerGenericPublic{APIServices: r.Svcs, Handler: handleFunc})
}

func (r *ApiObjectRouter) addHandler(path string, methodPerm MethodPermission, handler http.Handler) {
	handlerToSave := handler

	// If needed, wrap in a sentry handler
	if r.Svcs.Config.EnvironmentName != "unit-test" && r.Svcs.Config.EnvironmentName != "local" {
		sentryHandler := sentryhttp.New(sentryhttp.Options{
			Repanic:         true,
			WaitForDelivery: true,
		})

		handlerToSave = sentryHandler.Handle(handler)
	}

	methodRoute := methodPerm.Method + path

	// Save to permissions table
	_, ok := r.Permissions[methodRoute]
	if ok {
		r.Svcs.Log.Errorf("Path handler already defined for: %v, method: %v", path, methodPerm.Method)
		return
	}

	r.Permissions[methodRoute] = methodPerm.Permission

	// Add to router
	r.Router.Handle(path, handlerToSave).Methods(methodPerm.Method)
}

func (r *ApiObjectRouter) GetPermissions() RouteMethodPermissions {
	return r.Permissions
}
