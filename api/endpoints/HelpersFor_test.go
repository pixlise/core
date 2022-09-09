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

	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/notifications"

	"github.com/pixlise/core/api/esutil"
	"github.com/pixlise/core/core/pixlUser"

	"github.com/gorilla/mux"
	cmap "github.com/orcaman/concurrent-map"
	"github.com/pixlise/core/api/config"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/logger"
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

func mockElasticSearch() *esutil.Connection {
	testServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer testServer.Close()
	var ExpIndexObject = []string{}
	var ExpRespObject = []string{}

	d := esutil.DummyElasticClient{}
	foo, _ := d.DummyElasticSearchClient(testServer.URL, ExpRespObject, ExpIndexObject, ExpRespObject, nil)

	apiConfig := config.APIConfig{EnvironmentName: "Test"}

	connection, _ := esutil.Connect(foo, apiConfig)
	return &connection
}

func MakeMockSvcs(mockS3 *awsutil.MockS3Client, idGen services.IDGenerator, signer services.URLSigner, esconnection *esutil.Connection, logLevel *logger.LogLevel) services.APIServices {
	logging := logger.LogDebug
	if logLevel != nil {
		logging = *logLevel
	}
	if esconnection == nil {
		esconnection = mockElasticSearch()
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
	}

	fs := fileaccess.MakeS3Access(mockS3)

	var notes []notifications.UINotificationObj

	notificationStack := notifications.NotificationStack{
		Notifications: notes,
		FS:            fs,
		Bucket:        UsersBucketForUnitTest,
		Track:         cmap.New(), //make(map[string]bool),
	}

	return services.APIServices{
		Config:        cfg,
		Log:           &logger.NullLogger{},
		AWSSessionCW:  nil,
		S3:            mockS3,
		JWTReader:     MockJWTReader{},
		IDGen:         idGen,
		Signer:        signer,
		Notifications: &notificationStack,
		ES:            *esconnection,
		FS:            fs,
	}
}

// NOTE: The following came from https://semaphoreci.com/community/tutorials/building-and-testing-a-rest-api-in-go-with-gorilla-mux-and-postgresql
func executeRequest(req *http.Request, router *mux.Router) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	return rr
}
