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

	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/expressions/expressions"
	"github.com/pixlise/core/v3/core/expressions/modules"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// DataExpressions - storing/retrieving/sharing expressions for data, for context images and widgets

func registerDataExpressionHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "data-expression"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), dataExpressionList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), dataExpressionPost)

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), dataExpressionPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), dataExpressionDelete)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), dataExpressionGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix+"/execution-stat", idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), dataExpressionExecutionStatPut)

	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedExpression), dataExpressionShare)
	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/doi/"+pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedExpression), publishDataExpressionToZenodo)
}

func toWire(expr expressions.DataExpression) expressions.DataExpressionWire {
	orig := expr.Origin
	id := expr.ID
	// If it's shared, we prefix the ID (kept around for legacy reasons)
	if expr.Origin.Shared {
		id = utils.SharedItemIDPrefix + id
	}
	resultItem := expressions.DataExpressionWire{
		ID:               id,
		Name:             expr.Name,
		SourceCode:       expr.SourceCode,
		SourceLanguage:   expr.SourceLanguage,
		Comments:         expr.Comments,
		Tags:             expr.Tags,
		ModuleReferences: expr.ModuleReferences,
		APIObjectItem:    &orig,
		RecentExecStats:  expr.RecentExecStats,
		DOIMetadata:      expr.DOIMetadata,
	}
	return resultItem
}

func dataExpressionList(params handlers.ApiHandlerParams) (interface{}, error) {
	// Get user and expressions
	items, err := params.Svcs.Expressions.ListExpressions(params.UserInfo.UserID, true, true)
	if err != nil {
		return nil, err
	}

	result := map[string]expressions.DataExpressionWire{}

	// We're sending them back in a different struct for legacy reasons
	for _, item := range items {
		resultItem := toWire(item)
		result[resultItem.ID] = resultItem
	}

	// Return the combined set
	return &result, nil
}

func dataExpressionGet(params handlers.ApiHandlerParams) (interface{}, error) {
	itemID := params.PathParams[idIdentifier]
	strippedID, _ := utils.StripSharedItemIDPrefix(itemID)

	// Get expression
	expr, err := params.Svcs.Expressions.GetExpression(strippedID, true)
	if err != nil {
		if params.Svcs.Expressions.IsNotFoundError(err) {
			return nil, api.MakeNotFoundError(itemID)
		}
		return nil, err
	}

	resultItem := toWire(expr)
	return resultItem, nil
}

func readRequest(params handlers.ApiHandlerParams) (*expressions.DataExpressionInput, error) {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req expressions.DataExpressionInput
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Validate - these used to be file names, but lets still keep them sensible
	if !fileaccess.IsValidObjectName(req.Name) {
		return nil, api.MakeBadRequestError(fmt.Errorf("Invalid expression name: %v", req.Name))
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}

	return &req, nil
}

func dataExpressionPost(params handlers.ApiHandlerParams) (interface{}, error) {
	req, err := readRequest(params)
	if err != nil {
		return nil, err
	}

	if len(req.SourceCode) <= 0 {
		return nil, api.MakeBadRequestError(errors.New("Expression source code cannot be empty"))
	}

	// Check there aren't any silly modules configured
	err = isValidModuleReferences(req.ModuleReferences)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	result, err := params.Svcs.Expressions.CreateExpression(*req, params.UserInfo, false)

	resultItem := toWire(result)
	return resultItem, err
}

func dataExpressionPut(params handlers.ApiHandlerParams) (interface{}, error) {
	itemID := params.PathParams[idIdentifier]
	strippedID, _ := utils.StripSharedItemIDPrefix(itemID)

	req, err := readRequest(params)
	if err != nil {
		return nil, err
	}

	// Check there aren't any silly modules configured
	err = isValidModuleReferences(req.ModuleReferences)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Check that it exists, and that this user has the same ID (don't want to allow editing others expressions
	// or editing shared ones you didn't create)
	existingExpr, err := params.Svcs.Expressions.GetExpression(strippedID, false)
	if err != nil {
		return nil, api.MakeNotFoundError(itemID)
	}

	if params.UserInfo.UserID != existingExpr.Origin.Creator.UserID {
		return nil, api.MakeBadRequestError(errors.New("cannot edit expression not owned by user"))
	}

	result, err := params.Svcs.Expressions.UpdateExpression(
		strippedID,
		*req,
		params.UserInfo,
		existingExpr.Origin.CreatedUnixTimeSec,
		existingExpr.Origin.Shared,
		existingExpr.SourceCode,
	)

	resultItem := toWire(result)
	return resultItem, err
}

func isValidModuleReferences(refs []expressions.ModuleReference) error {
	// Make sure there are no duplicate modules and also that versions are parsable
	moduleIDs := map[string]bool{}
	for _, r := range refs {
		if _, ok := moduleIDs[r.ModuleID]; ok {
			return fmt.Errorf("Duplicate modules: %v", r.ModuleID)
		}
		moduleIDs[r.ModuleID] = true

		_, err := modules.SemanticVersionFromString(r.Version)
		if err != nil {
			return fmt.Errorf("Invalid version for module: %v. Error was: %v", r.ModuleID, r.Version)
		}
	}
	return nil
}

func dataExpressionExecutionStatPut(params handlers.ApiHandlerParams) (interface{}, error) {
	itemID := params.PathParams[idIdentifier]

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	var req expressions.DataExpressionExecStats
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	// Get the id, stripped of shared- otherwise we can't find it
	strippedID, _ := utils.StripSharedItemIDPrefix(itemID)

	// Set the time stamp to now
	req.TimeStampUnixSec = params.Svcs.TimeStamper.GetTimeNowSec()

	return req, params.Svcs.Expressions.StoreExpressionRecentRunStats(strippedID, req)
}

func dataExpressionDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	itemID := params.PathParams[idIdentifier]

	// Read to make sure it exists and we have the permissions to delete it
	existingExpr, err := params.Svcs.Expressions.GetExpression(itemID, false)
	if err != nil {
		return nil, api.MakeNotFoundError(itemID)
	}

	if existingExpr.Origin.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", itemID, params.UserInfo.UserID))
	}

	// If deleting a shared item, we need to strip the prefix from the ID and also ensure that only the creator can delete
	id, _ := utils.StripSharedItemIDPrefix(itemID)
	err = params.Svcs.Expressions.DeleteExpression(id)
	if err != nil {
		return nil, err
	}

	// Return just the one deleted id
	return itemID, nil
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

func publishDataExpressionToZenodo(params handlers.ApiHandlerParams) (interface{}, error) {
	expressionID := params.PathParams[idIdentifier]

	// Get the uploaded zip data
	zipData, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	zenodoURI := params.Svcs.Config.ZenodoURI
	zenodoToken := params.Svcs.Config.ZenodoAccessToken

	expression, err := params.Svcs.Expressions.PublishExpressionToZenodo(expressionID, zipData, zenodoURI, zenodoToken)
	if err != nil {
		return nil, err
	}

	return expression, nil
}

func shareExpressions(svcs *services.APIServices, userID string, expressionIDs []string) ([]string, error) {
	generatedIDs := []string{}

	// Loop through & load each one
	for _, exprId := range expressionIDs {
		expr, err := svcs.Expressions.GetExpression(exprId, false)
		if err != nil {
			if svcs.Expressions.IsNotFoundError(err) {
				return generatedIDs, api.MakeNotFoundError(exprId)
			}
			return generatedIDs, err
		}

		// Make sure it isn't already shared
		if expr.Origin.Shared {
			return generatedIDs, api.MakeStatusError(http.StatusBadRequest, fmt.Errorf("%v already shared", exprId))
		}

		// Check that user has rights to do this
		if expr.Origin.Creator.UserID != userID {
			return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", exprId, userID))
		}

		// Sharing is an act of saving the same expression, with a different ID, and the share flag set
		sharedExpr, err := svcs.Expressions.CreateExpression(
			expressions.DataExpressionInput{
				Name:             expr.Name,
				SourceCode:       expr.SourceCode,
				SourceLanguage:   expr.SourceLanguage,
				Comments:         expr.Comments,
				Tags:             expr.Tags,
				ModuleReferences: expr.ModuleReferences,
			},
			expr.Origin.Creator,
			true,
		)

		if err != nil {
			return generatedIDs, err
		}

		generatedIDs = append(generatedIDs, sharedExpr.ID)
	}

	return generatedIDs, nil
}
