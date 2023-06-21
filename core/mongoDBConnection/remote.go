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
	"io/ioutil"
	"strings"
	"time"

	"github.com/pixlise/core/v3/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func connectToRemoteMongoDB(
	MongoEndpoint string,
	MongoUsername string,
	MongoPassword string,
	iLog logger.ILogger,
) (*mongo.Client, error) {
	//ctx := context.Background()
	var err error
	var client *mongo.Client

	iLog.Infof("Connecting to remote mongo db: %v, user: %v", MongoEndpoint, MongoUsername)

	tlsConfig, err := getCustomTLSConfig("./rds-combined-ca-bundle.pem")
	if err != nil {
		return nil, fmt.Errorf("Failed getting TLS configuration: %v", err)
	}

	if strings.Contains(MongoEndpoint, "localhost") {
		tlsConfig.InsecureSkipVerify = true
	}

	const extraOptions = "" //"&retryWrites=false&tlsAllowInvalidHostnames=true" //"&replicaSet=rs0&readpreference=secondaryPreferred"
	connectionURI := fmt.Sprintf("mongodb://%s/%s", MongoEndpoint, extraOptions)

	cmdMonitor := makeMongoCommandMonitor(iLog)

	client, err = mongo.NewClient(
		options.Client().
			ApplyURI(connectionURI).
			SetMonitor(cmdMonitor).
			SetTLSConfig(tlsConfig).
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

	ctx, _ := context.WithTimeout(context.Background(), 3*time.Second)
	err = client.Connect(ctx)
	if err != nil {
		return nil, err
	}

	// Try to ping the DB to confirm connection
	var result bson.M
	err = client.Database("admin").RunCommand(context.TODO(), bson.D{{"ping", 1}}).Decode(&result)
	if err != nil {
		return nil, err
	}

	iLog.Infof("Successfully connected to remote mongo db!")

	//defer client.Disconnect(ctx)
	return client, nil
}

func getCustomTLSConfig(caFile string) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	certs, err := ioutil.ReadFile(caFile)

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

func makeMongoCommandMonitor(log logger.ILogger) *event.CommandMonitor {
	return &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			log.Debugf("Mongo request:\n%v", evt.Command)
		},
		Succeeded: func(_ context.Context, evt *event.CommandSucceededEvent) {
			log.Debugf("Mongo success:\n%v", evt.CommandFinishedEvent)
		},
		Failed: func(_ context.Context, evt *event.CommandFailedEvent) {
			log.Errorf("Mongo FAIL:\n%v", evt.Failure)
		},
	}
}
