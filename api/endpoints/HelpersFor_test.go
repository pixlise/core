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

package endpoints

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/pixlise/core/v4/api/services"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"
)

type MockExporter struct {
	downloadReturn []byte
	datasetID      string
	userID         string
	quantID        string
	fileIDs        []string
	fileNamePrefix string
}

func (m *MockExporter) MakeExportFilesZip(svcs *services.APIServices, fileNamePrefix string, userID string, datasetID string, quantID string, quantPath string, fileIDs []string, roiIDs []string) ([]byte, error) {
	m.fileNamePrefix = fileNamePrefix
	m.datasetID = datasetID
	m.userID = userID
	m.quantID = quantID
	m.fileIDs = fileIDs
	return m.downloadReturn, nil
}

// NOTE: The following came from https://semaphoreci.com/community/tutorials/building-and-testing-a-rest-api-in-go-with-gorilla-mux-and-postgresql
func executeRequest(req *http.Request, router *mux.Router) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}

func makeNotFoundMongoResponse() []primitive.D {
	return []primitive.D{
		mtest.CreateCursorResponse(
			1,
			"userdatabase-unit_test.notifications",
			mtest.FirstBatch,
		),
		mtest.CreateCursorResponse(
			0,
			"userdatabase-unit_test.notifications",
			mtest.NextBatch,
		),
	}
}

func checkResult(t *testing.T, resp *httptest.ResponseRecorder, expectedStatus int, expectedBody string) {
	if resp.Code != expectedStatus {
		t.Errorf("Bad resp code: %v", resp.Code)
	}

	gotRespBody := resp.Body.String()
	if gotRespBody != expectedBody {
		t.Errorf("Bad resp body:\n|%v|", gotRespBody)
		t.Errorf("vs expected body:\n|%v|", expectedBody)
	}
}

/*
func minifyJSON(jsonStr string) string {
	minifiedStr := &bytes.Buffer{}
	if err := json.Compact(minifiedStr, []byte(jsonStr)); err != nil {
		panic(err)
	}
	return minifiedStr.String()
}

func standardizeJSON(jsonStr string) string {
	standardizedStr := &bytes.Buffer{}
	if err := json.Indent(standardizedStr, []byte(jsonStr), "", "    "); err != nil {
		panic(err)
	}

	return standardizedStr.String()
}
*/
