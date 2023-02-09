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

// Package classification Data Expression API
//
// Documentation for the Data Expression Endpoints, used for storing/retrieving/sharing expressions for data, for context images and widgets
//
//  Schemes: http
//  BasePath: /
//  Version: 1.0.0
//
//  Consumes:
//  - application/json
//
//  Produces:
//  - application/json
//
//     SecurityDefinitions:
//     oauth:
//         type: oauth2
//         flow: accessCode
//         authorizationUrl: 'https://accounts.google.com/o/oauth2/v2/auth'
//         tokenUrl: 'https://www.googleapis.com/oauth2/v4/token'
//         scopes:
//           write: Admin scope
//           read: User scope
// swagger:meta
package endpoints

import (
	"encoding/json"
	"errors"
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
	dataExpression "github.com/pixlise/core/v2/core/expression"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/utils"
)

// swagger:response genericError
//in: body
type _ string

// swagger:response deleteResponse
//in: body
type deleteResponse map[string]string

// swagger:response shareResponse
//in: body
type shareResponse string

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

	// swagger:route GET /data-expression data-expression dataExpressionList
	//
	// Lists available data expressions.
	//
	// This will list the data expressions available to the user.
	//
	//     Consumes:
	//     - application/json
	//
	//     Produces:
	//     - application/json
	//     - text/plain
	//
	//     Schemes: http, https
	//
	//     Deprecated: false
	//
	//     Security:
	//       oauth: read, write
	//
	//     Responses:
	//       default: genericError
	//       200: dataExpressionLookup
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

	// Read keys in alphabetical order, else we randomly fail unit test
	keys := []string{}
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Update all creator infos
	for _, k := range keys {
		item := items[k]

		// Ensure tags is not nil as this is a new field
		if item.Tags == nil {
			item.Tags = []string{}
		}

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

	if req.Tags == nil {
		req.Tags = []string{}
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
	// swagger:route POST /data-expression data-expression dataExpressionPost
	//
	// Post a new data expression.
	//
	// Creates a new data expression for a user.
	//
	//     Consumes:
	//     - application/json
	//
	//     Produces:
	//     - application/json
	//
	//     Schemes: http, https
	//
	//     Deprecated: false
	//
	//     Security:
	//       oauth: read, write
	//
	//     Responses:
	//       default: genericError
	//       200: dataExpressionLookup

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

	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()

	(*expressions)[saveID] = dataExpression.DataExpression{
		DataExpressionInput: req,
		APIObjectItem: &pixlUser.APIObjectItem{
			Shared:              false,
			Creator:             params.UserInfo,
			CreatedUnixTimeSec:  timeNow,
			ModifiedUnixTimeSec: timeNow,
		},
	}

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *expressions)
	if err != nil {
		return nil, err
	}

	// Only return new item
	response := dataExpression.DataExpressionLookup{}
	response[saveID] = (*expressions)[saveID]

	return response, nil
}

func dataExpressionPut(params handlers.ApiHandlerParams) (interface{}, error) {
	// swagger:route PUT /data-expression/{id} data-expression dataExpressionPut
	//
	// Update an existing data expression.
	//
	// Updates and existing expression for a user.
	//
	//     Consumes:
	//     - application/json
	//
	//     Produces:
	//     - application/json
	//
	//     Schemes: http, https
	//
	//     Parameters:
	//       + name: id
	//         in: path
	//         description: the id of the expression to delete
	//         required: true
	//         type: string
	//
	//     Deprecated: false
	//
	//     Security:
	//       oauth: read, write
	//
	//     Responses:
	//       default: genericError
	//       200: dataExpressionLookup

	itemID := params.PathParams[idIdentifier]

	s3Path := filepaths.GetExpressionPath(params.UserInfo.UserID)
	id, isSharedReq := utils.StripSharedItemIDPrefix(itemID)

	if isSharedReq {
		s3Path = filepaths.GetExpressionPath(pixlUser.ShareUserID)
	}

	expressions, req, err := setupForSave(params, s3Path)
	if err != nil {
		return nil, err
	}

	existing, ok := (*expressions)[id]
	if !ok {
		return nil, api.MakeNotFoundError(id)
	}

	if isSharedReq && params.UserInfo.UserID != existing.Creator.UserID {
		return nil, api.MakeBadRequestError(errors.New("cannot edit shared expression not owned by user"))
	}

	// Save it & upload
	(*expressions)[id] = dataExpression.DataExpression{
		DataExpressionInput: req,
		APIObjectItem: &pixlUser.APIObjectItem{
			Shared:              isSharedReq,
			Creator:             existing.Creator,
			CreatedUnixTimeSec:  existing.CreatedUnixTimeSec,
			ModifiedUnixTimeSec: params.Svcs.TimeStamper.GetTimeNowSec(),
		},
	}
	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *expressions)
	if err != nil {
		return nil, err
	}

	// Use non-stripped ID for response
	response := dataExpression.DataExpressionLookup{}
	response[itemID] = (*expressions)[id]

	return response, nil
}

func dataExpressionDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	// swagger:route DELETE /data-expression/{id} data-expression dataExpressionDelete
	//
	// Deletes a data expression.
	//
	// This endpoint deletes an existing data expression.
	//
	//     Consumes:
	//     - application/json
	//
	//     Produces:
	//     - application/json
	//
	//     Schemes: http, https
	//
	//     Deprecated: false
	//
	//     Parameters:
	//       + name: id
	//         in: path
	//         description: the id of the expression to delete
	//         required: true
	//         type: string
	//
	//     Security:
	//       oauth: read, write
	//
	//     Responses:
	//       default: genericError
	//       200: deleteResponse

	// If deleting a shared item, we need to strip the prefix from the ID and also ensure that only the creator can delete
	itemID := params.PathParams[idIdentifier]
	s3Path := filepaths.GetExpressionPath(params.UserInfo.UserID)

	id, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetExpressionPath(pixlUser.ShareUserID)
	}

	// Using path params, work out path
	items, err := dataExpression.ReadExpressionData(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	sharedItem, ok := items[id]
	if !ok {
		return nil, api.MakeNotFoundError(id)
	}

	if isSharedReq && sharedItem.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", id, params.UserInfo.UserID))
	}

	// Found it, delete & we're done
	delete(items, id)

	err = params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, items)
	if err != nil {
		return nil, err
	}

	// Return just the one deleted id
	response := deleteResponse{}
	response[itemID] = itemID

	return response, nil
}

func dataExpressionShare(params handlers.ApiHandlerParams) (interface{}, error) {
	// swagger:route POST /share/data-expression/{id} data-expression dataExpressionShare
	//
	// Shares a data expression.
	//
	// This endpoint shares an existing data expression with other users.
	//
	//     Consumes:
	//     - application/json
	//
	//     Produces:
	//     - application/json
	//
	//     Schemes: http, https
	//
	//     Deprecated: false
	//
	//     Parameters:
	//       + name: id
	//         in: path
	//         description: the id of the expression to delete
	//         required: true
	//         type: string
	//
	//     Security:
	//       oauth: read, write
	//
	//     Responses:
	//       default: genericError
	//       200: shareResponse

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

		tags := exprItem.Tags
		if tags == nil {
			tags = []string{}
		}

		// Add it to the shared file and we're done
		timeNow := svcs.TimeStamper.GetTimeNowSec()
		sharedCopy := dataExpression.DataExpression{
			DataExpressionInput: &dataExpression.DataExpressionInput{
				Name:       exprItem.Name,
				Expression: exprItem.Expression,
				Type:       exprItem.Type,
				Tags:       tags,
				Comments:   exprItem.Comments,
			},
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:              true,
				Creator:             exprItem.Creator,
				CreatedUnixTimeSec:  exprItem.CreatedUnixTimeSec,
				ModifiedUnixTimeSec: timeNow,
			},
		}

		sharedItems[sharedID] = sharedCopy
		generatedIDs = append(generatedIDs, sharedID)
	}

	// Save the shared file
	return generatedIDs, svcs.FS.WriteJSON(svcs.Config.UsersBucket, sharedS3Path, sharedItems)
}
