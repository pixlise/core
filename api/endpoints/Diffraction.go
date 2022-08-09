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
	"encoding/json"
	"io/ioutil"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
)

type UserDiffractionPeak struct {
	PMC int32   `json:"pmc"`
	KeV float32 `json:"keV"`
}

type userDiffractionPeakFileContents struct {
	Peaks map[string]UserDiffractionPeak `json:"peaks"`
}

const statusId = "statusid"

func registerDiffractionHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefixStatus = "diffraction/status"
	const pathPrefixManual = "diffraction/manual"

	// "Accepting" a diffraction peak. This is stored in the "shared" area, all users who have access to this API are editing the
	// same file in S3. Race conditions possible if concurrently editing, but assumption is that'll be rare.
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefixStatus, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDiffractionPeaks), diffractionPeakStatusList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefixStatus, statusId, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermEditDiffractionPeaks), diffractionPeakStatusPost)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefixStatus, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermEditDiffractionPeaks), diffractionPeakStatusDelete)

	// Adding/deleting/getting manually entered diffraction peaks
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefixManual, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDiffractionPeaks), manualDiffractionPeakList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefixManual, datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermEditDiffractionPeaks), manualDiffractionPeakPost)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefixManual, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermEditDiffractionPeaks), manualDiffractionPeakDelete)

	// Streaming from S3
	// NOTE: This is downloaded through the dataset download endpoint, by specifying diffraction as the file name!
}

func readManualDiffractionFile(svcs *services.APIServices, s3Path string) (userDiffractionPeakFileContents, error) {
	manualPeaks := userDiffractionPeakFileContents{Peaks: map[string]UserDiffractionPeak{}}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &manualPeaks, true)

	if err != nil {
		manualPeaks = userDiffractionPeakFileContents{Peaks: map[string]UserDiffractionPeak{}}
	}

	return manualPeaks, err
}

func readDiffractionStatusFile(svcs *services.APIServices, s3Path string) (map[string]string, error) {
	statuses := map[string]string{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &statuses, true)
	return statuses, err
}

func diffractionPeakStatusList(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetSharedContentDatasetPath(params.PathParams[datasetIdentifier], filepaths.DiffractionPeakStatusFileName)

	ids, err := readDiffractionStatusFile(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	return ids, nil
}

func diffractionPeakStatusPost(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetSharedContentDatasetPath(params.PathParams[datasetIdentifier], filepaths.DiffractionPeakStatusFileName)
	peakId := params.PathParams[idIdentifier]
	status := params.PathParams[statusId]

	statuses, err := readDiffractionStatusFile(params.Svcs, s3Path)
	if err != nil {
		params.Svcs.Log.Errorf("Failed to load existing diffraction peak status file when editing, dataset: %v. Error: %v", params.PathParams[datasetIdentifier], err)
		statuses = map[string]string{}
	}

	statuses[peakId] = status

	return statuses, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, statuses)
}

func diffractionPeakStatusDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetSharedContentDatasetPath(params.PathParams[datasetIdentifier], filepaths.DiffractionPeakStatusFileName)
	idToDelete := params.PathParams[idIdentifier]

	statuses, err := readDiffractionStatusFile(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	_, ok := statuses[idToDelete]
	if !ok {
		return nil, api.MakeNotFoundError(idToDelete)
	}

	delete(statuses, idToDelete)

	return statuses, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, statuses)
}

func manualDiffractionPeakList(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetSharedContentDatasetPath(params.PathParams[datasetIdentifier], filepaths.DiffractionPeakManualFileName)

	contents, err := readManualDiffractionFile(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	return contents.Peaks, nil
}

func manualDiffractionPeakPost(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetSharedContentDatasetPath(params.PathParams[datasetIdentifier], filepaths.DiffractionPeakManualFileName)

	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req UserDiffractionPeak
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	existingPeaks, err := readManualDiffractionFile(params.Svcs, s3Path)
	if err != nil {
		params.Svcs.Log.Errorf("Failed to load existing diffraction file when adding, dataset: %v. Error: %v", params.PathParams[datasetIdentifier], err)
	}

	// Generate a new ID and add
	id := params.Svcs.IDGen.GenObjectID()
	existingPeaks.Peaks[id] = req

	return existingPeaks.Peaks, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, existingPeaks)
}

func manualDiffractionPeakDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetSharedContentDatasetPath(params.PathParams[datasetIdentifier], filepaths.DiffractionPeakManualFileName)
	idToDelete := params.PathParams[idIdentifier]

	existingPeaks, err := readManualDiffractionFile(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	// Find and remove
	_, ok := existingPeaks.Peaks[idToDelete]
	if !ok {
		return nil, api.MakeNotFoundError(idToDelete)
	}

	delete(existingPeaks.Peaks, idToDelete)

	return existingPeaks.Peaks, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, existingPeaks)
}
