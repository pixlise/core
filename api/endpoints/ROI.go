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
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/roiModel"
	"github.com/pixlise/core/v2/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// ROI - regions of interest

type roiHandler struct {
	svcs *services.APIServices
}

func registerROIHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "roi"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), roiList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), roiPost)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, "bulk"), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), roiBulkPost)

	//router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permReadDataAnalysis), roiGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), roiPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), roiDelete)

	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedROI), roiShare)
}

func roiList(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	rois := roiModel.ROILookup{}

	// Get user item summaries
	err := roiModel.GetROIs(params.Svcs, params.UserInfo.UserID, datasetID, &rois)
	if err != nil {
		return nil, err
	}

	// Get shared item summaries (into same map)
	err = roiModel.GetROIs(params.Svcs, pixlUser.ShareUserID, datasetID, &rois)
	if err != nil {
		return nil, err
	}

	// Return the combined set
	return &rois, nil
}

func createROIs(params handlers.ApiHandlerParams, rois []roiModel.ROIItem, overwriteName bool, skipDuplicate bool) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	s3Path := filepaths.GetROIPath(params.UserInfo.UserID, datasetID)
	allROIs, err := roiModel.ReadROIData(params.Svcs, s3Path)
	// Download the file
	if err != nil && !params.Svcs.FS.IsNotFoundError(err) {
		// Only return error if it's not about the file missing, because user may not have interacted with this dataset yet
		return nil, err
	}

	for _, roi := range rois {
		// Validate
		if !fileaccess.IsValidObjectName(roi.Name) {
			return nil, api.MakeBadRequestError(fmt.Errorf("Invalid ROI name: %v", roi.Name))
		}
		// Check that name is not duplicate. At first we didn't care, because we operate on IDs, but it came up that when exporting
		// using ROI names, files overwrote each other, so better to have unique names
		saveID := ""
		skipROI := false
		for roiID, roi := range allROIs {
			if roi.Name == roi.Name && roi.Creator.UserID == params.UserInfo.UserID {
				if overwriteName {
					saveID = roiID
				} else if skipDuplicate {
					skipROI = true
					break
				} else {
					return nil, api.MakeBadRequestError(fmt.Errorf("ROI name already used: %v", roi.Name))
				}
			}
		}

		if skipROI {
			continue
		}

		// Save it & upload
		if saveID == "" {
			saveID = params.Svcs.IDGen.GenObjectID()
		}
		_, exists := allROIs[saveID]
		if exists {
			return nil, fmt.Errorf("Failed to generate unique ID")
		}

		allROIs[saveID] = roiModel.ROISavedItem{
			ROIItem: &roi,
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:  false,
				Creator: params.UserInfo,
			},
		}
	}
	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, allROIs)
}

func roiBulkPost(params handlers.ApiHandlerParams) (interface{}, error) {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var roiOptions roiModel.ROIItemOptions
	err = json.Unmarshal(body, &roiOptions)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Bulk create ROIs with overwrite options
	return createROIs(params, roiOptions.ROIItems, roiOptions.Overwrite, roiOptions.SkipDuplicates)
}

func roiPost(params handlers.ApiHandlerParams) (interface{}, error) {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var roi roiModel.ROIItem
	err = json.Unmarshal(body, &roi)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	rois := []roiModel.ROIItem{roi}
	return createROIs(params, rois, false, false)
}

func roiPut(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	itemID := params.PathParams[idIdentifier]

	s3Path := filepaths.GetROIPath(params.UserInfo.UserID, datasetID)

	// Can't edit shared ones
	_, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		return nil, api.MakeBadRequestError(errors.New("Cannot edit shared ROIs"))
	}

	// Get all ROIs so we can check that it exists already
	allROIs, err := roiModel.ReadROIData(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	// Read in body of message, this will be what overwrites the existing ROI entry
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req roiModel.ROIItem
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Validate
	if !fileaccess.IsValidObjectName(req.Name) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid ROI name: \"%v\"", req.Name))
	}

	// Check that it exists
	_, exists := allROIs[itemID]
	if !exists {
		return nil, api.MakeStatusError(http.StatusNotFound, fmt.Errorf("ROI %v not found", itemID))
	}

	allROIs[itemID] = roiModel.ROISavedItem{
		ROIItem: &req,
		APIObjectItem: &pixlUser.APIObjectItem{
			Shared:  false,
			Creator: params.UserInfo,
		},
	}

	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, allROIs)
}

func roiDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	// If deleting a shared item, we need to strip the prefix from the ID and also ensure that only the creator can delete
	itemID := params.PathParams[idIdentifier]
	s3Path := filepaths.GetROIPath(params.UserInfo.UserID, datasetID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetROIPath(pixlUser.ShareUserID, datasetID)
		itemID = strippedID
	}

	// Using path params, work out path
	items, err := roiModel.ReadROIData(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	sharedItem, ok := items[itemID]
	if !ok {
		return nil, api.MakeNotFoundError(itemID)
	}

	if isSharedReq && sharedItem.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", itemID, params.UserInfo.UserID))
	}

	// Found it, delete & we're done
	delete(items, itemID)

	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, items)
}

func roiShare(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	idToFind := params.PathParams[idIdentifier]

	sharedIDs, err := roiModel.ShareROIs(params.Svcs, params.UserInfo.UserID, datasetID, []string{idToFind})
	if err != nil {
		return nil, err
	}

	// shared IDs should only contain one item!
	if len(sharedIDs) != 1 {
		return nil, errors.New("Failed to share ROI with ID: " + idToFind)
	}

	// Return just the one shared id
	return sharedIDs[0], nil
}
