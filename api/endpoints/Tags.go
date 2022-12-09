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
	"fmt"
	"io/ioutil"
	"net/http"
	"sort"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/handlers"
	"github.com/pixlise/core/v2/api/permission"
	apiRouter "github.com/pixlise/core/v2/api/router"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
	"github.com/pixlise/core/v2/core/pixlUser"
	tagModel "github.com/pixlise/core/v2/core/tagModel"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Tags - Object Tagging Service

type tagHandler struct {
	svcs *services.APIServices
}

func registerTagHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "tags"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), tagList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), tagPost)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), tagDelete)
}

func tagList(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetTagPath(pixlUser.ShareUserID)

	tags := tagModel.TagLookup{}
	err := tagModel.GetTags(params.Svcs, s3Path, &tags)
	if err != nil {
		return nil, err
	}

	// Read tag IDs in alphabetical order, else we randomly fail unit test
	tagIDs := []string{}
	for k := range tags {
		tagIDs = append(tagIDs, k)
	}
	sort.Strings(tagIDs)

	// Update Tag creator names/emails
	for _, tagID := range tagIDs {
		tag := tags[tagID]
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(tag.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Errorf("Failed to lookup user details for ID: %v, creator name in file: %v (Tag listing). Error: %v", tag.Creator.UserID, tag.Creator.Name, creatorErr)
		} else {
			tag.Creator = updatedCreator
		}
	}

	// Return the combined set
	return &tags, nil
}

func createTag(params handlers.ApiHandlerParams, tag tagModel.Tag) (string, error) {
	if tag.DatasetID == "" {
		tag.DatasetID = params.PathParams[datasetIdentifier]
	}

	s3Path := filepaths.GetTagPath(pixlUser.ShareUserID)

	tags := tagModel.TagLookup{}
	err := tagModel.GetTags(params.Svcs, s3Path, &tags)
	if err != nil {
		return "", err
	}

	// Download the file
	if err != nil && !params.Svcs.FS.IsNotFoundError(err) {
		// Only return error if it's not about the file missing, because user may not have interacted with this dataset yet
		return "", err
	}

	for _, currentTag := range tags {
		if tag.Name == currentTag.Name {
			return "", api.MakeBadRequestError(fmt.Errorf("tag name already used: %v", tag.Name))
		}
	}

	// If saveID is empty, this means it's either new or we're not overwriting, so we generate a new one
	saveID := params.Svcs.IDGen.GenObjectID()
	_, exists := tags[saveID]
	if exists {
		return "", fmt.Errorf("failed to generate unique ID")
	}

	tag.ID = saveID
	tag.Creator = params.UserInfo
	tag.DateCreated = params.Svcs.TimeStamper.GetTimeNowSec()

	tags[saveID] = tag

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, tags)

	return saveID, err
}

func tagPost(params handlers.ApiHandlerParams) (interface{}, error) {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var tag tagModel.Tag
	err = json.Unmarshal(body, &tag)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	tagID, err := createTag(params, tag)

	validTag := tagModel.TagID{}
	validTag.ID = tagID

	return validTag, err
}

func tagDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	tagID := params.PathParams[idIdentifier]
	s3Path := filepaths.GetTagPath(pixlUser.ShareUserID)

	tags := tagModel.TagLookup{}
	err := tagModel.GetTags(params.Svcs, s3Path, &tags)
	if err != nil {
		return nil, err
	}

	item, ok := tags[tagID]
	if !ok {
		return nil, api.MakeNotFoundError(tagID)
	}

	// Only allow shared tags to be deleted if same user requested it
	if item.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", tagID, params.UserInfo.UserID))
	}

	delete(tags, tagID)
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, tags)

	return nil, err
}
