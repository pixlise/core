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
	"log"
	"net/http"
	"sort"
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/quantModel"
	"github.com/pixlise/core/v3/core/roiModel"
	"github.com/pixlise/core/v3/core/utils"
)

type Workspace struct {
	ViewState   wholeViewState `json:"viewState"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	*pixlUser.APIObjectItem
}

func (a Workspace) SetTimes(userID string, t int64) {
	if a.APIObjectItem == nil {
		log.Printf("Workspace: %v has no APIObjectItem\n", a.Name)
		a.APIObjectItem = &pixlUser.APIObjectItem{
			Creator: pixlUser.UserInfo{
				UserID: userID,
			},
		}
	}
	if a.CreatedUnixTimeSec == 0 {
		a.CreatedUnixTimeSec = t
	}
	if a.ModifiedUnixTimeSec == 0 {
		a.ModifiedUnixTimeSec = t
	}
}

type workspaceSummary struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	*pixlUser.APIObjectItem
}

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Saved View State CRUD calls

func savedViewStateList(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	userList, _ := getViewStateListing(params.Svcs, datasetID, params.UserInfo.UserID)
	// We don't check for errors because path may not exist yet and thats a valid scenario, just don't return any in that case
	sharedList, _ := getViewStateListing(params.Svcs, datasetID, pixlUser.ShareUserID)
	// We don't check for errors because path may not exist yet and thats a valid scenario, just don't return any in that case

	result := []workspaceSummary{}
	for _, item := range userList {
		result = append(result, item)
	}

	for _, item := range sharedList {
		item.ID = utils.SharedItemIDPrefix + item.ID
		result = append(result, item)
	}

	return result, nil
}

func getViewStateListing(svcs *services.APIServices, datasetID string, userID string) ([]workspaceSummary, error) {
	s3Path := filepaths.GetWorkspacePath(userID, datasetID, "")

	// Return each name
	filePaths, err := svcs.FS.ListObjects(svcs.Config.UsersBucket, s3Path)
	if err != nil {
		svcs.Log.Errorf("Failed to list view states in %v/%v: %v", svcs.Config.UsersBucket, s3Path, err)
		return []workspaceSummary{}, api.MakeStatusError(http.StatusInternalServerError, errors.New("Failed to list saved view states"))
	}

	listing := []workspaceSummary{}
	for _, filePath := range filePaths {
		// Before requiring load of the workspace, this was simply working off the file name:
		//fileName := path.Base(filePath)
		//fileExt := path.Ext(fileName)
		// Name aka workspace ID returned: fileName[0 : len(fileName)-len(fileExt)]

		// Get creator info
		state := Workspace{
			ViewState: defaultWholeViewState(),
		}

		err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, filePath, &state, false)
		if err != nil {
			return nil, api.MakeNotFoundError(filePath)
		}

		updatedCreator, creatorErr := svcs.Users.GetCurrentCreatorDetails(state.Creator.UserID)
		if creatorErr != nil {
			svcs.Log.Infof("Failed to lookup user details for ID: %v, creator name in file: %v (Workspace listing). Error: %v", state.Creator.UserID, state.Creator.Name, creatorErr)
		} else {
			state.Creator = updatedCreator
		}

		listing = append(listing, workspaceSummary{ID: state.Name, Name: state.Name, APIObjectItem: state.APIObjectItem})
	}

	return listing, nil
}

func savedViewStateGet(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	viewStateID := params.PathParams[idIdentifier]

	// Verify user has access to dataset (need to do this now that permissions are on a per-dataset basis)
	_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, params.Svcs.Config.ConfigBucket, datasetID)
	if err != nil {
		return nil, err
	}

	s3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, viewStateID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(viewStateID)
	if isSharedReq {
		s3Path = filepaths.GetWorkspacePath(pixlUser.ShareUserID, datasetID, strippedID)
		viewStateID = strippedID
	}

	state := Workspace{
		ViewState: defaultWholeViewState(),
	}

	err = params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &state, false)
	if err != nil {
		return nil, api.MakeNotFoundError(viewStateID)
	}

	applyQuantByROIFallback(&state.ViewState.Quantification)

	// Remove any view state items which are not shown by analysis layout
	filterUnusedWidgetStates(&state.ViewState)

	// Update creator name/email
	if state.APIObjectItem != nil {
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(state.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Infof("Failed to lookup user details for ID: %v, creator name in file: %v (Workspace GET). Error: %v", state.Creator.UserID, state.Creator.Name, creatorErr)
		} else {
			state.Creator = updatedCreator
		}
	}

	return &state, nil
}

func savedViewStatePut(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	viewStateID := params.PathParams[idIdentifier]
	forceFlag := params.PathParams["force"]

	_, isSharedReq := utils.StripSharedItemIDPrefix(viewStateID)
	if isSharedReq {
		return nil, api.MakeBadRequestError(errors.New("Cannot edit shared workspaces"))
	}

	s3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, viewStateID)

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	// Set up a default view state
	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()

	stateToSave := Workspace{
		ViewState: defaultWholeViewState(),
		APIObjectItem: &pixlUser.APIObjectItem{
			Shared:              false,
			Creator:             params.UserInfo,
			CreatedUnixTimeSec:  timeNow,
			ModifiedUnixTimeSec: timeNow,
		},
	}
	err = json.Unmarshal(body, &stateToSave)
	if err != nil {
		return nil, err
	}

	// At this point, if the force flag is not true, we check if it already exists and send back an error if it does
	if forceFlag != "true" {
		existingSaved := Workspace{
			ViewState: defaultWholeViewState(),
		}

		err = params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &existingSaved, false)
		if err == nil {
			// NO error, so this file exists. Here we return a 409 to UI so it knows this would be an overwrite
			return nil, api.StatusError{
				Code: http.StatusConflict,
				Err:  fmt.Errorf("%v already exists", viewStateID),
			}
		}
	}

	// Quant storage changed a while back, we have a fallback though
	applyQuantByROIFallback(&stateToSave.ViewState.Quantification)

	// Remove any view state items which are not shown by analysis layout
	filterUnusedWidgetStates(&stateToSave.ViewState)

	// If the name doesn't match, set it explicitly here
	if stateToSave.Name != viewStateID {
		stateToSave.Name = viewStateID
	}

	// Save it
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, stateToSave)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func savedViewStateDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	viewStateID := params.PathParams[idIdentifier]

	// Get its path, check that only the owner is deleting if shared
	s3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, viewStateID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(viewStateID)
	if isSharedReq {
		s3Path = filepaths.GetWorkspacePath(pixlUser.ShareUserID, datasetID, strippedID)
		viewStateID = strippedID
	}

	// Check that it exists
	state := Workspace{}
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &state, false)
	if err != nil {
		return nil, api.MakeNotFoundError(viewStateID)
	}

	// If it's not the owner deleting it, reject
	if isSharedReq && state.APIObjectItem != nil && state.APIObjectItem.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", viewStateID, params.UserInfo.UserID))
	}

	if !isSharedReq {
		// Check to ensure it doesn't exist in any collections first. Report the collection it's in as part of the error string if we find one
		listing, err := getViewStateCollectionListing(params.Svcs, datasetID, params.UserInfo.UserID)
		if err != nil {
			return nil, err
		}

		// Run through each collection, retrieve and complain if we find the view state in it
		for _, listingItem := range listing {
			collectionS3Path := filepaths.GetCollectionPath(params.UserInfo.UserID, datasetID, listingItem.Name)

			collectionContents := WorkspaceCollection{ViewStateIDs: []string{}}

			err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, collectionS3Path, &collectionContents, false)
			if err != nil {
				return nil, api.MakeStatusError(http.StatusNotFound, fmt.Errorf("Collection failed to load: \"%v\"", listingItem.Name))
			}

			if utils.StringInSlice(viewStateID, collectionContents.ViewStateIDs) {
				return nil, api.MakeStatusError(http.StatusConflict, fmt.Errorf("Workspace \"%v\" is in collection \"%v\". Please delete the workspace from all collections before before trying to delete it.", viewStateID, listingItem.Name))
			}
		}
	}

	err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, s3Path)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

// ByReferencedID To sort referenced IDs by ID field
type ByReferencedID []referencedIDItem

func (a ByReferencedID) Len() int           { return len(a) }
func (a ByReferencedID) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByReferencedID) Less(i, j int) bool { return strings.Compare(a[i].ID, a[j].ID) < 0 }

func savedViewStateGetReferencedIDs(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	viewStateID := params.PathParams[idIdentifier]
	s3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, viewStateID)

	// Verify user has access to dataset (need to do this now that permissions are on a per-dataset basis)
	_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, params.Svcs.Config.ConfigBucket, datasetID)
	if err != nil {
		return nil, err
	}

	// Read the file in
	state := Workspace{
		ViewState: defaultWholeViewState(),
	}

	err = params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &state, false)
	if err != nil {
		return nil, api.MakeNotFoundError(viewStateID)
	}

	// Check if there are any referenced IDs that are not already shared... If so, reject
	ids := state.ViewState.getReferencedIDs()

	// Sort the IDs so unit tests don't fail
	sort.Sort(ByReferencedID(ids.Expressions))
	sort.Sort(ByReferencedID(ids.ROIs))
	sort.Sort(ByReferencedID(ids.RGBMixes))

	// We've just got ids, now fill it out with more info from the shared objects. UI may not have this
	// available at this point, better that we supply it all
	if len(ids.ROIs) > 0 {
		rois := roiModel.ROILookup{}

		// Get user item summaries
		err := roiModel.GetROIs(params.Svcs, params.UserInfo.UserID, datasetID, &rois)
		if err != nil {
			params.Svcs.Log.Errorf("Failed to load user ROIs file, returned items may not have name/creator. Error: %v", err)
		}

		// Get shared item summaries (into same map)
		err = roiModel.GetROIs(params.Svcs, pixlUser.ShareUserID, datasetID, &rois)
		if err != nil {
			params.Svcs.Log.Errorf("Failed to load shared ROIs file, returned items may not have name/creator. Error: %v", err)
		}

		for idx, roi := range ids.ROIs {
			if item, ok := rois[roi.ID]; ok {
				ids.ROIs[idx].Name = item.Name
				ids.ROIs[idx].Creator = item.Creator
			}
		}
	}

	if len(ids.Expressions) > 0 {
		// Read all expressions
		exprs, err := params.Svcs.Expressions.ListExpressions(params.UserInfo.UserID, true, true)

		if err != nil {
			params.Svcs.Log.Errorf("Failed to load user/shared expressions file, returned items may not have name/creator. Error: %v", err)
		}

		for idx, expr := range ids.Expressions {
			if item, ok := exprs[expr.ID]; ok {
				ids.Expressions[idx].Name = item.Name
				ids.Expressions[idx].Creator = item.Origin.Creator
			}
		}
	}

	if len(ids.RGBMixes) > 0 {
		userS3Path := filepaths.GetRGBMixPath(params.UserInfo.UserID)

		items, err := readRGBMixData(params.Svcs, userS3Path)
		if err != nil {
			params.Svcs.Log.Errorf("Failed to load user RGB mixes, returned items may not have name/creator. Error: %v", err)
		}

		sharedS3Path := filepaths.GetRGBMixPath(pixlUser.ShareUserID)
		sharedItems, err := readRGBMixData(params.Svcs, sharedS3Path)
		if err != nil {
			params.Svcs.Log.Errorf("Failed to load shared RGB mixes, returned items may not have name/creator. Error: %v", err)
		}

		// Form 1 list
		for k, v := range sharedItems {
			items["shared-"+k] = v
		}

		for idx, rgb := range ids.RGBMixes {
			if item, ok := items[rgb.ID]; ok {
				ids.RGBMixes[idx].Name = item.Name
				ids.RGBMixes[idx].Creator = item.Creator
			}
		}
	}

	if len(ids.Quant.ID) > 0 {
		quantUserID := params.UserInfo.UserID
		if strings.HasPrefix(ids.Quant.ID, utils.SharedItemIDPrefix) {
			quantUserID = pixlUser.ShareUserID
		}
		quantSummary, err := quantModel.GetJobSummary(params.Svcs.FS, params.Svcs.Config.UsersBucket, quantUserID, datasetID, ids.Quant.ID)
		if err == nil {
			ids.Quant.Name = quantSummary.Params.Name
			ids.Quant.Creator = quantSummary.Params.Creator
		} else {
			params.Svcs.Log.Errorf("Failed to load quant summary file, returned items may not have name/creator. Error: %v", err)
		}
	}

	return ids, nil
}

const autoShareParamID = "auto-share"

func viewStateShare(params handlers.ApiHandlerParams) (interface{}, error) {
	viewStateID := params.PathParams[idIdentifier]
	datasetID := params.PathParams[datasetIdentifier]
	autoShare := params.PathParams[autoShareParamID] == "true"
	s3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, viewStateID)

	_, isSharedReq := utils.StripSharedItemIDPrefix(viewStateID)
	if isSharedReq {
		return nil, api.MakeBadRequestError(fmt.Errorf("Cannot share a shared ID"))
	}

	// Read the file in
	state := Workspace{
		ViewState: defaultWholeViewState(),
	}

	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &state, false)
	if err != nil {
		return nil, api.MakeNotFoundError(viewStateID)
	}

	// Check if there are any referenced IDs that are not already shared... If so, reject
	ids := state.ViewState.getReferencedIDs()
	if ids.NonSharedCount > 0 {
		if !autoShare {
			// We only share if the user has built this solely from shared ROIs, Quants and Expressions
			return nil, api.MakeBadRequestError(fmt.Errorf("Cannot share workspaces if they reference non-shared objects"))
		} else {
			// Share items if we are instructed to
			remappedIDs, err := autoShareNonSharedItems(params.Svcs, ids, datasetID, params.UserInfo.UserID)
			if err != nil {
				return nil, fmt.Errorf("Error while attempting to auto-share the non-shared objects in this workspace: %v", err)
			}

			// Now that we've shared, we need to set the right IDs in the view state we're about to share
			// otherwise what was the point?
			state.ViewState.replaceReferencedIDs(remappedIDs)
		}
	}

	applyQuantByROIFallback(&state.ViewState.Quantification)

	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()

	// Set shared owner info
	if state.APIObjectItem == nil {
		// Initially didn't have this field, so if anyone shares one where it didn't exist, they are set
		// as the creator. This works well because you can only see your own view states
		state.APIObjectItem = &pixlUser.APIObjectItem{
			Shared:              true,
			Creator:             params.UserInfo,
			CreatedUnixTimeSec:  timeNow,
			ModifiedUnixTimeSec: timeNow,
		}
	} else {
		state.Shared = true
		state.ModifiedUnixTimeSec = timeNow
	}

	// Write it to shared space
	// Save under the new name
	s3SavePath := filepaths.GetWorkspacePath(pixlUser.ShareUserID, datasetID, viewStateID)

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3SavePath, state)
	if err != nil {
		return nil, err
	}

	return viewStateID + " shared", nil
}

// Saved view state helpers
func autoShareNonSharedItems(svcs *services.APIServices, ids viewStateReferencedIDs, datasetID string, userID string) (map[string]string, error) {
	idRemap := map[string]string{}

	unsharedROIIDs := []string{}
	unsharedExpressionIDs := []string{}
	unsharedRGBMixIDs := []string{}

	for _, item := range ids.ROIs {
		if !strings.HasPrefix(item.ID, utils.SharedItemIDPrefix) {
			unsharedROIIDs = append(unsharedROIIDs, item.ID)
		}
	}

	for _, item := range ids.Expressions {
		if !strings.HasPrefix(item.ID, utils.SharedItemIDPrefix) {
			unsharedExpressionIDs = append(unsharedExpressionIDs, item.ID)
		}
	}

	for _, item := range ids.RGBMixes {
		if !strings.HasPrefix(item.ID, utils.SharedItemIDPrefix) {
			unsharedRGBMixIDs = append(unsharedRGBMixIDs, item.ID)
		}
	}

	newIDs, err := roiModel.ShareROIs(svcs, userID, datasetID, unsharedROIIDs)
	if err != nil {
		return idRemap, err
	}

	for idx, id := range newIDs {
		idRemap[unsharedROIIDs[idx]] = utils.SharedItemIDPrefix + id
	}

	newIDs, err = shareExpressions(svcs, userID, unsharedExpressionIDs)
	if err != nil {
		return idRemap, err
	}

	for idx, id := range newIDs {
		idRemap[unsharedExpressionIDs[idx]] = utils.SharedItemIDPrefix + id
	}

	newIDs, err = shareRGBMixes(svcs, userID, unsharedRGBMixIDs)
	if err != nil {
		return idRemap, err
	}

	for idx, id := range newIDs {
		idRemap[unsharedRGBMixIDs[idx]] = utils.SharedItemIDPrefix + id
	}

	if !strings.HasPrefix(ids.Quant.ID, utils.SharedItemIDPrefix) {
		err := quantModel.ShareQuantification(svcs, userID, datasetID, ids.Quant.ID)
		if err != nil {
			return idRemap, err
		}

		idRemap[ids.Quant.ID] = utils.SharedItemIDPrefix + ids.Quant.ID
	}

	return idRemap, nil
}
