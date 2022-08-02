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
	"net/http"
	"path"
	"sync"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/quantModel"
	"github.com/pixlise/core/core/utils"
)

func quantificationDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	// If deleting a shared quant, we need to strip the prefix from the ID and also ensure that only the creator can delete
	itemID := params.PathParams[idIdentifier]
	datasetID := params.PathParams[datasetIdentifier]

	s3Path := filepaths.GetUserQuantPath(params.UserInfo.UserID, datasetID, "")

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetSharedQuantPath(datasetID, "")
		itemID = strippedID
	}

	// TODO: work out if it's in progress - if so, need to cancel the job

	// Download the summary to get the creator
	summaryPath := path.Join(s3Path, filepaths.MakeQuantSummaryFileName(itemID))
	summary := quantModel.JobSummaryItem{}
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, summaryPath, &summary, false)
	if err != nil {
		if params.Svcs.FS.IsNotFoundError(err) {
			return nil, api.MakeNotFoundError(itemID)
		}
		return nil, err
	}

	if isSharedReq && summary.Params.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", itemID, params.UserInfo.UserID))
	}

	// OK to delete summary file
	err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, summaryPath)
	if err != nil {
		return nil, err
	}

	// And the quant file
	quantPath := path.Join(s3Path, filepaths.MakeQuantDataFileName(itemID))
	err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, quantPath)
	if err != nil {
		return nil, err
	}

	// Delete the job status file from Jobs bucket too. This may not exist (files in there may have a life
	// span of days/weeks, so don't worry about the result).
	// NOTE: This is important, because this will trigger the job-updater lambda function, which will update
	// the summary file for this dataset, thereby removing the jobs status information from there.
	jobBucketStatusPath := filepaths.GetJobStatusPath(datasetID, itemID)
	params.Svcs.FS.DeleteObject(params.Svcs.Config.PiquantJobsBucket, jobBucketStatusPath)

	// And the CSV file (file may not exist, so we don't care about errors here)
	csvPath := path.Join(s3Path, filepaths.MakeQuantCSVFileName(itemID))
	params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, csvPath)

	// Delete log files (they're in a sub-dir, so need to list then delete them)
	// NOTE: They may not exist here (old quants didnt save logs here for eg), so we're
	//       not that worried if it fails, just writing to log
	logPathRoot := path.Join(s3Path, filepaths.MakeQuantLogDirName(itemID)) + "/"
	logPaths, err := params.Svcs.FS.ListObjects(params.Svcs.Config.UsersBucket, logPathRoot)

	if err == nil {
		itemCount := len(logPaths)

		var wg sync.WaitGroup
		wg.Add(itemCount)
		errs := make(chan error, itemCount)

		for _, logPath := range logPaths {
			go func(path string) {
				defer wg.Done()

				err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, path)
				if err != nil {
					errs <- err
				}
			}(logPath)
		}

		wg.Wait()
		close(errs)

		if len(errs) > 0 {
			params.Svcs.Log.Errorf("Failed to delete %v logs from s3://%v/%v", len(errs), logPathRoot)
		}
	} /* else {
		fmt.Printf("Failed to list log files in: " + logPathRoot)
	}*/

	return nil, nil
}
