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
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Exporting stuff, UI requesting things that can be saved on client side

type exportFilesParams struct {
	FileName string   `json:"fileName"`
	QuantID  string   `json:"quantificationId"`
	FileIDs  []string `json:"fileIds"`
	ROIIDs   []string `json:"roiIds"`
}

func registerExportHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "export"

	router.AddGenericHandler(handlers.MakeEndpointPath(pathPrefix+"/files", datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermExportMap), exportFilesPost)
}

func exportFilesPost(params handlers.ApiHandlerGenericParams) error {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return api.MakeBadRequestError(err)
	}

	var req exportFilesParams
	err = json.Unmarshal(body, &req)
	if err != nil {
		return api.MakeBadRequestError(err)
	}

	if len(req.FileIDs) <= 0 {
		return api.MakeBadRequestError(fmt.Errorf("No File IDs specified, nothing to export"))
	}

	if !strings.HasSuffix(req.FileName, ".zip") {
		return api.MakeBadRequestError(fmt.Errorf("File name must end in .zip"))
	}

	// We have all parameters:
	// Dataset ID
	// Quant ID <-- Can be blank if no quant loaded!
	// File IDs

	// We need to export a ZIP file containing what is identified in File IDs

	datasetID := params.PathParams[datasetIdentifier]

	// Get the quantification file - if it's a shared file, quantUserID should be empty
	quantPath := filepaths.GetUserQuantPath(params.UserInfo.UserID, datasetID, "")
	quantID := req.QuantID

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(req.QuantID)
	if isSharedReq {
		quantPath = filepaths.GetSharedQuantPath(datasetID, "")
		quantID = strippedID
	}

	// File name is likely to (well, mandated really) to end in .zip, but we don't want this in all our exports!
	filePrefix := req.FileName
	if strings.HasSuffix(filePrefix, ".zip") {
		filePrefix = filePrefix[0 : len(filePrefix)-4]
	}

	zipData, err := params.Svcs.Exporter.MakeExportFilesZip(
		params.Svcs,
		filePrefix,
		params.UserInfo.UserID,
		datasetID,
		quantID,
		quantPath,
		req.FileIDs,
		req.ROIIDs,
	)

	if err != nil {
		return api.MakeStatusError(http.StatusNotFound, err)
	}

	// We write our responses as octet streams, and include the file name...
	params.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", req.FileName))
	params.Writer.Header().Set("Content-Type", "application/octet-stream")
	params.Writer.Header().Set("Cache-Control", "no-store")
	params.Writer.Header().Set("Access-Control-Expose-Headers", "Content-Disposition")

	params.Writer.Header().Set("Content-Length", fmt.Sprintf("%v", len(zipData)))

	_, copyErr := io.Copy(params.Writer, bytes.NewReader(zipData))
	if copyErr != nil {
		fmt.Printf("Failed to write zip contents of %v to response", req.FileName)
	}

	return nil
}
