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
	"strings"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
	dataExpression "github.com/pixlise/core/v2/core/expression"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/quantModel"
	"github.com/pixlise/core/v2/core/roiModel"
	"github.com/pixlise/core/v2/core/utils"
)

type workspace struct {
	ViewState   wholeViewState `json:"viewState"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	*pixlUser.APIObjectItem
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
		state := workspace{
			ViewState: defaultWholeViewState(),
		}

		err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, filePath, &state, false)
		if err != nil {
			return nil, api.MakeNotFoundError(filePath)
		}

		listing = append(listing, workspaceSummary{ID: state.Name, Name: state.Name, APIObjectItem: state.APIObjectItem})
	}

	return listing, nil
}

func savedViewStateGet(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	viewStateID := params.PathParams[idIdentifier]

	s3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, viewStateID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(viewStateID)
	if isSharedReq {
		s3Path = filepaths.GetWorkspacePath(pixlUser.ShareUserID, datasetID, strippedID)
		viewStateID = strippedID
	}

	state := workspace{
		ViewState: defaultWholeViewState(),
	}

	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &state, false)
	if err != nil {
		return nil, api.MakeNotFoundError(viewStateID)
	}

	applyQuantByROIFallback(&state.ViewState.Quantification)

	// Remove any view state items which are not shown by analysis layout
	filterUnusedWidgetStates(&state.ViewState)

	return &state, nil
}

func savedViewStatePut(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	viewStateID := params.PathParams[idIdentifier]

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
	stateToSave := workspace{
		ViewState: defaultWholeViewState(),
		APIObjectItem: &pixlUser.APIObjectItem{
			Shared:  false,
			Creator: params.UserInfo,
		},
	}
	err = json.Unmarshal(body, &stateToSave)
	if err != nil {
		return nil, err
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

func savedViewStateRenamePost(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	existingViewStateID := params.PathParams[idIdentifier]

	// Body should contain new name
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	newViewStateID := string(body)

	// NOTE: we convert the new ID to a saveable one here
	newViewStateID = fileaccess.MakeValidObjectName(newViewStateID)

	// Ensure new name is valid and different
	if len(newViewStateID) <= 0 || newViewStateID == existingViewStateID {
		return nil, api.MakeBadRequestError(errors.New("New workspace name must be different to previous name"))
	}

	// Check that it exists first!
	existingS3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, existingViewStateID)

	state := workspace{
		ViewState: defaultWholeViewState(),
	}
	err = params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, existingS3Path, &state, false)
	if err != nil {
		return nil, api.MakeNotFoundError(existingViewStateID)
	}

	applyQuantByROIFallback(&state.ViewState.Quantification)

	// Work out the path to save to
	newS3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, newViewStateID)

	// Now the tedious part - look through all collections, if we find this item in there, rename in the collection
	listing, err := getViewStateCollectionListing(params.Svcs, datasetID, params.UserInfo.UserID)
	if err != nil {
		return nil, err
	}

	// Run through each collection, retrieve and complain if we find the view state in it
	for _, listingItem := range listing {
		collectionS3Path := filepaths.GetCollectionPath(params.UserInfo.UserID, datasetID, listingItem.Name)

		err = renameWorkspaceIfExistsInCollection(params.Svcs, listingItem.Name, collectionS3Path, existingViewStateID, newViewStateID)
		// Here we can only log! If we fail here, we're part-way through a "transaction" type situation... even if we fail to update
		// one collection, we'd prefer the rest to be updated and the operation to go through
		params.Svcs.Log.Errorf("Failed to check collection \"%v\" in case it needs workspace \"%v\" renamed to: \"%v\". Error: %v", collectionS3Path, existingViewStateID, newViewStateID, err)
	}

	// Delete the old view state file
	err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, existingS3Path)
	if err != nil {
		// Here we can only log! If we fail here, we've renamed in collections but don't have a new copy yet!
		params.Svcs.Log.Errorf("Failed to delete old named workspace: %v", existingS3Path)
		//return nil, err
	}

	// Save under the new name
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, newS3Path, state)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func renameWorkspaceIfExistsInCollection(svcs *services.APIServices, collectionID string, collectionS3Path string, existingViewStateID string, newViewStateID string) error {
	collectionContents := workspaceCollection{ViewStateIDs: []string{}}

	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, collectionS3Path, &collectionContents, false)
	if err != nil {
		return api.MakeStatusError(http.StatusNotFound, fmt.Errorf("Failed to load collection: \"%v\"", collectionID))
	}

	// Run through all items in the file
	collectionToSave := []string{}
	renames := 0
	for _, workspace := range collectionContents.ViewStateIDs {
		toSave := workspace
		if workspace == existingViewStateID {
			renames++
			toSave = newViewStateID
		}

		collectionToSave = append(collectionToSave, toSave)
	}

	if renames > 0 {
		// we've renamed at least one workspace in this collection, so we have to save it
		collectionContents.ViewStateIDs = collectionToSave

		// Ensure we aren't writing view state info to the collection file, as this should ONLY be called for non-shared!
		collectionContents.ViewStates = nil

		err = svcs.FS.WriteJSON(svcs.Config.UsersBucket, collectionS3Path, collectionContents)
		if err != nil {
			return err
		}
	}

	return nil
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
	state := workspace{}
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

			collectionContents := workspaceCollection{ViewStateIDs: []string{}}

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

	// Read the file in
	state := workspace{
		ViewState: defaultWholeViewState(),
	}

	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &state, false)
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
		exprs := dataExpression.DataExpressionLookup{}

		// Read user items
		err := dataExpression.GetListing(params.Svcs, params.UserInfo.UserID, &exprs)

		if err != nil {
			params.Svcs.Log.Errorf("Failed to load user expressions file, returned items may not have name/creator. Error: %v", err)
		}

		// Read shared items
		err = dataExpression.GetListing(params.Svcs, pixlUser.ShareUserID, &exprs)
		if err != nil {
			params.Svcs.Log.Errorf("Failed to load shared expressions file, returned items may not have name/creator. Error: %v", err)
		}

		for idx, expr := range ids.Expressions {
			if item, ok := exprs[expr.ID]; ok {
				ids.Expressions[idx].Name = item.Name
				ids.Expressions[idx].Creator = item.Creator
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
	state := workspace{
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

	// Set shared owner info
	if state.APIObjectItem == nil {
		// Initially didn't have this field, so if anyone shares one where it didn't exist, they are set
		// as the creator. This works well because you can only see your own view states
		state.APIObjectItem = &pixlUser.APIObjectItem{
			Shared:  true,
			Creator: params.UserInfo,
		}
	} else {
		state.Shared = true
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
