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
	"time"

	"github.com/pixlise/core/v3/core/logger"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Assumes local mongo running in docker as per this command:
// docker run -d  --name mongo-on-docker  -p 27888:27017 -e MONGO_INITDB_ROOT_USERNAME=mongoadmin -e MONGO_INITDB_ROOT_PASSWORD=secret mongo
func connectToLocalMongoDB(log logger.ILogger) (*mongo.Client, error) {
	cmdMonitor := makeMongoCommandMonitor(log)

	log.Infof("Connecting to local mongo db...")

	//ctx := context.Background()
	client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://mongoadmin:secret@localhost:27888/?authSource=admin").SetMonitor(cmdMonitor))
	if err != nil {
		return nil, fmt.Errorf("Failed to create new local mongo DB connection: %v", err)
	}

	ctx, _ := context.WithTimeout(context.Background(), 1*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	log.Infof("Successfully connected to local mongo db!")

	//defer client.Disconnect(ctx)
	return client, nil
}