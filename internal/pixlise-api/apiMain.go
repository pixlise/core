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

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"math/rand"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/pixlise/core/v3/core/notifications"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	"github.com/gorilla/handlers"

	"github.com/pixlise/core/v3/api/config"
	"github.com/pixlise/core/v3/api/endpoints"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/utils"

	_ "net/http/pprof"

	"github.com/pixlise/core/v3/core/export"

	expressionDB "github.com/pixlise/core/v3/core/expressions/database"
)

func printRoutePermissions(routePermissions map[string]string) {
	// Gather keys
	paths := []string{}
	longestPath := 0
	for k := range routePermissions {
		pathStart := strings.Index(k, "/")
		method := k[0:pathStart]
		path := k[pathStart:]

		// Store it so it's sortable but we can split it later
		paths = append(paths, fmt.Sprintf("%v|%v|%v", path, method, k))

		pathLen := len(path)
		if pathLen > longestPath {
			longestPath = pathLen
		}
	}
	sort.Strings(paths)

	// Print
	fmt.Println("Route Permissions:")
	fmtString := fmt.Sprintf("%%-7v%%-%vv -> %%v\n", longestPath)

	for _, path := range paths {
		// Make it more presentable
		bits := strings.Split(path, "|")
		path := bits[0]
		method := bits[1]
		query := bits[2]

		fmt.Printf(fmtString, method, path, routePermissions[query])
	}
}

func main() {
	// This was added for a profiler to be able to connect, otherwise uses no reasources really
	go func() {
		http.ListenAndServe(":1234", nil)
	}()
	// This is for prometheus
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		http.ListenAndServe(":2112", nil)
	}()

	rand.Seed(time.Now().UnixNano())

	log.Printf("API version: \"%v\"", services.ApiVersion)

	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("Something went wrong with API config. Error: %v\n", err)
	}

	// Show the config
	cfgJSON, err := json.MarshalIndent(cfg, "", utils.PrettyPrintIndentForJSON)
	if err != nil {
		log.Fatalf("Error trying to display config\n")
	}

	// Core count can't be 0!
	if cfg.CoresPerNode <= 0 {
		cfg.CoresPerNode = 6 // Reasonable, our laptops have 6...
	}

	if cfg.MaxQuantNodes <= 0 {
		cfg.MaxQuantNodes = 40
	}

	cfgStr := string(cfgJSON)
	log.Println(cfgStr)

	idGen := utils.IDGen{}
	signer := awsutil.RealURLSigner{}
	exporter := export.Exporter{}

	svcs := services.InitAPIServices(cfg, api.RealJWTReader{}, &idGen, &signer, &exporter)

	svcs.Log.Infof(cfgStr)

	notificationStack, err := notifications.MakeNotificationStack(svcs.Mongo, cfg.EnvironmentName, svcs.TimeStamper, svcs.Log, cfg.AdminEmails)

	if err != nil {
		err2 := fmt.Errorf("Failed to create notification stack: %v", err)
		svcs.Log.Errorf("%v", err2)
		log.Fatalf("%v\n", err2)
	}

	svcs.Notifications = notificationStack

	// Init all Mongo db interfaces here
	svcs.Users = pixlUser.MakeUserDetailsLookup(svcs.Mongo, cfg.EnvironmentName)
	svcs.Expressions = expressionDB.MakeExpressionDB(cfg.EnvironmentName, &svcs)

	jwtReader := api.RealJWTReader{Validator: initJWTValidator(cfg.Auth0Domain, svcs.FS, cfg, svcs.Log)}
	svcs.JWTReader = jwtReader

	router := endpoints.MakeRouter(svcs)

	// Setup middleware
	routePermissions := router.GetPermissions()
	printRoutePermissions(routePermissions)

	authware := authMiddleWareData{
		routePermissionsRequired: routePermissions,
		jwtValidator:             jwtReader.Validator,
		logger:                   svcs.Log,
	}
	logware := endpoints.LoggerMiddleware{
		APIServices:  &svcs,
		JwtValidator: jwtReader.Validator,
	}

	promware := endpoints.PrometheusMiddleware

	router.Router.Use(authware.Middleware, logware.Middleware, promware)

	// Now also log this to the world...
	svcs.Log.Infof("API version \"%v\" started...", services.ApiVersion)

	log.Fatal(
		http.ListenAndServe(":8080",
			handlers.CORS(
				handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
				handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}),
				handlers.AllowedOrigins([]string{"*"}))(router.Router)))
}
