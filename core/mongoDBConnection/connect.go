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
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/pixlise/core/v4/core/logger"
	"go.mongodb.org/mongo-driver/mongo"
)

// Helpers for connecting to Mongo DB
// NOTE: we support remote, local and "test" connections as per https://medium.com/@victor.neuret/mocking-the-official-mongo-golang-driver-5aad5b226a78

func ConnectToMongo(
	sess *session.Session, // Can be nil for local connection
	mongoSecret string, // empty for local connection
	iLog logger.ILogger,
	mongoDebug bool,
) (*mongo.Client, MongoConnectionInfo, error) {
	var mongoInfo MongoConnectionInfo
	var err error

	if len(mongoSecret) > 0 {
		// If the secret is NOT blank, assume we're connect to remote DB and get the details from secret cache
		mongoInfo, err = getMongoConnectionInfoFromSecretCache(sess, mongoSecret)
		if err != nil {
			return nil, mongoInfo, fmt.Errorf("Failed to read mongo secret \"%v\" info from secrets cache: %v", mongoSecret, err)
		}
	} else {
		// assume we're connecting to a local DB with no auth
		mongoUri, _ := os.LookupEnv("LOCAL_MONGO_URI")
		mongoInfo.Host = mongoUri
	}

	if strings.Contains(mongoInfo.Host, "docdb.") {
		cl, err := connectToRemoteDocDB(
			mongoInfo.Host,
			mongoInfo.Username,
			mongoInfo.Password,
			iLog,
			mongoDebug,
		)
		return cl, mongoInfo, err
	}

	mongoInfo.Host = MakeMongoURI(mongoInfo.Host, mongoInfo.Options)

	cl, err := connectAndCheckDB(mongoInfo, iLog, mongoDebug)

	return cl, mongoInfo, err
}

// server selection error:
// server selection timeout, current topology:
// { Type: Unknown, Servers: [{
//   Addr: pixlise-db.cluster-clcm0b2sosn0.us-east-1.docdb.amazonaws.com:27017,
//   Type: Unknown,
//   Last error:  connection(pixlise-db.cluster-clcm0b2sosn0.us-east-1.docdb.amazonaws.com:27017[-3])
//   incomplete read of message header: read tcp 192.168.157.130:60298->192.168.11.233:27017: i/o timeout:
//   connection(pixlise-db.cluster-clcm0b2sosn0.us-east-1.docdb.amazonaws.com:27017[-3])
//   incomplete read of message header: read tcp 192.168.157.130:60298->192.168.11.233:27017: i/o timeout }, ] }

func GetDatabaseName(dbName string, envName string) string {
	return dbName + "-" + envName
}

func MakeMongoURI(host, options string) string {
	uri := strings.Trim(host, "\t ")
	options = strings.Trim(options, "\t& ")
	if len(uri) <= 0 {
		uri = "localhost"
	}

	// Add mongodb prefix if needed
	if !strings.HasPrefix(uri, "mongodb://") {
		uri = "mongodb://" + uri
	}

	// Now make sure the 2 are joined with /? but only if there are options
	if len(options) > 0 {
		uri = strings.TrimRight(uri, "/?")
		options = strings.TrimLeft(options, "/?")

		uri = uri + "/?" + options
	}

	return uri
}
