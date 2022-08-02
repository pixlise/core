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
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/handlers"
	"github.com/pixlise/core/api/permission"
	apiRouter "github.com/pixlise/core/api/router"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
	"github.com/pixlise/core/core/fileaccess"
	"github.com/pixlise/core/core/pixlUser"
	"github.com/pixlise/core/core/utils"
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

	// Get the annotatios for reuesting  user
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

	return getAnnotationByID(params.Svcs, itemID, s3Path, isSharedReq)
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
	saveItem := spectrumAnnotationLine{req.Name, req.RoiID, req.EV, &pixlUser.APIObjectItem{Shared: false, Creator: params.UserInfo}}
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
	saveItem := spectrumAnnotationLine{req.Name, req.RoiID, req.EV, &pixlUser.APIObjectItem{Shared: false, Creator: existing.Creator}}
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
