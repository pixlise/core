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
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/olivere/elastic/v7"
)

type DummyHttpClient struct {
	responseMock    []string
	IndexedObjects  []string
	ResponseObjects []string
	time            string
}

type DummyElasticClient struct {
	client *DummyHttpClient
}

func (c *DummyHttpClient) Do(r *http.Request) (*http.Response, error) {
	var x string
	x, c.IndexedObjects = c.IndexedObjects[0], c.IndexedObjects[1:]
	buf := new(strings.Builder)
	_, err := io.Copy(buf, r.Body)
	var s = buf.String()
	if c.time != "" {
		s = strings.Replace(s, s[23:56], c.time, -1)
	}
	if err != nil {
		return nil, err
	}
	if x != s {
		return nil, errors.New(fmt.Sprintf("incorrect input object expected: %v, got: %v", x, s))
	}
	recorder := httptest.NewRecorder()
	var re string
	re, c.responseMock = c.responseMock[0], c.responseMock[1:]
	recorder.Write([]byte(re))

	recorder.Header().Set("Content-Type", "application/json")

	return recorder.Result(), nil
}

func MockHttpClient(responseMock []string, indexedObjects []string, responseObjects []string, t string) *DummyHttpClient {
	return &DummyHttpClient{responseMock, indexedObjects, responseObjects, t}
}

func (d *DummyElasticClient) DummyElasticSearchClient(endpoint string, responseMock []string, indexedObjects []string, responseObjects []string, t *string) (*elastic.Client, error) {
	timestr := ""
	if t != nil {
		timestr = *t
	}
	d.client = MockHttpClient(responseMock, indexedObjects, responseObjects, timestr)

	client, err := elastic.NewClient(
		elastic.SetURL(endpoint),
		elastic.SetSniff(false),
		elastic.SetHealthcheck(false),
		elastic.SetHttpClient(d.client))

	return client, err
}

func (d *DummyElasticClient) FinishTest() error {
	if d.client.IndexedObjects != nil && len(d.client.IndexedObjects) > 0 {
		fmt.Println("indexed objects not empty, expecting more data")
		return errors.New("indexed objects not empty, expecting more data")
	}

	if d.client.responseMock != nil && len(d.client.responseMock) > 0 {
		fmt.Println("response objects not empty expecting more data")
		return errors.New("response objects not empty expecting more data")
	}
	return nil
}
