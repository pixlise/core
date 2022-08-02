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
