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
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"os"
	"time"

	"github.com/pixlise/core/v2/api/config"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/awsutil"
	expressionDB "github.com/pixlise/core/v2/core/expressions/database"
	"github.com/pixlise/core/v2/core/expressions/expressions"
	"github.com/pixlise/core/v2/core/expressions/modules"
	"github.com/pixlise/core/v2/core/logger"
	mongoDBConnection "github.com/pixlise/core/v2/core/mongo"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/timestamper"
	"go.mongodb.org/mongo-driver/mongo"
)

func main() {
	envName := "mongo-test" // Be VERY careful, this should never be changed to an existing environments name
	// otherwise our test will DROP those collections!!!

	rand.Seed(time.Now().UnixNano())

	if len(os.Args) != 2 {
		fmt.Println("Arguments: mongo connection string")
		os.Exit(1)
	}

	var mongoStr = os.Args[1]

	fmt.Println("Running tests for mongo: " + mongoStr)

	cfg := config.APIConfig{
		DatasetsBucket:     "",
		ConfigBucket:       "",
		UsersBucket:        "",
		PiquantJobsBucket:  "",
		EnvironmentName:    envName,
		LogLevel:           logger.LogInfo,
		KubernetesLocation: "external",
		QuantExecutor:      "null",
		NodeCountOverride:  0,
		DataSourceSNSTopic: "arn:1:2:3:4:5",
	}

	idGen := services.MockIDGenerator{
		IDs: []string{"exp1", "exp2", "mod1", "mod2", "exp2sh"},
	}

	ts := &timestamper.MockTimeNowStamper{
		QueuedTimeStamps: []int64{1234500001, 1234500002, 1234500003, 1234500004, 1234500005, 1234500006, 1234500007},
	}

	l := logger.StdOutLogger{}
	l.SetLogLevel(cfg.LogLevel)

	svcs := services.APIServices{
		Config:       cfg,
		Log:          &l,
		AWSSessionCW: nil,
		S3:           nil,
		SNS:          &awsutil.MockSNS{},
		JWTReader:    nil,
		IDGen:        &idGen,
		TimeStamper:  ts,
		Signer:       nil,
		FS:           nil,
	}

	mongoClient, err := mongoDBConnection.ConnectToLocalMongoDB(&l)
	if err != nil {
		fmt.Errorf("%v", err)
		os.Exit(1)
	}
	svcs.Mongo = mongoClient

	// Be VERY careful, envName should never be changed to an existing environments name
	// otherwise our test will DROP those collections!!!
	db := expressionDB.MakeExpressionDB(envName, &svcs)
	svcs.Expressions = db
	svcs.Users = pixlUser.MakeUserDetailsLookup(mongoClient, envName)

	userDatabase := mongoClient.Database(mongoDBConnection.GetUserDatabaseName(envName))

	err = runTests(db, &svcs.Users, userDatabase.Collection("users"))
	if err != nil {
		svcs.Log.Errorf("%v", err)
		os.Exit(1)
	}
}

func dropWithSizeCheck(collection *mongo.Collection, maxSize int64) {
	ctx := context.TODO()
	count, err := collection.CountDocuments(ctx, nil, nil)
	if err != nil && count <= maxSize {
		collection.Drop(ctx)
	}
}

func verifyResult(err error, r interface{}, expected string, where string) {
	if err != nil {
		fmt.Printf("ERROR at: %v. Error: %v\n", where, err)
		os.Exit(1)
	}

	rJson, err := json.Marshal(r)

	if err != nil {
		fmt.Printf("FAILED to convert result to JSON at %v. Error: %v\n", where, err)
		os.Exit(1)
	}

	rStr := string(rJson)
	if rStr != expected {
		err := fmt.Errorf("FAIL at: %v.\nExpected:  %v\nRetrieved: %v\n", where, expected, rStr)
		fmt.Printf("%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("PASSED: %v\n", where)
}

func runTests(db *expressionDB.ExpressionDB, userDB *pixlUser.UserDetailsLookup, userColl *mongo.Collection) error {
	// Drop them, but make sure we don't drop something real, if they have significantly more
	// records than our test generates, we shouldn't drop them!
	dropWithSizeCheck(db.Expressions, 2)
	dropWithSizeCheck(db.ModuleVersions, 3)
	dropWithSizeCheck(db.Modules, 2)
	dropWithSizeCheck(userColl, 2)

	userInfoPeter := pixlUser.UserInfo{
		Name:        "Peter",
		UserID:      "999",
		Email:       "peter@pixlise.org",
		Permissions: map[string]bool{},
	}

	// Add an expression
	expr1, err := db.CreateExpression(
		expressions.DataExpressionInput{
			Name:           "expression 1",
			SourceCode:     "expression1()",
			SourceLanguage: "PIXLANG",
			Comments:       "expression one",
			Tags:           []string{"v1"},
		},
		userInfoPeter,
		false,
	)

	verifyResult(
		err,
		expr1,
		`{"id":"exp1","name":"expression 1","sourceCode":"expression1()","sourceLanguage":"PIXLANG","comments":"expression one","tags":["v1"],"origin":{"shared":false,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500001,"mod_unix_time_sec":1234500001}}`,
		"Add expression 1",
	)

	// And another
	expr2Input := expressions.DataExpressionInput{
		Name:           "expression 2",
		SourceCode:     "expression2()",
		SourceLanguage: "LUA",
		Comments:       "expression two",
		Tags:           []string{"v2"},
	}
	expr2, err := db.CreateExpression(
		expr2Input,
		userInfoPeter,
		false,
	)

	verifyResult(
		err,
		expr2,
		`{"id":"exp2","name":"expression 2","sourceCode":"expression2()","sourceLanguage":"LUA","comments":"expression two","tags":["v2"],"origin":{"shared":false,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500002,"mod_unix_time_sec":1234500002}}`,
		"Add expression 2",
	)

	// Create a module
	mod1, err := db.CreateModule(
		modules.DataModuleInput{
			Name:       "module 1",
			SourceCode: "module1()",
			Comments:   "module one",
			Tags:       []string{"v1"},
		},
		userInfoPeter,
	)

	verifyResult(
		err,
		mod1,
		`{"id":"mod1","name":"module 1","comments":"module one","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500003,"mod_unix_time_sec":1234500003},"version":{"sourceCode":"module1()","version":"0.0.1","tags":["v1"],"comments":"Initial version","mod_unix_time_sec":1234500003}}`,
		"Add module 1",
	)

	mod2, err := db.CreateModule(
		modules.DataModuleInput{
			Name:       "module 2",
			SourceCode: "module2()",
			Comments:   "module two",
			Tags:       []string{"v2"},
		},
		userInfoPeter,
	)

	verifyResult(
		err,
		mod2,
		`{"id":"mod2","name":"module 2","comments":"module two","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500004,"mod_unix_time_sec":1234500004},"version":{"sourceCode":"module2()","version":"0.0.1","tags":["v2"],"comments":"Initial version","mod_unix_time_sec":1234500004}}`,
		"Add module 2",
	)

	// Edit expression 1
	expr1A, err := db.UpdateExpression(
		"exp1",
		expressions.DataExpressionInput{
			Name:           "expression 1a",
			SourceCode:     "expression1a()",
			SourceLanguage: "LUA", // NOTE: converting to LUA
			Comments:       "expression one A",
			Tags:           []string{"v1", "v1.1"},
		},
		userInfoPeter,
		1234400000,
	)

	verifyResult(
		err,
		expr1A,
		`{"id":"exp1","name":"expression 1a","sourceCode":"expression1a()","sourceLanguage":"LUA","comments":"expression one A","tags":["v1","v1.1"],"origin":{"shared":false,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234400000,"mod_unix_time_sec":1234500005}}`,
		"Edit expression 1",
	)

	// Send stats for running expression 2
	err = db.StoreExpressionRecentRunStats(
		"exp2",
		expressions.DataExpressionExecStats{
			DataRequired:     []string{"elem-Ca", "elem-Fe", "data-ChiSq", "pseudo-Na", "spectrum"},
			RuntimeMS:        233,
			TimeStampUnixSec: 1234500006,
		},
	)
	if err != nil {
		return err
	}

	// Add version to module 1
	mod1a, err := db.AddModuleVersion(
		"mod1",
		modules.DataModuleVersionInput{
			SourceCode: "module1a()",
			Comments:   "module one A",
			Tags:       []string{"v1", "v1.1"},
		},
	)

	verifyResult(
		err,
		mod1a,
		`{"id":"mod1","name":"module 1","comments":"module one","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500003,"mod_unix_time_sec":1234500003},"version":{"sourceCode":"module1a()","version":"0.0.2","tags":["v1","v1.1"],"comments":"module one A","mod_unix_time_sec":1234500006}}`,
		"Add version to module 1",
	)

	// Share expression 2
	shareExpr2, err := db.CreateExpression(
		expr2Input,
		userInfoPeter,
		true,
	)

	verifyResult(
		err,
		shareExpr2,
		`{"id":"exp2sh","name":"expression 2","sourceCode":"expression2()","sourceLanguage":"LUA","comments":"expression two","tags":["v2"],"origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500007,"mod_unix_time_sec":1234500007}}`,
		"Share expression 2",
	)

	// List expressions
	exprList, err := db.ListExpressions("999", true, true)

	verifyResult(
		err,
		exprList,
		`{"exp1":{"id":"exp1","name":"expression 1a","sourceCode":"","sourceLanguage":"LUA","comments":"expression one A","tags":["v1","v1.1"],"origin":{"shared":false,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234400000,"mod_unix_time_sec":1234500005}},"exp2":{"id":"exp2","name":"expression 2","sourceCode":"","sourceLanguage":"LUA","comments":"expression two","tags":["v2"],"origin":{"shared":false,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500002,"mod_unix_time_sec":1234500002},"recentExecStats":{"dataRequired":["elem-Ca","elem-Fe","data-ChiSq","pseudo-Na","spectrum"],"runtimeMs":233,"mod_unix_time_sec":1234500006}},"exp2sh":{"id":"exp2sh","name":"expression 2","sourceCode":"","sourceLanguage":"LUA","comments":"expression two","tags":["v2"],"origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500007,"mod_unix_time_sec":1234500007}}}`,
		"List expressions",
	)

	// List modules
	modList, err := db.ListModules(true)

	verifyResult(
		err,
		modList,
		`{"mod1":{"id":"mod1","name":"module 1","comments":"module one","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500003,"mod_unix_time_sec":1234500003},"versions":[{"version":"0.0.1","tags":["v1"],"comments":"Initial version","mod_unix_time_sec":1234500003},{"version":"0.0.2","tags":["v1","v1.1"],"comments":"module one A","mod_unix_time_sec":1234500006}]},"mod2":{"id":"mod2","name":"module 2","comments":"module two","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500004,"mod_unix_time_sec":1234500004},"versions":[{"version":"0.0.1","tags":["v2"],"comments":"Initial version","mod_unix_time_sec":1234500004}]}}`,
		"List modules",
	)

	// Get version 0.0.1 of module 1
	mod1v1, err := db.GetModule("mod1", modules.SemanticVersion{0, 0, 1}, true)

	verifyResult(
		err,
		mod1v1,
		`{"id":"mod1","name":"module 1","comments":"module one","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500003,"mod_unix_time_sec":1234500003},"version":{"sourceCode":"module1()","version":"0.0.1","tags":["v1"],"comments":"Initial version","mod_unix_time_sec":1234500003}}`,
		"Get version 0.0.1 of module 1",
	)

	// Get version 0.0.2 of module 1
	mod1v2, err := db.GetModule("mod1", modules.SemanticVersion{0, 0, 2}, true)

	verifyResult(
		err,
		mod1v2,
		`{"id":"mod1","name":"module 1","comments":"module one","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500003,"mod_unix_time_sec":1234500003},"version":{"sourceCode":"module1a()","version":"0.0.2","tags":["v1","v1.1"],"comments":"module one A","mod_unix_time_sec":1234500006}}`,
		"Get version 0.0.2 of module 1",
	)

	// Get version 0.0.1 of module 2
	mod2v1, err := db.GetModule("mod2", modules.SemanticVersion{0, 0, 1}, true)

	verifyResult(
		err,
		mod2v1,
		`{"id":"mod2","name":"module 2","comments":"module two","origin":{"shared":true,"creator":{"name":"Peter","user_id":"999","email":"peter@pixlise.org"},"create_unix_time_sec":1234500004,"mod_unix_time_sec":1234500004},"version":{"sourceCode":"module2()","version":"0.0.1","tags":["v2"],"comments":"Initial version","mod_unix_time_sec":1234500004}}`,
		"Get version 0.0.1 of module 2",
	)
	// Change name in DB so we can verify that listing/get responds to it
	modifiedUserInfo := pixlUser.UserStruct{
		Userid:        "999",
		Notifications: pixlUser.Notifications{},
		Config: pixlUser.UserDetails{
			Name:           "Peter N",
			Email:          "peter_n@pixlise.org",
			Cell:           "",
			DataCollection: "1.1",
		},
	}

	err = userDB.WriteUser(modifiedUserInfo)
	if err != nil {
		return err
	}

	// Delete expression 2
	err = db.DeleteExpression("exp2")
	if err != nil {
		return err
	}

	// List expressions again
	exprList2, err := db.ListExpressions("999", true, true)

	verifyResult(
		err,
		exprList2,
		`{"exp1":{"id":"exp1","name":"expression 1a","sourceCode":"","sourceLanguage":"LUA","comments":"expression one A","tags":["v1","v1.1"],"origin":{"shared":false,"creator":{"name":"Peter N","user_id":"999","email":"peter_n@pixlise.org"},"create_unix_time_sec":1234400000,"mod_unix_time_sec":1234500005}},"exp2sh":{"id":"exp2sh","name":"expression 2","sourceCode":"","sourceLanguage":"LUA","comments":"expression two","tags":["v2"],"origin":{"shared":true,"creator":{"name":"Peter N","user_id":"999","email":"peter_n@pixlise.org"},"create_unix_time_sec":1234500007,"mod_unix_time_sec":1234500007}}}`,
		"List expressions again",
	)

	// List modules
	modList2, err := db.ListModules(true)

	verifyResult(
		err,
		modList2,
		`{"mod1":{"id":"mod1","name":"module 1","comments":"module one","origin":{"shared":true,"creator":{"name":"Peter N","user_id":"999","email":"peter_n@pixlise.org"},"create_unix_time_sec":1234500003,"mod_unix_time_sec":1234500003},"versions":[{"version":"0.0.1","tags":["v1"],"comments":"Initial version","mod_unix_time_sec":1234500003},{"version":"0.0.2","tags":["v1","v1.1"],"comments":"module one A","mod_unix_time_sec":1234500006}]},"mod2":{"id":"mod2","name":"module 2","comments":"module two","origin":{"shared":true,"creator":{"name":"Peter N","user_id":"999","email":"peter_n@pixlise.org"},"create_unix_time_sec":1234500004,"mod_unix_time_sec":1234500004},"versions":[{"version":"0.0.1","tags":["v2"],"comments":"Initial version","mod_unix_time_sec":1234500004}]}}`,
		"List modules again",
	)

	// Get expression 1
	expr2Get, err := db.GetExpression("exp1", true)

	verifyResult(
		err,
		expr2Get,
		`{"id":"exp1","name":"expression 1a","sourceCode":"expression1a()","sourceLanguage":"LUA","comments":"expression one A","tags":["v1","v1.1"],"origin":{"shared":false,"creator":{"name":"Peter N","user_id":"999","email":"peter_n@pixlise.org"},"create_unix_time_sec":1234400000,"mod_unix_time_sec":1234500005}}`,
		"Get expression 1",
	)

	return nil
}
