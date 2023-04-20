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
	"sort"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/roiModel"
	"github.com/pixlise/core/v3/core/utils"
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

	// Read keys in alphabetical order, else we randomly fail unit test
	keys := []string{}
	for k := range rois {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Update ROI creator names/emails
	for _, k := range keys {
		roi := rois[k]
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(roi.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Infof("Failed to lookup user details for ID: %v, creator name in file: %v (ROI listing). Error: %v", roi.Creator.UserID, roi.Creator.Name, creatorErr)
		} else {
			roi.Creator = updatedCreator
		}

		if roi.Tags == nil {
			roi.Tags = []string{}
		}
	}

	// Return the combined set
	return &rois, nil
}

func createROIs(params handlers.ApiHandlerParams, rois []roiModel.ROIItem, overwriteName bool, skipDuplicate bool, deleteExistingMistROIs bool, shareROIs bool) error {
	datasetID := params.PathParams[datasetIdentifier]
	s3Path := filepaths.GetROIPath(params.UserInfo.UserID, datasetID)

	// If we're creating a shared ROI, change the S3 path
	if shareROIs {
		s3Path = filepaths.GetROIPath(pixlUser.ShareUserID, datasetID)
	}

	allROIs, err := roiModel.ReadROIData(params.Svcs, s3Path)
	// Download the file
	if err != nil && !params.Svcs.FS.IsNotFoundError(err) {
		// Only return error if it's not about the file missing, because user may not have interacted with this dataset yet
		return err
	}

	// Need to run this first so we don't delete newly created MIST ROIs
	if deleteExistingMistROIs {
		for roiID := range allROIs {
			if allROIs[roiID].MistROIItem.ClassificationTrail != "" {
				// All users have the ability to wipe out all Mist ROIs
				delete(allROIs, roiID)
			}
		}
	}

	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()
	for i := range rois {
		// Validate
		if !fileaccess.IsValidObjectName(rois[i].Name) {
			return api.MakeBadRequestError(fmt.Errorf("Invalid ROI name: %v", rois[i].Name))
		}
		// Check that name is not duplicate. At first we didn't care, because we operate on IDs, but it came up that when exporting
		// using ROI names, files overwrote each other, so better to have unique names
		skipROI := false
		for roiID, roi := range allROIs {
			// Mist ROIs can be overwritten by others, but regular ROIs can only be overwritten by their creators
			if roi.Name == rois[i].Name && (rois[i].MistROIItem.ClassificationTrail != "" || roi.Creator.UserID == params.UserInfo.UserID) {
				if overwriteName {
					// Delete any existing ROI keys if we're overwriting
					delete(allROIs, roiID)
				} else if skipDuplicate {
					skipROI = true
					break
				} else {
					return api.MakeBadRequestError(fmt.Errorf("ROI name already used: %v", roi.Name))
				}
			}
		}

		if skipROI {
			continue
		}

		// If saveID is empty, this means it's either new or we're not overwriting, so we generate a new one
		saveID := params.Svcs.IDGen.GenObjectID()
		_, exists := allROIs[saveID]
		if exists {
			return fmt.Errorf("failed to generate unique ID")
		}

		// Tags is a new field, so if it's not set, we set it to empty
		if rois[i].Tags == nil {
			rois[i].Tags = []string{}
		}

		allROIs[saveID] = roiModel.ROISavedItem{
			ROIItem: &rois[i],
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:              false,
				Creator:             params.UserInfo,
				CreatedUnixTimeSec:  timeNow,
				ModifiedUnixTimeSec: timeNow,
			},
		}
	}

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, allROIs)

	return err
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
	err = createROIs(params, roiOptions.ROIItems, roiOptions.Overwrite, roiOptions.SkipDuplicates, roiOptions.DeleteExistingMistROIs, roiOptions.ShareROIs)
	return nil, err
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
	err = createROIs(params, rois, false, false, false, false)
	return nil, err
}

func roiPut(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	itemID := params.PathParams[idIdentifier]

	s3Path := filepaths.GetROIPath(params.UserInfo.UserID, datasetID)

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

	var rois []roiModel.ROIReference

	if itemID == "bulk" {
		err = json.Unmarshal(body, &rois)
		if err != nil {
			return nil, api.MakeBadRequestError(err)
		}

	} else {
		var roi roiModel.ROIItem
		err = json.Unmarshal(body, &roi)
		if err != nil {
			return nil, api.MakeBadRequestError(err)
		}

		rois = append(rois, roiModel.ROIReference{ID: itemID, ROI: roi})
	}

	for i := range rois {
		// Can't edit shared ones
		_, isSharedReq := utils.StripSharedItemIDPrefix(rois[i].ID)
		if isSharedReq {
			return nil, api.MakeBadRequestError(errors.New("cannot edit shared rois"))
		}

		// Validate
		if !fileaccess.IsValidObjectName(rois[i].ROI.Name) {
			return nil, api.MakeBadRequestError(fmt.Errorf("invalid roi name: \"%v\"", rois[i].ROI.Name))
		}

		// Check that it exists
		existing, exists := allROIs[rois[i].ID]
		if !exists {
			return nil, api.MakeStatusError(http.StatusNotFound, fmt.Errorf("roi %v not found", rois[i].ID))
		}

		// Tags are a new field, so if it doesnt exist, add it in
		if rois[i].ROI.Tags == nil {
			rois[i].ROI.Tags = []string{}
		}

		allROIs[rois[i].ID] = roiModel.ROISavedItem{
			ROIItem: &rois[i].ROI,
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:              false,
				Creator:             params.UserInfo,
				CreatedUnixTimeSec:  existing.CreatedUnixTimeSec,
				ModifiedUnixTimeSec: params.Svcs.TimeStamper.GetTimeNowSec(),
			},
		}
	}

	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, allROIs)
}

func roiDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	itemID := params.PathParams[idIdentifier]

	userS3Path := filepaths.GetROIPath(params.UserInfo.UserID, datasetID)
	sharedS3Path := filepaths.GetROIPath(pixlUser.ShareUserID, datasetID)

	// If id is "bulk", then check body for a list of ROI IDs
	var roiIDs roiModel.ROIIDs
	if itemID == "bulk" {
		// Read in body
		body, err := ioutil.ReadAll(params.Request.Body)
		if err != nil {
			return nil, err
		}

		err = json.Unmarshal(body, &roiIDs)
		if err != nil {
			return nil, api.MakeBadRequestError(err)
		}
	} else {
		roiIDs.IDs = append(roiIDs.IDs, itemID)
	}

	// Run through ROIs and do an initial check of whether we need to update user or shared ROIs
	// This is mainly done to make test cases easier
	userIDFound := false
	sharedIDFound := false
	for i := range roiIDs.IDs {
		roiID := roiIDs.IDs[i]
		_, isSharedReq := utils.StripSharedItemIDPrefix(roiID)
		if isSharedReq {
			sharedIDFound = true
		} else {
			userIDFound = true
		}
	}

	var err error

	// Read in user ROIs and keep track of whether we deleted any
	userROIsChanged := false
	var userROIs roiModel.ROILookup
	if userIDFound {
		userROIs, err = roiModel.ReadROIData(params.Svcs, userS3Path)
		if err != nil {
			return nil, err
		}
	}

	// Read in shared ROIs and keep track of whether we deleted any
	sharedROIsChanged := false
	var sharedROIs roiModel.ROILookup
	if sharedIDFound {
		sharedROIs, err = roiModel.ReadROIData(params.Svcs, sharedS3Path)
		if err != nil {
			return nil, err
		}
	}

	for i := range roiIDs.IDs {
		roiID := roiIDs.IDs[i]

		strippedID, isSharedReq := utils.StripSharedItemIDPrefix(roiID)
		if isSharedReq {
			sharedROIsChanged = true
			roiID = strippedID

			sharedItem, ok := sharedROIs[roiID]
			if !ok {
				return nil, api.MakeNotFoundError(roiID)
			}

			// Only allow shared item to be deleted if it's a MIST ROI or same user requested it
			if sharedItem.MistROIItem.ClassificationTrail == "" && sharedItem.Creator.UserID != params.UserInfo.UserID {
				return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", roiID, params.UserInfo.UserID))
			}

			delete(sharedROIs, roiID)
		} else {
			userROIsChanged = true

			sharedItem, ok := userROIs[roiID]
			if !ok {
				return nil, api.MakeNotFoundError(roiID)
			}

			// Only allow user items to be deleted by same user
			if sharedItem.Creator.UserID != params.UserInfo.UserID {
				return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", roiID, params.UserInfo.UserID))
			}

			delete(userROIs, roiID)
		}
	}

	// Only write to the user ROI json if we changed it
	if userROIsChanged {
		err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, userS3Path, userROIs)
		if err != nil {
			return nil, err
		}
	}

	// Only write to the shared ROI json if we changed it
	if sharedROIsChanged {
		err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, sharedS3Path, sharedROIs)
	}

	return nil, err
}

func roiShare(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	idToFind := params.PathParams[idIdentifier]

	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var roiIDs roiModel.ROIIDs
	if idToFind == "bulk" {
		err = json.Unmarshal(body, &roiIDs)
		if err != nil {
			return nil, api.MakeBadRequestError(err)
		}
	} else {
		roiIDs.IDs = append(roiIDs.IDs, idToFind)
	}

	sharedIDs, err := roiModel.ShareROIs(params.Svcs, params.UserInfo.UserID, datasetID, roiIDs.IDs)
	if err != nil {
		return nil, err
	}

	// shared IDs should only contain one item if not bulk
	if idToFind != "bulk" && len(sharedIDs) != 1 {
		return nil, errors.New("Failed to share ROI with ID: " + idToFind)
	}

	// Return just the one shared id
	return sharedIDs[0], nil
}
