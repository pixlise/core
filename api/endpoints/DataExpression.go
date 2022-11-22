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
	dataExpression "github.com/pixlise/core/v2/core/expression"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// DataExpressions - storing/retrieving/sharing expressions for data, for context images and widgets

func registerDataExpressionHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "data-expression"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), dataExpressionList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), dataExpressionPost)

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), dataExpressionPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), dataExpressionDelete)

	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedExpression), dataExpressionShare)
}

func dataExpressionList(params handlers.ApiHandlerParams) (interface{}, error) {
	items := dataExpression.DataExpressionLookup{}

	// Get user expressions
	err := dataExpression.GetListing(params.Svcs, params.UserInfo.UserID, &items)
	if err != nil {
		return nil, err
	}

	// Get shared expressions (into same map)
	err = dataExpression.GetListing(params.Svcs, pixlUser.ShareUserID, &items)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(item.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Errorf("Failed to lookup user details for ID: %v, creator name in file: %v (Expressions listing). Error: %v", item.Creator.UserID, item.Creator.Name, creatorErr)
		} else {
			item.Creator = updatedCreator
		}
	}

	// Return the combined set
	return &items, nil
}

func setupForSave(params handlers.ApiHandlerParams, s3Path string) (*dataExpression.DataExpressionLookup, *dataExpression.DataExpressionInput, error) {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, nil, err
	}

	var req dataExpression.DataExpressionInput
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, nil, api.MakeBadRequestError(err)
	}

	// Validate
	if !fileaccess.IsValidObjectName(req.Name) {
		return nil, nil, api.MakeBadRequestError(fmt.Errorf("Invalid ID: %v", req.Name))
	}
	err = req.Type.IsValid()
	if err != nil {
		return nil, nil, api.MakeBadRequestError(err)
	}
	if len(req.Expression) <= 0 {
		return nil, nil, api.MakeBadRequestError(errors.New("Expression cannot be empty"))
	}

	// Download the file
	items, err := dataExpression.ReadExpressionData(params.Svcs, s3Path)
	if err != nil && !params.Svcs.FS.IsNotFoundError(err) {
		// Only return error if it's not about the file missing, because user may not have interacted with this dataset yet
		return nil, nil, err
	}

	return &items, &req, nil
}

func dataExpressionPost(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetExpressionPath(params.UserInfo.UserID)
	expressions, req, err := setupForSave(params, s3Path)
	if err != nil {
		return nil, err
	}

	// Save it & upload
	saveID := params.Svcs.IDGen.GenObjectID()
	_, exists := (*expressions)[saveID]
	if exists {
		return nil, fmt.Errorf("Failed to generate unique ID")
	}

	(*expressions)[saveID] = dataExpression.DataExpression{DataExpressionInput: req, APIObjectItem: &pixlUser.APIObjectItem{Shared: false, Creator: params.UserInfo}}

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *expressions)
	if err != nil {
		return nil, err
	}

	return *expressions, nil
}

func dataExpressionPut(params handlers.ApiHandlerParams) (interface{}, error) {
	itemID := params.PathParams[idIdentifier]
	if _, isSharedReq := utils.StripSharedItemIDPrefix(itemID); isSharedReq {
		return nil, api.MakeBadRequestError(errors.New("Cannot edit shared expressions"))
	}

	s3Path := filepaths.GetExpressionPath(params.UserInfo.UserID)
	expressions, req, err := setupForSave(params, s3Path)
	if err != nil {
		return nil, err
	}

	existing, ok := (*expressions)[itemID]
	if !ok {
		return nil, api.MakeNotFoundError(itemID)
	}

	// Save it & upload
	(*expressions)[itemID] = dataExpression.DataExpression{DataExpressionInput: req, APIObjectItem: &pixlUser.APIObjectItem{Shared: false, Creator: existing.Creator}}
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *expressions)
	if err != nil {
		return nil, err
	}

	return *expressions, nil
}

func dataExpressionDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	// If deleting a shared item, we need to strip the prefix from the ID and also ensure that only the creator can delete
	itemID := params.PathParams[idIdentifier]
	s3Path := filepaths.GetExpressionPath(params.UserInfo.UserID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetExpressionPath(pixlUser.ShareUserID)
		itemID = strippedID
	}

	// Using path params, work out path
	items, err := dataExpression.ReadExpressionData(params.Svcs, s3Path)
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

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, items)
	if err != nil {
		return nil, err
	}

	return items, nil
}

func dataExpressionShare(params handlers.ApiHandlerParams) (interface{}, error) {
	// User is supplying an ID of an object to share. We should be able to find it in the users own data file
	// and put it in the shared file with a new ID, thereby implementing "share a copy"
	idToFind := params.PathParams[idIdentifier]

	sharedIDs, err := shareExpressions(params.Svcs, params.UserInfo.UserID, []string{idToFind})
	if err != nil {
		return nil, err
	}

	// shared IDs should only contain one item!
	if len(sharedIDs) != 1 {
		return nil, errors.New("Failed to share expression with ID: " + idToFind)
	}

	// Return just the one shared id
	return sharedIDs[0], nil
}

func shareExpressions(svcs *services.APIServices, userID string, expressionIDs []string) ([]string, error) {
	generatedIDs := []string{}
	// Read user items
	s3Path := filepaths.GetExpressionPath(userID)
	userItems, err := dataExpression.ReadExpressionData(svcs, s3Path)

	if err != nil {
		return generatedIDs, err
	}

	// Read shared items
	sharedS3Path := filepaths.GetExpressionPath(pixlUser.ShareUserID)
	sharedItems, err := dataExpression.ReadExpressionData(svcs, sharedS3Path)

	if err != nil {
		return generatedIDs, err
	}

	// Run through and share each one
	for _, id := range expressionIDs {
		exprItem, ok := userItems[id]
		if !ok {
			return generatedIDs, api.MakeNotFoundError(id)
		}

		// We found it, now generate id to save it to
		sharedID := svcs.IDGen.GenObjectID()
		_, ok = sharedItems[sharedID]
		if ok {
			return generatedIDs, fmt.Errorf("Failed to generate unique share ID for " + id)
		}

		// Add it to the shared file and we're done
		sharedCopy := dataExpression.DataExpression{
			DataExpressionInput: &dataExpression.DataExpressionInput{
				Name:       exprItem.Name,
				Expression: exprItem.Expression,
				Type:       exprItem.Type,
				Comments:   exprItem.Comments,
			},
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:  true,
				Creator: exprItem.Creator,
			},
		}

		sharedItems[sharedID] = sharedCopy
		generatedIDs = append(generatedIDs, sharedID)
	}

	// Save the shared file
	return generatedIDs, svcs.FS.WriteJSON(svcs.Config.UsersBucket, sharedS3Path, sharedItems)
}
