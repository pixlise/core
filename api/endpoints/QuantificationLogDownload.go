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
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	"github.com/pixlise/core/core/quantModel"
	"github.com/pixlise/core/core/utils"
)

// TODO: need to write unit test for this! At the time it was deamed difficult because we talk to cloudwatch, but no longer the case

func quantificationLogFileStream(params handlers.ApiHandlerStreamParams) (*s3.GetObjectOutput, string, error) {
	// First, check if the user is allowed to access the given dataset
	datasetID := params.PathParams[datasetIdentifier]

	_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, datasetID)
	if err != nil {
		return nil, "", err
	}

	jobID := params.PathParams[idIdentifier]
	logName := params.PathParams[quantLogIdentifier]

	s3Path := filepaths.GetUserQuantPath(params.UserInfo.UserID, datasetID, path.Join(filepaths.MakeQuantLogDirName(jobID), logName))

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(jobID)
	if isSharedReq {
		// Shared quants don't copy logs, so download the summary.json to find the creator user id, then try to find the logs in their S3 path
		summary := quantModel.JobSummaryItem{}

		summaryPath := filepaths.GetSharedQuantPath(datasetID, filepaths.MakeQuantSummaryFileName(strippedID))
		err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, summaryPath, &summary, false)
		if err != nil {
			return nil, "", err
		}

		// Find the path of the original (non-shared) logs directory
		s3Path = filepaths.GetUserQuantPath(summary.Params.Creator.UserID, datasetID, path.Join(filepaths.MakeQuantLogDirName(strippedID), logName))
	}

	obj := &s3.GetObjectInput{
		Bucket: aws.String(params.Svcs.Config.UsersBucket),
		Key:    aws.String(s3Path),
	}

	result, err := params.Svcs.S3.GetObject(obj)

	return result, logName, err
}
