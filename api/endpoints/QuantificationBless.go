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

package endpoints

import (
	"fmt"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/core/quantModel"

	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/utils"
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
