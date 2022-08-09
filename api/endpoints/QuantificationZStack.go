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
	"encoding/json"
	"errors"
	"io/ioutil"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	"github.com/pixlise/core/core/api"
)

func quantificationCombineListSave(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	reqBody, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, api.MakeBadRequestError(errors.New("Failed to get request body"))
	}

	req := QuantCombineList{}
	err = json.Unmarshal(reqBody, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(errors.New("Request body invalid"))
	}

	// Make sure this dataset exists already by loading its summary file
	_, err = permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, datasetID)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Save this for user+dataset
	s3Path := filepaths.GetMultiQuantZStackPath(params.UserInfo.UserID, datasetID)
	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, req)
}

func quantificationCombineListLoad(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	list := QuantCombineList{RoiZStack: []QuantCombineItem{}}

	// Save this for user+dataset
	s3Path := filepaths.GetMultiQuantZStackPath(params.UserInfo.UserID, datasetID)
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &list, true)
	if err != nil {
		return nil, err
	}

	return list, nil
}
