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
	"errors"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	"github.com/pixlise/core/v3/core/api"
)

func quantificationLastRunFileStream(params handlers.ApiHandlerStreamParams) (*s3.GetObjectOutput, string, error) {
	// First, check if the user is allowed to access the given dataset
	datasetID := params.PathParams[datasetIdentifier]

	// Check if user has rights for this dataset
	_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, params.Svcs.Config.ConfigBucket, datasetID)
	if err != nil {
		return nil, "", err
	}

	piquantCmd := params.PathParams[idIdentifier]
	fileRequested := params.PathParams[quantCmdOutputIdentifier]

	// Check these params
	// NOTE: for now we only support quant!
	if piquantCmd != "quant" || (fileRequested != "output" && fileRequested != "log") {
		return nil, "", api.MakeBadRequestError(errors.New("Invalid request"))
	}

	// Get the file name
	fileName := ""
	if fileRequested == "output" {
		fileName = filepaths.QuantLastOutputFileName + ".csv" // quant only supplies this
	} else {
		fileName = filepaths.QuantLastOutputLogName
	}

	// Get the path to stream
	s3Path := filepaths.GetUserLastPiquantOutputPath(params.UserInfo.UserID, datasetID, piquantCmd, fileName)

	obj := &s3.GetObjectInput{
		Bucket: aws.String(params.Svcs.Config.UsersBucket),
		Key:    aws.String(s3Path),
	}

	result, err := params.Svcs.S3.GetObject(obj)
	return result, fileName, err
}
