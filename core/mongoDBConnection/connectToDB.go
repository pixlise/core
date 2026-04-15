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
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/pixlise/core/v4/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Does the actual job of connecting:
// mongoInfo.Host: eg localhost or 192.168.1.1:27017,192.168.1.1:27018,192.168.1.1:27019
// mongoInfo.Options: eg "&replicaSet=rs0&readpreference=secondaryPreferred"
func connectAndCheckDB(
	mongoInfo MongoConnectionInfo,
	iLog logger.ILogger,
	mongoDebug bool,
) (*mongo.Client, error) {
	//ctx := context.Background()
	var err error
	var client *mongo.Client

	// We're only using SSL to connect to document DB right now
	useSSL := strings.Contains(mongoInfo.Host, "docdb.amazonaws.com")

	isLocalConnection := strings.Contains(mongoInfo.Host, "localhost")

	//iLog.Infof("Connecting to mongo db: %v", mongoInfo.Host)
	//iLog.Infof("mongoInfo: %+v", mongoInfo)

	if useSSL {
		//iLog.Infof("Using SSL")

		tlsConfig, err := getCustomTLSConfig("./global-bundle.pem")
		if err != nil {
			return nil, fmt.Errorf("Failed getting TLS configuration: %v", err)
		}

		if isLocalConnection {
			tlsConfig.InsecureSkipVerify = true
			//iLog.Infof("Using InsecureSkipVerify = true")
		}
	}

	cmdMonitor := makeMongoCommandMonitor(iLog, mongoDebug)

	opts := options.Client().ApplyURI(mongoInfo.Host).SetMonitor(cmdMonitor).SetRetryWrites(false)

	// To conform to how the document DB connection code was:
	/*
		client, err = mongo.Connect(
			context.TODO(),
			options.Client().
				ApplyURI(connectionURI).
				SetMonitor(cmdMonitor).
				//SetTLSConfig(tlsConfig).
				SetRetryWrites(false).
				SetDirect(true).
				SetAuth(
					options.Credential{
						Username:    MongoUsername,
						Password:    MongoPassword,
						PasswordSet: true,
						AuthSource:  "admin",
					}))
	*/

	// We weren't even applying the SSL settings it seems, but we had the direct flag on
	if useSSL || isLocalConnection {
		opts = opts.SetDirect(true)
	}

	if len(mongoInfo.Username) > 0 {
		iLog.Infof("Connect: Setting user name: %v, password length: %v", mongoInfo.Username, len(mongoInfo.Password))
		opts = opts.SetAuth(
			options.Credential{
				Username:    mongoInfo.Username,
				Password:    mongoInfo.Password,
				PasswordSet: true,
				AuthSource:  "admin",
			},
		)
	}

	//iLog.Infof("Connect: %+v", opts)
	client, err = mongo.Connect(context.TODO(), opts)

	if err != nil {
		return nil, fmt.Errorf("Failed to create new mongo DB connection: %v", err)
	}

	// Try to ping the DB to confirm connection
	if err := mongoTestPingDB(client); err != nil {
		return nil, err
	}

	iLog.Infof("Successfully connected to mongo db %v!", mongoInfo.Host)

	//defer client.Disconnect(ctx)
	return client, nil
}

func connectToRemoteDocDB(
	MongoEndpoint string,
	MongoUsername string,
	MongoPassword string,
	iLog logger.ILogger,
	mongoDebug bool,
) (*mongo.Client, error) {
	//ctx := context.Background()
	var err error
	var client *mongo.Client

	iLog.Infof("Connecting to document db: %v, user: %v", MongoEndpoint, MongoUsername)

	tlsConfig, err := getCustomTLSConfig("./global-bundle.pem")
	if err != nil {
		return nil, fmt.Errorf("Failed getting TLS configuration: %v", err)
	}

	if strings.Contains(MongoEndpoint, "localhost") {
		tlsConfig.InsecureSkipVerify = true
	}

	const extraOptions = "" //"&retryWrites=false&tlsAllowInvalidHostnames=true" //"&replicaSet=rs0&readpreference=secondaryPreferred"
	connectionURI := fmt.Sprintf("mongodb://%s/%s", MongoEndpoint, extraOptions)

	cmdMonitor := makeMongoCommandMonitor(iLog, mongoDebug)

	client, err = mongo.Connect(
		context.TODO(),
		options.Client().
			ApplyURI(connectionURI).
			SetMonitor(cmdMonitor).
			//SetTLSConfig(tlsConfig).
			SetRetryWrites(false).
			SetDirect(true).
			SetAuth(
				options.Credential{
					Username:    MongoUsername,
					Password:    MongoPassword,
					PasswordSet: true,
					AuthSource:  "admin",
				}))

	if err != nil {
		return nil, fmt.Errorf("Failed to create new mongo DB connection: %v", err)
	}

	// Try to ping the DB to confirm connection
	var result bson.M
	err = client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}

	iLog.Infof("Successfully connected to remote mongo db!")

	//defer client.Disconnect(ctx)
	return client, nil
}

func getCustomTLSConfig(caFile string) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	certs, err := os.ReadFile(caFile)

	if err != nil {
		return tlsConfig, err
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certs)

	if !ok {
		return tlsConfig, errors.New("Failed parsing pem file")
	}

	return tlsConfig, nil
}

func makeMongoCommandMonitor(log logger.ILogger, mongoDebug bool) *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			if mongoDebug {
				log.Debugf("Mongo request:\n%v", evt.Command)
			}
		},
		Succeeded: func(_ context.Context, evt *event.CommandSucceededEvent) {
			if mongoDebug {
				log.Debugf("Mongo success:\n%v", evt.CommandFinishedEvent)
			}
		},
		Failed: func(_ context.Context, evt *event.CommandFailedEvent) {
			log.Errorf("Mongo err:\n%v", evt.Failure)
		},
	}
}

func mongoTestPingDB(client *mongo.Client) error {
	// Try to ping the DB to confirm connection
	var result bson.M
	return client.Database("admin").RunCommand(context.TODO(), bson.D{{Key: "ping", Value: 1}}).Decode(&result)
}
