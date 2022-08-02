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
