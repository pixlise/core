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

// Lowest-level code to connect to Mongo DB (locally in Docker and remotely) and get consistant collection names.
package mongoDBConnection

import (
	"context"
	"fmt"
	"os"

	"github.com/pixlise/core/v4/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Assumes local mongo running in docker as per this command:
// docker run -d  --name mongo-on-docker  -p 27888:27017 -e MONGO_INITDB_ROOT_USERNAME=mongoadmin -e MONGO_INITDB_ROOT_PASSWORD=secret mongo
func connectToLocalMongoDB(log logger.ILogger) (*mongo.Client, error) {
	cmdMonitor := makeMongoCommandMonitor(log)

	log.Infof("Connecting to local mongo db...")
	mongoUri, set := os.LookupEnv("LOCAL_MONGO_URI")
	if !set {
		mongoUri = "mongodb://localhost"
	}
	//ctx := context.Background()
	client, err := mongo.Connect(context.TODO(), options.Client().ApplyURI(mongoUri).SetMonitor(cmdMonitor).SetDirect(true))
	if err != nil {
		return nil, fmt.Errorf("Failed to create new local mongo DB connection: %v", err)
	}

	// Try to ping the DB to confirm connection
	var result bson.M
	err = client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}

	log.Infof("Successfully connected to local mongo db!")

	//defer client.Disconnect(ctx)
	return client, nil
}
