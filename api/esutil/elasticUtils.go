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

package esutil

import (
	"context"
	"encoding/json"
	"time"

	"github.com/pixlise/core/core/logger"

	"github.com/olivere/elastic/v7"
	"github.com/pixlise/core/api/config"
)

// LoggingObject - Object used for adding metric to ES
type LoggingObject struct {
	Instance    string
	Time        time.Time
	Component   string
	Message     string
	Response    string
	Version     string
	Params      map[string]interface{}
	Environment string
	User        string
}

// Connection - default connection object
type Connection struct {
	client      *elastic.Client
	environment string
}

func FullFatClient(cfg config.APIConfig, log logger.ILogger) *elastic.Client {
	var (
		url  = cfg.ElasticURL
		user = cfg.ElasticUser
		pass = cfg.ElasticPassword
	)

	// Create an Elasticsearch client
	client, err := elastic.NewClient(
		//elastic.SetURL(url),
		elastic.SetURL(url),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetBasicAuth(user, pass),
	)
	if err != nil {
		log.Errorf("Failed to connect to elasticsearch: %v", err.Error())
	}

	return client
}

// Connect - Connect to the ES instance
func Connect(client *elastic.Client, cfg config.APIConfig) (Connection, error) {
	conn := Connection{
		client:      client,
		environment: cfg.EnvironmentName,
	}
	return conn, nil
}

// InsertLogRecord - Insert a quick log message
func InsertLogRecord(es Connection, o LoggingObject, log logger.ILogger) (*elastic.IndexResponse, error) {
	b, err := json.Marshal(o)

	ctx := context.Background()
	var res *elastic.IndexResponse

	// Don't do this when testing with the "local" env, that's meant to run on a laptop, we don't want to contaminate
	// and data stored about user activity
	if es.environment == "local" {
		return nil, nil
	}

	o.Instance = es.environment
	if es.client != nil {
		res, err = es.client.Index().
			Index("metrics").
			Type("trigger").
			BodyString(string(b)).
			Do(ctx)
		if err != nil {
			// Handle error
			// Took out because we already log at the caller of this function
			//log.Errorf("Elasticsearch error: %v", err.Error())
			res = &elastic.IndexResponse{
				Index:         "",
				Type:          "",
				Id:            "",
				Version:       0,
				Result:        "",
				Shards:        nil,
				SeqNo:         0,
				PrimaryTerm:   0,
				Status:        0,
				ForcedRefresh: false,
			}
			return res, err
		}
	}
	return res, nil
}
