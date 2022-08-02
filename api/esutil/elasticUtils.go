// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

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
