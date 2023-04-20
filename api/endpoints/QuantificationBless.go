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

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/core/quantModel"

	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/utils"
)

func quantificationBless(params handlers.ApiHandlerParams) (interface{}, error) {
	// Get the ids invovled
	jobID := params.PathParams[idIdentifier]
	datasetID := params.PathParams[datasetIdentifier]

	// If quantification is NOT shared, we implicitly share it
	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(jobID)
	if !isSharedReq {
		err := quantModel.ShareQuantification(params.Svcs, params.UserInfo.UserID, datasetID, jobID)
		if err != nil {
			return nil, fmt.Errorf("Failed to \"bless\" quantification due to error while trying to share it: %v", err)
		}
	} else {
		jobID = strippedID
	}

	// Assume it's shared, check that it's actually there
	summaryPath := filepaths.GetSharedQuantPath(datasetID, filepaths.MakeQuantSummaryFileName(jobID))

	summary := quantModel.JobSummaryItem{}
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, summaryPath, &summary, false)
	if err != nil {
		if params.Svcs.FS.IsNotFoundError(err) {
			return nil, api.MakeNotFoundError(jobID)
		}
		return nil, err
	}

	// Read bless file if it exists, as we add version information to it
	blessFile, blessItem, blessFilePath, err := quantModel.GetBlessedQuantFile(params.Svcs, datasetID)

	// Generate a new version
	verNum := 1
	if blessItem != nil {
		verNum = blessItem.Version + 1
	}

	newItem := quantModel.BlessFileItem{
		Version:      verNum,
		BlessUnixSec: params.Svcs.TimeStamper.GetTimeNowSec(),
		UserID:       params.UserInfo.UserID,
		UserName:     params.UserInfo.Name,
		JobID:        jobID,
	}

	blessFile.History = append(blessFile.History, newItem)

	// Write bless file with new blessed version
	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, blessFilePath, blessFile)
}
