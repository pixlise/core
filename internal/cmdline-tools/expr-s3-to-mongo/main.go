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
	"flag"
	"fmt"
	"log"
	"path"
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/awsutil"
	"github.com/pixlise/core/v3/core/expressions/expressions"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	mongoDBConnection "github.com/pixlise/core/v3/core/mongo"
	"github.com/pixlise/core/v3/core/pixlUser"
	"go.mongodb.org/mongo-driver/mongo"
)

// Read all expressions stored in S3 (all would be original "pixlang" pixlise expressions) and write them to MongoDB
// This is intended to be executed as a batch job once we release the new version that runs expressions out of Mongo

// Files to read:
// /UserContent/UserID/DataExpressions.json
// /UserContent/shared/DataExpressions.json

const expressionFile = "DataExpressions.json"

func main() {
	fmt.Println("=================================")
	fmt.Println("=  PIXLISE expression importer  =")
	fmt.Println("=  Don't forget to put PEM file =")
	fmt.Println("=       in local directory!     =")
	fmt.Println("=================================")

	ilog := logger.StdOutLogger{}
	ilog.SetLogLevel(logger.LogInfo)

	var s3Bucket = flag.String("bucket", "", "Name of bucket to import expressions from")
	var mongoConnString = flag.String("mongoDB", "", "Connection string to connect to mongo DB, of the form user:pass@host. If this is specified, mongoSecret is ignored")
	var mongoSecretString = flag.String("mongoSecret", "", "Mongo secret name, which allows retrieval of mongo connection string from AWS")
	var mongoDBName = flag.String("db", "", "Name of mongo DB to write expressions to")
	var mongoCollection = flag.String("collection", "", "Name of mongo collection to write expressions to")
	flag.Parse()

	var err error

	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("AWS GetSession failed: %v", err)
	}

	svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("AWS GetS3 failed: %v", err)
	}

	var mongoClient *mongo.Client

	if len(*mongoConnString) > 0 {
		user := ""
		pass := ""
		host := ""

		// Break it up
		idx := strings.Index(*mongoConnString, ":")
		if idx <= 0 {
			log.Fatalf("mongoDB connection string had no user name")
		}

		user = (*mongoConnString)[0:idx]

		remainder := (*mongoConnString)[(idx + 1):]

		idx = strings.Index(remainder, "@")
		if idx <= 0 {
			log.Fatalf("mongoDB connection string had no password@host")
		}

		pass = remainder[0:idx]
		host = remainder[(idx + 1):]

		mongoClient, err = mongoDBConnection.ConnectToRemoteMongoDB(
			host,
			user,
			pass,
			&ilog,
		)

		if err != nil {
			log.Fatalf("Failed to connect to remote mongo (%v): %v", *mongoConnString, err)
		}
	} else if len(*mongoSecretString) > 0 {
		mongoConnectionInfo, err := mongoDBConnection.GetMongoConnectionInfoFromSecretCache(sess, *mongoSecretString)
		if err != nil {
			log.Fatalf("Failed to get mongo connection info: %v", err)
		}

		mongoClient, err = mongoDBConnection.ConnectToRemoteMongoDB(
			mongoConnectionInfo.Host,
			mongoConnectionInfo.Username,
			mongoConnectionInfo.Password,
			&ilog,
		)

		if err != nil {
			log.Fatalf("Failed to connect to remote mongo: %v", err)
		}
	} else {
		mongoClient, err = mongoDBConnection.ConnectToLocalMongoDB(&ilog)
		if err != nil {
			log.Fatalf("%v", err)
		}

		if err != nil {
			log.Fatalf("Failed to connect to local mongo: %v", err)
		}
	}

	exprDatabase := mongoClient.Database(*mongoDBName)
	exprCollection := exprDatabase.Collection(*mongoCollection)

	remoteFS := fileaccess.MakeS3Access(svc)

	userContentPaths, err := remoteFS.ListObjects(*s3Bucket, filepaths.RootUserContent)
	if err != nil {
		log.Fatalf("Failed to list files in bucket: %v. Error: %v", *s3Bucket, err)
	}

	ilog.Infof("Listing returned %v files, processing only the expression files...", len(userContentPaths))

	exprFiles := []string{}

	for _, filePath := range userContentPaths {
		if strings.HasSuffix(filePath, expressionFile) {
			err = importExpressions(remoteFS, *s3Bucket, filePath, exprCollection, &ilog)

			if err != nil {
				ilog.Errorf("Failed to import: %v. Error: %v", filePath, err)
			} else {
				exprFiles = append(exprFiles, filePath)
			}
		}
	}

	if len(exprFiles) > 0 {
		ilog.Infof("Processed %v files. These can be deleted from S3:", len(exprFiles))
		for _, exprFile := range exprFiles {
			ilog.Infof("s3 rm s3://%v/%v", *s3Bucket, exprFile)
		}
	}
}

// Old-style expression struct we had stored in S3
type OldDataExpressionInput struct {
	Name       string   `json:"name"`
	Expression string   `json:"expression"`
	Comments   string   `json:"comments"`
	Tags       []string `json:"tags"`
}

type OldDataExpression struct {
	*OldDataExpressionInput
	*pixlUser.APIObjectItem
}

func importExpressions(remoteFS fileaccess.FileAccess, bucket string, s3Path string, exprCollection *mongo.Collection, l logger.ILogger) error {
	l.Infof("Reading expression file s3://%v/%v...", bucket, s3Path)

	itemLookup := map[string]OldDataExpression{}
	err := remoteFS.ReadJSON(bucket, s3Path, &itemLookup, true)
	if err != nil {
		return err
	}

	l.Infof(" Found %v expressions to insert...", len(itemLookup))
	count := 1
	for exprID, exprOld := range itemLookup {
		// Form a new expression struct
		expr := expressions.DataExpression{
			ID:               exprID,
			Name:             exprOld.Name,
			SourceCode:       exprOld.Expression,
			SourceLanguage:   "PIXLANG", // By definition, stuff stored in S3 was never Lua
			Comments:         exprOld.Comments,
			Tags:             exprOld.Tags,
			ModuleReferences: []expressions.ModuleReference{}, // Didn't support modules before
			Origin:           *exprOld.APIObjectItem,
			RecentExecStats:  nil, // Didn't support this before
		}
		// Prepare to write
		if expr.Tags == nil {
			expr.Tags = []string{}
		}
		if strings.HasPrefix(s3Path, path.Join(filepaths.RootUserContent, pixlUser.ShareUserID)) {
			if !expr.Origin.Shared {
				expr.Origin.Shared = true
				l.Errorf("Expression in shared dir: %v did not have shared flag set!", exprID)
			}
		}

		insertResult, err := exprCollection.InsertOne(context.TODO(), expr)
		if err != nil {
			return fmt.Errorf("Failed to insert expression: %v. Error: %v", exprID, err)
		}

		if insertResult.InsertedID != exprID {
			l.Errorf("Expected Mongo insert to return ID %v, got %v", exprID, insertResult.InsertedID)
		}

		l.Infof("  %v/%v: Inserted %v...", count, len(itemLookup), exprID)
		count++
	}

	return nil
}
