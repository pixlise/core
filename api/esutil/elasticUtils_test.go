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
	"fmt"
	"net/http"
	"net/http/httptest"
	"time"

	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/core/logger"
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
