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
	"fmt"
	"io/ioutil"
	"path"
	"time"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/core/api"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Metrics

func registerMetricsHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "metrics"
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteMetrics), metricsPost)
}

func metricsPost(params handlers.ApiHandlerParams) (interface{}, error) {
	metricId := params.PathParams[idIdentifier]

	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Write to S3 where we store our analytics
	timeNow := params.Svcs.TimeStamper.GetTimeNowSec() // Read time this way so it can be mocked
	timestamp := time.Unix(timeNow, 0).Format("2006-01-02")

	s3Path := path.Join(filepaths.RootUserActivity, timestamp, fmt.Sprintf("metric-%v-%v-%v.json", metricId, params.UserInfo.UserID, timeNow))

	// Save it & upload
	return nil, params.Svcs.FS.WriteObject(params.Svcs.Config.UsersBucket, s3Path, body)
}
