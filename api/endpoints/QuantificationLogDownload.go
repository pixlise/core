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
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"gitlab.com/pixlise/pixlise-go-api/api/filepaths"
	"gitlab.com/pixlise/pixlise-go-api/api/handlers"
	"gitlab.com/pixlise/pixlise-go-api/api/permission"
	"gitlab.com/pixlise/pixlise-go-api/core/quantModel"
	"gitlab.com/pixlise/pixlise-go-api/core/utils"
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
