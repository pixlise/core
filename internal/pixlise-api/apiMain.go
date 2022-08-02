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

	"github.com/pixlise/core/core/notifications"

	"github.com/gorilla/handlers"

	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/api/endpoints"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/utils"

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
	rand.Seed(time.Now().UnixNano())

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

	// Reinitialised because of dependency on S3
	notificationStack := notifications.NotificationStack{
		Notifications: notes,
		FS:            svcs.FS,
		Track:         cmap.New(), //make(map[string]bool),
		Bucket:        cfg.UsersBucket,
		AdminEmails:   cfg.AdminEmails,
		Environment:   cfg.EnvironmentName,
		Logger:        svcs.Log,
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
	svcs.Log.Infof("API Started...")

	log.Fatal(
		http.ListenAndServe(":8080",
			handlers.CORS(
				handlers.AllowedHeaders([]string{"X-Requested-With", "Content-Type", "Authorization"}),
				handlers.AllowedMethods([]string{"GET", "POST", "PUT", "DELETE", "HEAD", "OPTIONS"}),
				handlers.AllowedOrigins([]string{"*"}))(router.Router)))
}
