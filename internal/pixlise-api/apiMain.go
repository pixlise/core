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

	"github.com/aws/aws-secretsmanager-caching-go/secretcache"

	"github.com/pixlise/core/core/notifications"

	"github.com/gorilla/handlers"

	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/api/endpoints"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/utils"

	_ "net/http/pprof"

	cmap "github.com/orcaman/concurrent-map"
	"github.com/pixlise/core/core/export"
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
	rand.Seed(time.Now().UnixNano())

	log.Printf("API version: \"%v\"", services.ApiVersion)

	cfg, err := config.Init()
	if err != nil {
		log.Fatalf("Something went wrong with API config. Check that your AWS region is set the same as the bucket. Error: %v\n", err)
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
		cfg.MaxQuantNodes = 20 // Was hard-coded to this anyway
	}

	cfgStr := string(cfgJSON)
	log.Println(cfgStr)

	idGen := utils.IDGen{}
	signer := awsutil.RealURLSigner{}
	exporter := export.Exporter{}

	var notes []notifications.UINotificationObj

	svcs := services.InitAPIServices(cfg, api.RealJWTReader{}, &idGen, &signer, &exporter, &notifications.NotificationStack{})

	svcs.Log.Infof(cfgStr)

	seccache, err := secretcache.New()
	mongo := notifications.MongoUtils{
		SecretsCache:     seccache,
		ConnectionSecret: cfg.MongoSecret,
		MongoUsername:    cfg.MongoUsername,
		MongoEndpoint:    cfg.MongoEndpoint,
		Log:              svcs.Log,
	}
	err = mongo.Connect()

	svcs.Log.Errorf("Failed to connect to Mongo: %v", err)
	// Reinitialised because of dependency on S3
	notificationStack := notifications.NotificationStack{
		Notifications: notes,
		FS:            svcs.FS,
		Track:         cmap.New(), //make(map[string]bool),
		Bucket:        cfg.UsersBucket,
		AdminEmails:   cfg.AdminEmails,
		Environment:   cfg.EnvironmentName,
		Logger:        svcs.Log,
		MongoUtils:    &mongo,
	}
	svcs.Notifications = &notificationStack
	jwtReader := api.RealJWTReader{Validator: initJWTValidator(cfg.Auth0Domain, svcs.FS, cfg)}
	svcs.JWTReader = jwtReader

	router := endpoints.MakeRouter(svcs)

	// Setup middleware
	routePermissions := router.GetPermissions()
	printRoutePermissions(routePermissions)

	authware := authMiddleWareData{routePermissionsRequired: routePermissions, jwtValidator: jwtReader.Validator}
	logware := endpoints.LoggerMiddleware{APIServices: &svcs, JwtValidator: jwtReader.Validator}

	router.Router.Use(authware.Middleware, logware.Middleware)

	// Now also log this to the world...
	svcs.Log.Infof("API version \"%v\" started...", services.ApiVersion)

	log.Fatal(
		http.ListenAndServe(":8080",
			handlers.CORS(
				handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
				handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}),
				handlers.AllowedOrigins([]string{"*"}))(router.Router)))
}
