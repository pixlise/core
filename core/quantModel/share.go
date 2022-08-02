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

package quantModel

import (
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
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
