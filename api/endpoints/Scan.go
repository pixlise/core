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
	"io"
	"path"

	"github.com/pixlise/core/v4/api/filepaths"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
)

const FormatIdentifier = "format"

// This allows the client to upload large zip files of data to import. This call simply saves the data in S3, it DOES NOT
// actually run the import process! For that, a ScanUploadReq needs to be sent with the same scanID and file name as passed
// to this call - otherwise it'll fail.
func PutScanData(params apiRouter.ApiHandlerGenericParams) error {
	destBucket := params.Svcs.Config.ManualUploadBucket

	scanId := fileaccess.MakeValidObjectName(params.PathParams[ScanIdentifier], false)

	if l := len(scanId); l <= 0 || l >= wsHelpers.IdFieldMaxLength {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid scanID: %v", scanId))
	}

	fileName := params.PathParams[FileNameIdentifier]
	if l := len(fileName); l <= 0 || l >= 100 {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid fileName: %v", fileName))
	}

	s3PathStart := path.Join(filepaths.DatasetUploadRoot, scanId)

	// NOTE: We overwrite any previous attempts without worry!
	/*
		// Check if this exists already...
		existingPaths, err := params.Svcs.FS.ListObjects(destBucket, s3PathStart)
		if err != nil {
			err = fmt.Errorf("Failed to list existing files for dataset ID: %v. Error: %v", scanId, err)
			params.Svcs.Log.Errorf("%v", err)
			return err
		}

		// If there are any existing paths, we stop here
		if len(existingPaths) > 0 {
			err = fmt.Errorf("Dataset ID already exists: %v", scanId)
			params.Svcs.Log.Errorf("%v", err)
			return errorwithstatus.MakeBadRequestError(err)
		}
	*/
	// Read in body
	zippedData, err := io.ReadAll(params.Request.Body)
	if err != nil {
		return err
	}

	savePath := path.Join(s3PathStart, fileName)

	err = params.Svcs.FS.WriteObject(destBucket, savePath, zippedData)
	if err != nil {
		return err
	}

	params.Svcs.Log.Debugf("  Wrote: s3://%v/%v", destBucket, savePath)
	return nil
}
