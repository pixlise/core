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

package mongoDBConnection

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/pixlise/core/v2/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// Helpers for connecting to Mongo DB
// NOTE: we support remote, local and "test" connections as per https://medium.com/@victor.neuret/mocking-the-official-mongo-golang-driver-5aad5b226a78

type MongoConnectionInfo struct {
	DbClusterIdentifier string `json:"dbClusterIdentifier"`
	Password            string `json:"password"`
	Engine              string `json:"engine"`
	Port                string `json:"port"`
	Host                string `json:"host"`
	Ssl                 string `json:"ssl"`
	Username            string `json:"username"`
}

func ConnectToRemoteMongoDB(
	MongoEndpoint string,
	MongoUsername string,
	MongoPassword string,
	log logger.ILogger,
) (*mongo.Client, error) {
	cmdMonitor := makeMongoCommandMonitor(log)

	//ctx := context.Background()
	var err error
	var client *mongo.Client

	log.Infof("Connecting to remote mongo db: %v, user: %v", MongoEndpoint, MongoUsername)

	tlsConfig, err := getCustomTLSConfig("./rds-combined-ca-bundle.pem")
	if err != nil {
		return nil, fmt.Errorf("Failed getting TLS configuration: %v", err)
	}

	if strings.Contains(MongoEndpoint, "localhost") {
		tlsConfig.InsecureSkipVerify = true
	}

	const extraOptions = "" //"&retryWrites=false&tlsAllowInvalidHostnames=true" //"&replicaSet=rs0&readpreference=secondaryPreferred"
	connectionURI := fmt.Sprintf("mongodb://%s/%s", MongoEndpoint, extraOptions)

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

	log.Infof("Successfully connected to remote mongo db!")

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

func GetMongoConnectionInfoFromSecretCache(session *session.Session, secretName string) (MongoConnectionInfo, error) {
	// Do some special init magic to get a secret manager with the right region set
	// This may not be needed in envs, but running locally it was needed!
	secMan := secretsmanager.New(session) //, aws.NewConfig().WithRegion("us-west-2"))

	var info MongoConnectionInfo
	seccache, err := secretcache.New(func(c *secretcache.Cache) { c.Client = secMan })
	if err != nil {
		return info, err
	}

	secretValue, err := seccache.GetSecretString(secretName)
	if err != nil {
		return info, err
	}

	// Secret cache seems to return these types... Unmarshall it
	json.Unmarshal([]byte(secretValue), &info)
	if err != nil {
		return info, fmt.Errorf("failed to parse secret: %v", secretName)
	}

	return info, nil
}

// Assumes local mongo running in docker as per this command:
// docker run -d  --name mongo-on-docker  -p 27888:27017 -e MONGO_INITDB_ROOT_USERNAME=mongoadmin -e MONGO_INITDB_ROOT_PASSWORD=secret mongo
func ConnectToLocalMongoDB(log logger.ILogger) (*mongo.Client, error) {
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

func GetDatabaseName(dbName string, envName string) string {
	return dbName + "-" + envName
}
