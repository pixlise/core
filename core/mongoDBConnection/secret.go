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
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
)

type MongoConnectionInfo struct {
	DbClusterIdentifier string `json:"dbClusterIdentifier"`
	Password            string `json:"password"`
	Engine              string `json:"engine"`
	Port                string `json:"port"`
	Host                string `json:"host"`
	Ssl                 string `json:"ssl"`
	Username            string `json:"username"`
}

func getMongoConnectionInfoFromSecretCache(session *session.Session, secretName string) (MongoConnectionInfo, error) {
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
