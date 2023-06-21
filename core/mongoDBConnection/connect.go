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
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pixlise/core/v3/core/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

// Helpers for connecting to Mongo DB
// NOTE: we support remote, local and "test" connections as per https://medium.com/@victor.neuret/mocking-the-official-mongo-golang-driver-5aad5b226a78

func Connect(
	sess *session.Session, // Can be nil for local connection
	mongoSecret string, // empty for local connection
	iLog logger.ILogger,
) (*mongo.Client, error) {
	// If the secret is blank, assume we're connecting to a local DB with no auth
	if len(mongoSecret) <= 0 {
		// Connect to local mongo
		return connectToLocalMongoDB(iLog)
	}

	// We're connecting to a remote one, first get the details from secret cache
	// Get a session for the bucket region
	mongoConnectionInfo, err := getMongoConnectionInfoFromSecretCache(sess, mongoSecret)
	if err != nil {
		return nil, fmt.Errorf("Failed to read mongo secret \"%v\" info from secrets cache: %v", mongoSecret, err)
	}

	return connectToRemoteMongoDB(
		mongoConnectionInfo.Host,
		mongoConnectionInfo.Username,
		mongoConnectionInfo.Password,
		iLog,
	)
}

func GetDatabaseName(dbName string, envName string) string {
	return dbName + "-" + envName
}
