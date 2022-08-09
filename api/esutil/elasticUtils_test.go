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
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"gitlab.com/pixlise/pixlise-go-api/api/config"
	"gitlab.com/pixlise/pixlise-go-api/core/logger"
)

func Example_testInsert() {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer testServer.Close()

	t := "2006-01-02T15:04:05-07:00"
	//tstr := fmt.Sprintf(t.Format("2006-01-02T15:04:05.0000000-07:00"))
	ti, err := time.Parse("2006-01-02T15:04:05-07:00", t)
	t2 := "2007-01-02T15:04:05-07:00"
	ti2, err := time.Parse("2006-01-02T15:04:05-07:00", t2)
	var ExpIndexObject = []string{
		`{"Instance":"Test","Time":"2006-01-02T15:04:05-07:00","Component":"Test Component","Message":"Test Message","Response":"","Version":"","Params":{"some param":"some param value"},"Environment":"Test","User":"5838239847"}`,
		`{"Instance":"Test","Time":"2007-01-02T15:04:05-07:00","Component":"Second Component","Message":"Test Message","Response":"","Version":"","Params":{"some param":"some param value"},"Environment":"Test","User":"5838239847"}`,
	}
	var ExpRespObject = []string{
		`{"_index":"metrics","_type":"trigger","_id":"B0tzT3wBosV6bFs8gJvY","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8468,"_primary_term":1}`,
		`{"_index":"metrics","_type":"trigger","_id":"B0tzT3wBosV6bFs8gJvY","_version":1,"result":"created","_shards":{"total":2,"successful":2,"failed":0},"_seq_no":8468,"_primary_term":1}`,
	}

	d := DummyElasticClient{}
	foo, err := d.DummyElasticSearchClient(testServer.URL, ExpRespObject, ExpIndexObject, ExpRespObject, nil)
	defer d.FinishTest()

	apiConfig := config.APIConfig{EnvironmentName: "Test"}
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	connection, err := Connect(foo, apiConfig)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	params := make(map[string]interface{})

	params["some param"] = "some param value"

	ilogger := logger.NullLogger{}

	o := LoggingObject{
		Instance:    "Test",
		Time:        ti,
		Component:   "Test Component",
		Message:     "Test Message",
		Params:      params,
		Environment: "Test",
		User:        "5838239847",
	}
	resp, err := InsertLogRecord(connection, o, ilogger)
	if err != nil {
		fmt.Printf("%v", err)
	}
	o = LoggingObject{
		Instance:    "Test",
		Time:        ti2,
		Component:   "Second Component",
		Message:     "Test Message",
		Params:      params,
		Environment: "Test",
		User:        "5838239847",
	}
	resp2, err := InsertLogRecord(connection, o, ilogger)
	if err != nil {
		fmt.Printf("%v", err)
	}
	fmt.Printf("Index: %v\n", resp.Index)
	fmt.Printf("Type: %v\n", resp.Type)
	fmt.Printf("Result: %v\n", resp.Result)
	fmt.Printf("Status: %v\n", resp.Status)
	fmt.Printf("Index: %v\n", resp2.Index)
	fmt.Printf("Type: %v\n", resp2.Type)
	fmt.Printf("Result: %v\n", resp2.Result)
	fmt.Printf("Status: %v\n", resp2.Status)
	// Output:
	// Index: metrics
	// Type: trigger
	// Result: created
	// Status: 0
	// Index: metrics
	// Type: trigger
	// Result: created
	// Status: 0

}

// Check to ensure that if there is an error thrown in the client we don't trip up and kill the server as the ES logs aren't mandatory
func Example_checkErrorHandling() {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer testServer.Close()

	t := "2006-01-02T15:04:05-07:00"
	twrong := "2007-01-02T15:04:05-07:00"
	//tstr := fmt.Sprintf(t.Format("2006-01-02T15:04:05.0000000-07:00"))
	ti, err := time.Parse("2006-01-02T15:04:05-07:00", t)
	var ExpIndexObject = []string{
		`{"Instance":"Test","Time":"` + twrong + `","Component":"Test Component","Message":"Test Message","Response":"","Version":"","Params":{"some param":"some param value"},"Environment":"Test","User":"5838239847"}`,
	}
	var ExpRespObject = []string{}

	d := DummyElasticClient{}
	foo, err := d.DummyElasticSearchClient(testServer.URL, ExpRespObject, ExpIndexObject, ExpRespObject, nil)
	defer d.FinishTest()

	apiConfig := config.APIConfig{EnvironmentName: "Test"}
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	connection, err := Connect(foo, apiConfig)
	if err != nil {
		fmt.Printf("%v\n", err)
	}
	params := make(map[string]interface{})

	params["some param"] = "some param value"

	ilogger := logger.NullLogger{}

	o := LoggingObject{
		Instance:    "Test",
		Time:        ti,
		Component:   "Test Component",
		Message:     "Test Message",
		Params:      params,
		Environment: "Test",
		User:        "5838239847",
	}
	resp, err := InsertLogRecord(connection, o, ilogger)
	if err != nil {
		fmt.Printf("%v", err)
	}

	fmt.Printf("%v", resp.Index)

	// Output:
	// incorrect input object expected: {"Instance":"Test","Time":"2007-01-02T15:04:05-07:00","Component":"Test Component","Message":"Test Message","Response":"","Version":"","Params":{"some param":"some param value"},"Environment":"Test","User":"5838239847"}, got: {"Instance":"Test","Time":"2006-01-02T15:04:05-07:00","Component":"Test Component","Message":"Test Message","Response":"","Version":"","Params":{"some param":"some param value"},"Environment":"Test","User":"5838239847"}
}
