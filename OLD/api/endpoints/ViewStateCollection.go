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
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/utils"
)

type WorkspaceCollection struct {
	ViewStateIDs []string `json:"viewStateIDs"`
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	// Optional - we ONLY store this for shared workspaces, but for users own collections
	// the get call downloads the individual workspaces and returns this field. This way
	// the UI can always expect this to exist, but API only saves it when a snapshot is
	// required (sharing)
	ViewStates map[string]wholeViewState `json:"viewStates"`
	*pixlUser.APIObjectItem
}

func (a WorkspaceCollection) SetTimes(userID string, t int64) {
	if a.APIObjectItem == nil {
		log.Printf("WorkspaceCollection: %v has no APIObjectItem\n", a.Name)
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

type workspaceCollectionListItem struct {
	Name                string `json:"name"`
	ModifiedUnixTimeSec int64  `json:"modifiedUnixSec"`
}

// CRUD operations for collections of view states
// - Each item is a list of strings (view state titles)
// - All stored in 1 file on S3
// - If user edits one, file is rewritten

func viewStateCollectionList(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	isPublicUser := !params.UserInfo.Permissions[permission.PermReadPIXLISESettings]
	publicObjectsAuth := permission.PublicObjectsAuth{}

	if isPublicUser {
		// Verify user has access to dataset (need to do this now that permissions are on a per-dataset basis)
		_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, params.Svcs.Config.ConfigBucket, datasetID)
		if err != nil {
			return nil, err
		}

		publicObjectsAuth, err = permission.GetPublicObjectsAuth(params.Svcs.FS, params.Svcs.Config.ConfigBucket, isPublicUser)
		if err != nil {
			return nil, err
		}
	}

	userList, _ := getViewStateCollectionListing(params.Svcs, datasetID, params.UserInfo.UserID)
	sharedList, _ := getViewStateCollectionListing(params.Svcs, datasetID, pixlUser.ShareUserID)

	result := []workspaceCollectionListItem{}

	// If user is not public, return their own collections
	if !isPublicUser {
		result = append(result, userList...)
	}

	for _, item := range sharedList {
		if isPublicUser {
			isCollectionPublic, err := permission.CheckIsObjectInPublicSet(publicObjectsAuth.Collections, item.Name)
			if err != nil {
				return nil, err
			}

			if !isCollectionPublic {
				continue
			}
		}
		item.Name = utils.SharedItemIDPrefix + item.Name
		result = append(result, item)
	}

	return result, nil
}

func getViewStateCollectionListing(svcs *services.APIServices, datasetID string, userID string) ([]workspaceCollectionListItem, error) {
	s3Path := filepaths.GetCollectionPath(userID, datasetID, "")

	// Return each name
	listingResp, err := svcs.S3.ListObjectsV2(
		&s3.ListObjectsV2Input{
			Bucket: aws.String(svcs.Config.UsersBucket),
			Prefix: aws.String(s3Path),
		},
	)

	if err != nil {
		svcs.Log.Errorf("Failed to list view state collections in %v/%v: %v", svcs.Config.UsersBucket, s3Path, err)
		return []workspaceCollectionListItem{}, api.MakeStatusError(http.StatusInternalServerError, errors.New("Failed to list view state collections"))
	}

	result := []workspaceCollectionListItem{}

	for _, listingItem := range listingResp.Contents {
		fileName := path.Base(*listingItem.Key)
		fileExt := path.Ext(fileName)

		result = append(result, workspaceCollectionListItem{
			Name:                fileName[0 : len(fileName)-len(fileExt)],
			ModifiedUnixTimeSec: listingItem.LastModified.Unix(),
		})
	}

	return result, nil
}

func viewStateCollectionGet(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	collectionID := params.PathParams[idIdentifier]

	isPublicUser := !params.UserInfo.Permissions[permission.PermReadPIXLISESettings]

	if isPublicUser {
		// Verify user has access to dataset (need to do this now that permissions are on a per-dataset basis)
		_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, params.Svcs.Config.ConfigBucket, datasetID)
		if err != nil {
			return nil, err
		}

		isCollectionPublic, err := permission.CheckIsObjectPublic(params.Svcs.FS, params.Svcs.Config.ConfigBucket, permission.PublicObjectCollection, collectionID)
		if err != nil {
			return nil, err
		}

		if !isCollectionPublic {
			return nil, api.MakeBadRequestError(errors.New("workspace is not public"))
		}
	}

	s3Path := filepaths.GetCollectionPath(params.UserInfo.UserID, datasetID, collectionID)
	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(collectionID)
	if isSharedReq {
		s3Path = filepaths.GetCollectionPath(pixlUser.ShareUserID, datasetID, strippedID)
		collectionID = strippedID
	}

	collectionContents, err := getCollection(params, collectionID, s3Path, !isSharedReq)
	if err != nil {
		return nil, err
	}

	return &collectionContents, nil
}

func getCollection(params handlers.ApiHandlerParams, collectionID string, s3Path string, loadChildViewStates bool) (WorkspaceCollection, error) {
	// Read the collection file itself - it may or may not contain the view states saved in it too (we want to be
	// saving that into shared collection files only)
	collectionContents := WorkspaceCollection{ViewStateIDs: []string{}}
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &collectionContents, false)
	if err != nil {
		return collectionContents, api.MakeNotFoundError(collectionID)
	}

	if collectionContents.APIObjectItem != nil {
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(collectionContents.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Infof("Failed to lookup user details for ID: %v, creator name in file: %v (collection GET). Error: %v", collectionContents.Creator.UserID, collectionContents.Creator.Name, creatorErr)
		} else {
			collectionContents.Creator = updatedCreator
		}
	}

	// If caller wants, we can load the child view states
	if loadChildViewStates {
		states, err := loadViewStates(params, collectionContents.ViewStateIDs)
		if err != nil {
			return collectionContents, api.MakeStatusError(http.StatusNotFound, err)
		}

		collectionContents.ViewStates = states
	}

	return collectionContents, nil
}

func loadViewStates(params handlers.ApiHandlerParams, viewStateIDs []string) (map[string]wholeViewState, error) {
	datasetID := params.PathParams[datasetIdentifier]
	result := map[string]wholeViewState{}

	for _, viewStateID := range viewStateIDs {
		s3Path := filepaths.GetWorkspacePath(params.UserInfo.UserID, datasetID, viewStateID)

		// If this is a shared view state, load it from the shared user's workspace
		strippedID, isShared := utils.StripSharedItemIDPrefix(viewStateID)
		if isShared {
			s3Path = filepaths.GetWorkspacePath(pixlUser.ShareUserID, datasetID, strippedID)
		}

		// Set up a default view state to read into
		loadedWorkspace := Workspace{
			ViewState: defaultWholeViewState(),
		}

		err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &loadedWorkspace, false)
		if err != nil {
			return nil, api.MakeNotFoundError(strippedID)
		}

		applyQuantByROIFallback(&loadedWorkspace.ViewState.Quantification)

		result[strippedID] = loadedWorkspace.ViewState
	}

	return result, nil
}

func viewStateCollectionPostPublic(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	collectionID := params.PathParams[idIdentifier]

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(collectionID)
	if !isSharedReq {
		return nil, api.MakeBadRequestError(errors.New("can't make non-shared collections public"))
	}

	// Verify user has access to dataset (need to do this now that permissions are on a per-dataset basis)
	_, err := permission.UserCanAccessDatasetWithSummaryDownload(params.Svcs.FS, params.UserInfo, params.Svcs.Config.DatasetsBucket, params.Svcs.Config.ConfigBucket, datasetID)
	if err != nil {
		return nil, err
	}

	// Verify the dataset is public before sharing contained objects
	isDatasetPublic, err := permission.CheckIsPublicDataset(params.Svcs.FS, params.Svcs.Config.ConfigBucket, datasetID)
	if err != nil {
		return nil, err
	}

	if !isDatasetPublic {
		return nil, api.MakeBadRequestError(errors.New("cannot share a collection from a non-public dataset"))
	}

	// Verify user has access to collection
	s3SharedPath := filepaths.GetCollectionPath(pixlUser.ShareUserID, datasetID, strippedID)
	collection, err := getCollection(params, strippedID, s3SharedPath, true)
	if err != nil {
		return nil, err
	}

	// Get all existing public objects
	publicObjectsPath := filepaths.GetPublicObjectsPath()
	publicObjects, err := permission.ReadPublicObjectsAuth(params.Svcs.FS, params.Svcs.Config.ConfigBucket, publicObjectsPath)
	if err != nil {
		// Assume the file doesn't exist yet
		log.Printf("No public objects file found, creating new one")
		publicObjects = permission.PublicObjectsAuth{}
	}

	// Get all view states in collection
	states, err := loadViewStates(params, collection.ViewStateIDs)
	if err != nil {
		return nil, err
	}

	// Get flat lists of all objects in view states
	collectionObjects := publicObjects

	// Ensure all lists are initialised
	if collectionObjects.Expressions == nil {
		collectionObjects.Expressions = []string{}
	}
	if collectionObjects.Modules == nil {
		collectionObjects.Modules = []string{}
	}
	if collectionObjects.ROIs == nil {
		collectionObjects.ROIs = []string{}
	}
	if collectionObjects.RGBMixes == nil {
		collectionObjects.RGBMixes = []string{}
	}
	if collectionObjects.Quantifications == nil {
		collectionObjects.Quantifications = []string{}
	}
	if collectionObjects.Workspaces == nil {
		collectionObjects.Workspaces = []string{}
	}
	if collectionObjects.Collections == nil {
		collectionObjects.Collections = []string{}
	}
	if collectionObjects.Datasets == nil {
		collectionObjects.Datasets = []string{}
	}

	// Add dataset to public objects
	if !utils.StringInSlice(datasetID, collectionObjects.Datasets) {
		collectionObjects.Datasets = append(collectionObjects.Datasets, datasetID)
	}

	// Add collection to public objects
	if !utils.StringInSlice(strippedID, collectionObjects.Collections) {
		collectionObjects.Collections = append(collectionObjects.Collections, strippedID)
	}

	for viewStateID, state := range states {

		// Add view state to public objects
		if !utils.StringInSlice(viewStateID, collectionObjects.Workspaces) {
			collectionObjects.Workspaces = append(collectionObjects.Workspaces, viewStateID)
		}

		// Add all objects in view state to public objects
		referencedIDs := state.getReferencedIDs()

		for _, roi := range referencedIDs.ROIs {
			if !utils.StringInSlice(roi.ID, collectionObjects.ROIs) {
				collectionObjects.ROIs = append(collectionObjects.ROIs, roi.ID)
			}
		}

		for _, expression := range referencedIDs.Expressions {
			if !utils.StringInSlice(expression.ID, collectionObjects.Expressions) {
				collectionObjects.Expressions = append(collectionObjects.Expressions, expression.ID)
			}

			strippedID, _ := utils.StripSharedItemIDPrefix(expression.ID)
			expr, err := params.Svcs.Expressions.GetExpression(strippedID, true)
			if err != nil {
				continue
			}

			// Still need to check expression module references even if it is already public because
			// references may have changed since it was made public
			for _, module := range expr.ModuleReferences {
				if !utils.StringInSlice(module.ModuleID, collectionObjects.Modules) {
					collectionObjects.Modules = append(collectionObjects.Modules, module.ModuleID)
				}
			}
		}

		for _, rgbmixes := range referencedIDs.RGBMixes {
			if !utils.StringInSlice(rgbmixes.ID, collectionObjects.RGBMixes) {
				collectionObjects.RGBMixes = append(collectionObjects.RGBMixes, rgbmixes.ID)
			}
		}

		if !utils.StringInSlice(referencedIDs.Quant.ID, collectionObjects.Quantifications) {
			collectionObjects.Quantifications = append(collectionObjects.Quantifications, referencedIDs.Quant.ID)
		}
	}

	// Save public objects
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.ConfigBucket, publicObjectsPath, collectionObjects)
	if err != nil {
		return nil, err
	}

	return collectionObjects, nil
}

func viewStateCollectionPut(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	collectionID := params.PathParams[idIdentifier]

	// Cant write to shared ones
	_, isSharedReq := utils.StripSharedItemIDPrefix(collectionID)
	if isSharedReq {
		return nil, api.MakeBadRequestError(errors.New("Can't edit shared collections"))
	}

	s3Path := filepaths.GetCollectionPath(params.UserInfo.UserID, datasetID, collectionID)

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	collectionToSave := WorkspaceCollection{ViewStateIDs: []string{}, ViewStates: nil}
	err = json.Unmarshal(body, &collectionToSave)
	if err != nil {
		return nil, err
	}

	// Ensure names match
	if collectionToSave.Name != collectionID {
		collectionToSave.Name = collectionID
	}

	// Include creator info
	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()
	collectionToSave.APIObjectItem = &pixlUser.APIObjectItem{
		Shared:              false,
		Creator:             params.UserInfo,
		CreatedUnixTimeSec:  timeNow,
		ModifiedUnixTimeSec: timeNow,
	}

	// Save it
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, collectionToSave)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func viewStateCollectionDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	collectionID := params.PathParams[idIdentifier]

	// Get its path, check that only the owner is deleting if shared
	s3Path := filepaths.GetCollectionPath(params.UserInfo.UserID, datasetID, collectionID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(collectionID)
	if isSharedReq {
		s3Path = filepaths.GetCollectionPath(pixlUser.ShareUserID, datasetID, strippedID)
		collectionID = strippedID
	}

	collection := WorkspaceCollection{}
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &collection, false)
	if err != nil {
		return nil, api.MakeNotFoundError("View state collection")
	}

	// If it's not the owner deleting it, reject
	if isSharedReq && collection.APIObjectItem != nil && collection.APIObjectItem.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", collectionID, params.UserInfo.UserID))
	}

	err = params.Svcs.FS.DeleteObject(params.Svcs.Config.UsersBucket, s3Path)
	if err != nil {
		return nil, err
	}

	return nil, nil
}

func viewStateCollectionShare(params handlers.ApiHandlerParams) (interface{}, error) {
	// User is supplying an ID of an object to share. We should be able to find it in the users own data file
	// and put it in the shared directory, thereby implementing "share a copy". The only extra complexity for
	// collection sharing is that we copy all the view state data into the saved file so this no longer
	// has to rely on other
	datasetID := params.PathParams[datasetIdentifier]
	collectionID := params.PathParams[idIdentifier]

	_, isSharedReq := utils.StripSharedItemIDPrefix(collectionID)
	if isSharedReq {
		return nil, fmt.Errorf("cannot share a shared ID")
	}

	s3Path := filepaths.GetCollectionPath(params.UserInfo.UserID, datasetID, collectionID)
	collectionContents, err := getCollection(params, collectionID, s3Path, true)
	if err != nil {
		return nil, err
	}

	// Set shared flag
	if collectionContents.APIObjectItem == nil {
		timeNow := params.Svcs.TimeStamper.GetTimeNowSec()
		collectionContents.APIObjectItem = &pixlUser.APIObjectItem{
			Shared:              true,
			Creator:             params.UserInfo,
			CreatedUnixTimeSec:  timeNow,
			ModifiedUnixTimeSec: timeNow,
		}
	} else {
		collectionContents.Shared = true
	}

	// Write it to the shared area
	s3SharedPath := filepaths.GetCollectionPath(pixlUser.ShareUserID, datasetID, collectionID)
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3SharedPath, collectionContents)
	if err != nil {
		return nil, err
	}

	return collectionID + " shared", nil
}
