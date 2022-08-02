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
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/pixlUser"
	"github.com/pixlise/core/core/utils"
)

type workspaceCollection struct {
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
	userList, _ := getViewStateCollectionListing(params.Svcs, datasetID, params.UserInfo.UserID)
	sharedList, _ := getViewStateCollectionListing(params.Svcs, datasetID, pixlUser.ShareUserID)

	result := []workspaceCollectionListItem{}
	for _, item := range userList {
		result = append(result, item)
	}

	for _, item := range sharedList {
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

func getCollection(params handlers.ApiHandlerParams, collectionID string, s3Path string, loadChildViewStates bool) (workspaceCollection, error) {
	// Read the collection file itself - it may or may not contain the view states saved in it too (we want to be
	// saving that into shared collection files only)
	collectionContents := workspaceCollection{ViewStateIDs: []string{}}
	err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &collectionContents, false)
	if err != nil {
		return collectionContents, api.MakeNotFoundError(collectionID)
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

		// Set up a default view state to read into
		loadedWorkspace := workspace{
			ViewState: defaultWholeViewState(),
		}
		err := params.Svcs.FS.ReadJSON(params.Svcs.Config.UsersBucket, s3Path, &loadedWorkspace, false)
		if err != nil {
			return nil, api.MakeNotFoundError(viewStateID)
		}

		applyQuantByROIFallback(&loadedWorkspace.ViewState.Quantification)

		result[viewStateID] = loadedWorkspace.ViewState
	}

	return result, nil
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

	collectionToSave := workspaceCollection{ViewStateIDs: []string{}, ViewStates: nil}
	err = json.Unmarshal(body, &collectionToSave)
	if err != nil {
		return nil, err
	}

	// Ensure names match
	if collectionToSave.Name != collectionID {
		collectionToSave.Name = collectionID
	}

	// Include creator info
	collectionToSave.APIObjectItem = &pixlUser.APIObjectItem{
		Shared:  false,
		Creator: params.UserInfo,
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

	collection := workspaceCollection{}
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

	s3Path := filepaths.GetCollectionPath(params.UserInfo.UserID, datasetID, collectionID)
	_, isSharedReq := utils.StripSharedItemIDPrefix(collectionID)
	if isSharedReq {
		return nil, fmt.Errorf("Cannot share a shared ID")
	}

	collectionContents, err := getCollection(params, collectionID, s3Path, true)
	if err != nil {
		return nil, err
	}

	// Set shared flag
	if collectionContents.APIObjectItem == nil {
		collectionContents.APIObjectItem = &pixlUser.APIObjectItem{
			Shared:  true,
			Creator: params.UserInfo,
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
