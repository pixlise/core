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

// The guts of PIXLISE API endpoint handler/routing code. Allows us to define a router, permissions, and services to be used
// by code that processes HTTP requests.
package apiRouter

import (
	"net/http"

	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/mux"
	"github.com/pixlise/core/v4/api/permission"
	"github.com/pixlise/core/v4/api/services"
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

func (r *ApiObjectRouter) AddGenericHandler(path string, methodPerm MethodPermission, handleFunc ApiHandlerGenericFunc) {
	r.addHandler(path, methodPerm, &ApiHandlerGeneric{APIServices: r.Svcs, Handler: handleFunc})
}

// Not used yet in this project
// func (r *ApiObjectRouter) AddStreamHandler(path string, methodPerm MethodPermission, handleFunc ApiStreamHandlerFunc) {
// 	r.addHandler(path, methodPerm, &ApiStreamFromS3Handler{APIServices: r.Svcs, Stream: handleFunc})
// }

func (r *ApiObjectRouter) AddCacheControlledStreamHandler(path string, methodPerm MethodPermission, handleFunc ApiCacheControlledStreamHandlerFunc) {
	r.addHandler(path, methodPerm, &ApiCacheControlledStreamFromS3Handler{APIServices: r.Svcs, Stream: handleFunc})
}

/*
	func (r *ApiObjectRouter) AddJSONHandler(path string, methodPerm MethodPermission, handleFunc handlers.ApiHandlerFunc) {
		r.addHandler(path, methodPerm, &handlers.ApiHandlerJSON{APIServices: r.Svcs, Handler: handleFunc})
	}

	func (r *ApiObjectRouter) AddShareHandler(path string, methodPerm MethodPermission, handleFunc handlers.ApiHandlerFunc) {
		r.addHandler(path, methodPerm, &handlers.ApiSharingHandler{APIServices: r.Svcs, Share: handleFunc})
	}
*/

func (r *ApiObjectRouter) AddPublicHandler(path string, method string, handleFunc ApiHandlerGenericPublicFunc) {
	r.addHandler(path, MethodPermission{method, permission.PermPublic}, &ApiHandlerGenericPublic{APIServices: r.Svcs, Handler: handleFunc})
}

func (r *ApiObjectRouter) addHandler(path string, methodPerm MethodPermission, handler http.Handler) {
	handlerToSave := handler

	// If needed, wrap in a sentry handler
	if r.Svcs.Config.EnvironmentName != "unittest" && r.Svcs.Config.EnvironmentName != "local" {
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
