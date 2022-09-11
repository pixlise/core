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

package quantModel

import (
	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
)

func ShareQuantification(svcs *services.APIServices, userID string, datasetID string, jobID string) error {
	// Gather "from" paths
	userBinPath := filepaths.GetUserQuantPath(userID, datasetID, filepaths.MakeQuantDataFileName(jobID))
	userSummaryPath := filepaths.GetUserQuantPath(userID, datasetID, filepaths.MakeQuantSummaryFileName(jobID))
	userCSVPath := filepaths.GetUserQuantPath(userID, datasetID, filepaths.MakeQuantCSVFileName(jobID))

	// Download summary file to make sure it exists/check creator matches
	summary := JobSummaryItem{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, userSummaryPath, &summary, false)
	if err != nil {
		if svcs.FS.IsNotFoundError(err) {
			return api.MakeNotFoundError(jobID)
		}
		return err
	}

	// Assume the "bin" file also exists...

	// Gather "to" paths
	sharedBinPath := filepaths.GetSharedQuantPath(datasetID, filepaths.MakeQuantDataFileName(jobID))
	sharedSummaryPath := filepaths.GetSharedQuantPath(datasetID, filepaths.MakeQuantSummaryFileName(jobID))
	sharedCSVPath := filepaths.GetSharedQuantPath(datasetID, filepaths.MakeQuantCSVFileName(jobID))

	// Copy the files to the destination - if any fails, return an overall error
	err = svcs.FS.CopyObject(svcs.Config.UsersBucket, userBinPath, svcs.Config.UsersBucket, sharedBinPath)
	if err != nil {
		return err
	}
	err = svcs.FS.CopyObject(svcs.Config.UsersBucket, userSummaryPath, svcs.Config.UsersBucket, sharedSummaryPath)
	if err != nil {
		return err
	}
	err = svcs.FS.CopyObject(svcs.Config.UsersBucket, userCSVPath, svcs.Config.UsersBucket, sharedCSVPath)
	if err != nil {
		return err
	}

	return nil
}
