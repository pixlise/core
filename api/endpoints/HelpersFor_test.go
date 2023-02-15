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
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/notifications"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/integration/mtest"

	"github.com/pixlise/core/v2/core/pixlUser"

	"github.com/gorilla/mux"
	"github.com/pixlise/core/v2/api/config"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/logger"
)

const DatasetsBucketForUnitTest = "datasets-bucket"
const ConfigBucketForUnitTest = "config-bucket"
const UsersBucketForUnitTest = "users-bucket"
const jobBucketForUnitTest = "job-bucket"

type MockJWTReader struct {
	InfoToReturn *pixlUser.UserInfo
}

func (m MockJWTReader) GetUserInfo(*http.Request) (pixlUser.UserInfo, error) {
	if m.InfoToReturn != nil {
		return *m.InfoToReturn, nil
	}
	//This user id is real don't change it....
	return pixlUser.UserInfo{
		Name:        "Niko Bellic",
		UserID:      "600f2a0806b6c70071d3d174",
		Email:       "niko@spicule.co.uk",
		Permissions: map[string]bool{},
	}, nil
}

type MockIDGenerator struct {
	ids []string
}

func (m *MockIDGenerator) GenObjectID() string {
	if len(m.ids) > 0 {
		id := m.ids[0]
		m.ids = m.ids[1:]
		return id
	}
	return "NO_ID_DEFINED"
}

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

func MakeMockSvcs(mockS3 *awsutil.MockS3Client, idGen services.IDGenerator, signer services.URLSigner, logLevel *logger.LogLevel) services.APIServices {
	logging := logger.LogDebug
	if logLevel != nil {
		logging = *logLevel
	}

	cfg := config.APIConfig{
		DatasetsBucket:      DatasetsBucketForUnitTest,
		ConfigBucket:        ConfigBucketForUnitTest,
		UsersBucket:         UsersBucketForUnitTest,
		PiquantJobsBucket:   jobBucketForUnitTest,
		AWSBucketRegion:     "us-east-1",
		AWSCloudwatchRegion: "us-east-1",
		EnvironmentName:     "unit-test",
		LogLevel:            logging,
		KubernetesLocation:  "external",
		QuantExecutor:       "null",
		NodeCountOverride:   0,
		DataSourceSNSTopic:  "arn:1:2:3:4:5",
	}

	fs := fileaccess.MakeS3Access(mockS3)

	return services.APIServices{
		Config:       cfg,
		Log:          &logger.NullLogger{},
		AWSSessionCW: nil,
		S3:           mockS3,
		SNS:          &awsutil.MockSNS{},
		JWTReader:    MockJWTReader{},
		IDGen:        idGen,
		Signer:       signer,
		FS:           fs,
	}
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

func runOneURLCallTest(t *testing.T, method string, url string, requestPayload io.Reader, expectedStatusCode int, expectedResult string, mongoMockedResponses []primitive.D, endCallback func()) {
	mt := mtest.New(t, mtest.NewOptions().ClientType(mtest.Mock))
	defer mt.Close()

	mt.Run("success", func(mt *mtest.T) {
		mt.AddMockResponses(mongoMockedResponses...)

		var mockS3 awsutil.MockS3Client
		defer mockS3.FinishTest()

		svcs := MakeMockSvcs(&mockS3, nil, nil, nil)
		setTestAuth0Config(&svcs)
		notifications, err := notifications.MakeNotificationStack(mt.Client, "unit_test", nil, &logger.StdOutLoggerForTest{}, []string{})
		if err != nil {
			t.Error(err)
		}

		svcs.Notifications = notifications

		svcs.Users = pixlUser.MakeUserDetailsLookup(mt.Client, "unit_test")

		apiRouter := MakeRouter(svcs)

		req, _ := http.NewRequest(method, url, requestPayload)
		resp := executeRequest(req, apiRouter.Router)

		checkResult(t, resp, expectedStatusCode, expectedResult)

		// When we're done, call end callback in case caller has anything else to verify
		if endCallback != nil {
			endCallback()
		}
	})
}

func checkResult(t *testing.T, resp *httptest.ResponseRecorder, expectedStatus int, expectedBody string) {
	if resp.Code != expectedStatus {
		t.Errorf("Bad resp code: %v", resp.Code)
	}

	gotRespBody := resp.Body.String()
	if gotRespBody != expectedBody {
		t.Errorf("Bad resp body:\n%v", gotRespBody)
		t.Errorf("vs expected body:\n%v", expectedBody)
	}
}

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
