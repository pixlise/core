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
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/pixlUser"
	"github.com/pixlise/core/v2/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// Spectrum Chart Annotations

const roiIdentifier = "roi"

type annotationHandler struct {
	svcs *services.APIServices
}

type spectrumAnnotationLineInput struct {
	Name  string  `json:"name"`
	RoiID string  `json:"roiID"`
	EV    float32 `json:"eV"`
}

type spectrumAnnotationLine struct {
	Name  string  `json:"name"`
	RoiID string  `json:"roiID"`
	EV    float32 `json:"eV"`
	*pixlUser.APIObjectItem
}

type annotationLookup map[string]spectrumAnnotationLine

func registerAnnotationHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "annotation"

	// We can get all annotations for the current ROI, or post new ones:
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), annotationList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), annotationPost)

	// Getting individual line, replacing or deleting by id:
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), annotationGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), annotationPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), annotationDelete)

	// sharing
	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix, datasetIdentifier, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedAnnotation), annotationShare)
}

func readAnnotationData(svcs *services.APIServices, s3Path string) (annotationLookup, error) {
	itemLookup := annotationLookup{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &itemLookup, true)
	return itemLookup, err
}

func getAnnotationByID(svcs *services.APIServices, ID string, s3PathFrom string, markShared bool) (spectrumAnnotationLine, error) {
	result := spectrumAnnotationLine{}

	items, err := readAnnotationData(svcs, s3PathFrom)
	if err != nil {
		return result, err
	}

	// Find the named one and return it
	result, ok := items[ID]
	if !ok {
		return result, api.MakeNotFoundError(ID)
	}

	result.Shared = markShared

	return result, nil
}

func annotationList(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	// It's a get, we don't care about the body...

	// Get the annotatios for requesting user
	s3Path := filepaths.GetAnnotationsPath(params.UserInfo.UserID, datasetID)
	annotations, err := readAnnotationData(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	sharedS3Path := filepaths.GetAnnotationsPath(pixlUser.ShareUserID, datasetID)
	sharedAnnotations, err := readAnnotationData(params.Svcs, sharedS3Path)
	if err != nil {
		return nil, err
	}

	// Copy shared stuff into list we're returning
	for id, item := range sharedAnnotations {
		item.Shared = true
		annotations[utils.SharedItemIDPrefix+id] = item
	}

	// Read keys in alphabetical order, else we randomly fail unit test
	keys := []string{}
	for k := range annotations {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Update all creator infos
	for _, k := range keys {
		item := annotations[k]
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(item.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Errorf("Failed to lookup user details for ID: %v, creator name in file: %v (Spectrum annotation listing). Error: %v", item.Creator.UserID, item.Creator.Name, creatorErr)
		} else {
			item.Creator = updatedCreator
		}
	}

	return &annotations, nil
}

func annotationGet(params handlers.ApiHandlerParams) (interface{}, error) {
	// Depending on if the user ID starts with our shared marker, we load from different files...
	itemID := params.PathParams[idIdentifier]
	datasetID := params.PathParams[datasetIdentifier]

	// Check if it's a shared one, if so, change our query variables
	s3Path := filepaths.GetAnnotationsPath(params.UserInfo.UserID, datasetID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetAnnotationsPath(pixlUser.ShareUserID, datasetID)
		itemID = strippedID
	}

	line, err := getAnnotationByID(params.Svcs, itemID, s3Path, isSharedReq)

	if err == nil {
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(line.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Errorf("Failed to lookup user details for ID: %v, creator name in file: %v (Spectrum annotation GET). Error: %v", line.Creator.UserID, line.Creator.Name, creatorErr)
		} else {
			line.Creator = updatedCreator
		}
	}

	return line, err
}

func setupAnnotationForSave(svcs *services.APIServices, body []byte, s3Path string) (*annotationLookup, *spectrumAnnotationLineInput, error) {
	// Read in body
	var req spectrumAnnotationLineInput
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, nil, api.MakeBadRequestError(err)
	}

	// Validate
	if !fileaccess.IsValidObjectName(req.Name) {
		return nil, nil, api.MakeBadRequestError(fmt.Errorf("Invalid name: %v", req.Name))
	}
	if req.EV <= 0 {
		return nil, nil, api.MakeBadRequestError(fmt.Errorf("Invalid eV: %v", req.EV))
	}
	// Download the file
	annotations, err := readAnnotationData(svcs, s3Path)
	if err != nil && !svcs.FS.IsNotFoundError(err) {
		return nil, nil, err
	}

	return &annotations, &req, nil
}

func annotationPost(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	s3Path := filepaths.GetAnnotationsPath(params.UserInfo.UserID, datasetID)
	annotationFile, req, err := setupAnnotationForSave(params.Svcs, body, s3Path)
	if err != nil {
		return nil, err
	}

	itemID := params.Svcs.IDGen.GenObjectID()
	_, ok := (*annotationFile)[itemID]
	if ok {
		return nil, fmt.Errorf("Failed to generate unique ID")
	}

	// Save it & upload
	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()
	saveItem := spectrumAnnotationLine{
		req.Name,
		req.RoiID,
		req.EV,
		&pixlUser.APIObjectItem{
			Shared:              false,
			Creator:             params.UserInfo,
			CreatedUnixTimeSec:  timeNow,
			ModifiedUnixTimeSec: timeNow,
		},
	}
	(*annotationFile)[itemID] = saveItem
	return *annotationFile, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *annotationFile)
}

func annotationPut(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	s3Path := filepaths.GetAnnotationsPath(params.UserInfo.UserID, datasetID)
	annotationFile, req, err := setupAnnotationForSave(params.Svcs, body, s3Path)
	if err != nil {
		return nil, err
	}

	itemID := params.PathParams[idIdentifier]
	existing, ok := (*annotationFile)[itemID]
	if !ok {
		return nil, api.MakeNotFoundError(itemID)
	}

	// Save it & upload
	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()
	saveItem := spectrumAnnotationLine{
		req.Name,
		req.RoiID,
		req.EV,
		&pixlUser.APIObjectItem{
			Shared:              false,
			Creator:             existing.Creator,
			CreatedUnixTimeSec:  existing.CreatedUnixTimeSec,
			ModifiedUnixTimeSec: timeNow,
		},
	}
	(*annotationFile)[itemID] = saveItem

	return *annotationFile, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *annotationFile)
}

func annotationDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	// If deleting a shared item, we need to strip the prefix from the ID and also ensure that only the creator can delete
	itemID := params.PathParams[idIdentifier]
	datasetID := params.PathParams[datasetIdentifier]

	s3Path := filepaths.GetAnnotationsPath(params.UserInfo.UserID, datasetID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetAnnotationsPath(pixlUser.ShareUserID, datasetID)
		itemID = strippedID
	}

	// Using path params, work out path
	items, err := readAnnotationData(params.Svcs, s3Path)
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

func annotationShare(params handlers.ApiHandlerParams) (interface{}, error) {
	// User is supplying an ID of an object to share. We should be able to find it in the users own data file
	// and put it in the shared file with a new ID, thereby implementing "share a copy"
	idToFind := params.PathParams[idIdentifier]
	datasetID := params.PathParams[datasetIdentifier]

	s3Path := filepaths.GetAnnotationsPath(params.UserInfo.UserID, datasetID)
	itemToShare, err := getAnnotationByID(params.Svcs, idToFind, s3Path, true)
	if err != nil {
		return nil, err
	}

	// We've found it, download the shared file, so we can add it
	sharedS3Path := filepaths.GetAnnotationsPath(pixlUser.ShareUserID, datasetID)
	sharedItems, err := readAnnotationData(params.Svcs, sharedS3Path)
	if err != nil {
		return nil, err
	}

	// Add it to the shared file and we're done
	sharedID := params.Svcs.IDGen.GenObjectID()
	_, ok := sharedItems[sharedID]
	if ok {
		return nil, fmt.Errorf("Failed to generate unique share ID")
	}

	sharedItems[sharedID] = itemToShare
	return sharedID, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, sharedS3Path, sharedItems)
}
