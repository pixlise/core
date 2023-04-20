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

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/api/permission"
	apiRouter "github.com/pixlise/core/v3/api/router"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/utils"
)

////////////////////////////////////////////////////////////////////////////////////////////////////////////
// RGB Mixes - storing/retrieving/sharing named mixes of elements on the one RGB map

const rgbMixIdPrefix = "rgbmix-"

// RGBMixInput - only public so we can use it embedded in dataExpression
type ChannelConfig struct {
	ExpressionID string  `json:"expressionID"`
	RangeMin     float32 `json:"rangeMin"`
	RangeMax     float32 `json:"rangeMax"`

	// We used to store this, now only here for reading in old files (backwards compatible). PIXLISE then converts it to an ExpressionID when saving again
	Element string `json:"element,omitempty"`
}

type RGBMixInput struct {
	Name  string        `json:"name"`
	Red   ChannelConfig `json:"red"`
	Green ChannelConfig `json:"green"`
	Blue  ChannelConfig `json:"blue"`
	Tags  []string      `json:"tags"`
}

type RGBMix struct {
	*RGBMixInput
	*pixlUser.APIObjectItem
}

func (a RGBMix) SetTimes(userID string, t int64) {
	if a.CreatedUnixTimeSec == 0 {
		a.CreatedUnixTimeSec = t
	}
	if a.ModifiedUnixTimeSec == 0 {
		a.ModifiedUnixTimeSec = t
	}
}

type RGBMixLookup map[string]RGBMix

func registerRGBMixHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "rgb-mix"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), rgbMixList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), rgbMixPost)

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), rgbMixPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), rgbMixDelete)

	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedExpression), rgbMixShare)
}

func readRGBMixData(svcs *services.APIServices, s3Path string) (RGBMixLookup, error) {
	itemLookup := RGBMixLookup{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &itemLookup, true)

	if err != nil {
		return itemLookup, err
	}

	// Convert any that only had an element defined to an expression ID. This is a backwards compatibility issue, we no longer store as "element"
	for _, v := range itemLookup {
		if len(v.Red.Element) > 0 && len(v.Red.ExpressionID) <= 0 {
			v.Red.ExpressionID = "expr-elem-" + v.Red.Element + "-%"
			v.Red.Element = ""
		}

		if len(v.Green.Element) > 0 && len(v.Green.ExpressionID) <= 0 {
			v.Green.ExpressionID = "expr-elem-" + v.Green.Element + "-%"
			v.Green.Element = ""
		}

		if len(v.Blue.Element) > 0 && len(v.Blue.ExpressionID) <= 0 {
			v.Blue.ExpressionID = "expr-elem-" + v.Blue.Element + "-%"
			v.Blue.Element = ""
		}
	}
	return itemLookup, nil
}

func getRGBMixByID(svcs *services.APIServices, ID string, s3PathFrom string, markShared bool) (RGBMix, error) {
	result := RGBMix{}

	items, err := readRGBMixData(svcs, s3PathFrom)
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

func getRGBMixListing(svcs *services.APIServices, s3PathFrom string, sharedFile bool, outMap *RGBMixLookup) error {
	items, err := readRGBMixData(svcs, s3PathFrom)
	if err != nil {
		return err
	}

	for id, item := range items {
		// We modify the ids of shared items, so if passed to GET/PUT/DELETE we know this refers to something that's shared
		saveID := id
		if sharedFile {
			saveID = utils.SharedItemIDPrefix + id
		}
		item.Shared = sharedFile

		(*outMap)[saveID] = item
	}

	return nil
}

func rgbMixList(params handlers.ApiHandlerParams) (interface{}, error) {
	items := RGBMixLookup{}

	// Get user item summaries
	err := getRGBMixListing(params.Svcs, filepaths.GetRGBMixPath(params.UserInfo.UserID), false, &items)
	if err != nil {
		return nil, err
	}

	// Get shared item summaries (into same map)
	err = getRGBMixListing(params.Svcs, filepaths.GetRGBMixPath(pixlUser.ShareUserID), true, &items)
	if err != nil {
		return nil, err
	}

	// Read keys in alphabetical order, else we randomly fail unit test
	keys := []string{}
	for k := range items {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		item := items[k]

		if item.Tags == nil {
			item.Tags = []string{}
		}

		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(item.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Infof("Failed to lookup user details for ID: %v, creator name in file: %v (RGB mix listing). Error: %v", item.Creator.UserID, item.Creator.Name, creatorErr)
		} else {
			item.Creator = updatedCreator
		}
	}

	// Return the combined set
	return &items, nil
}

func setupRGBMixForSave(params handlers.ApiHandlerParams, s3Path string) (*RGBMixLookup, *RGBMixInput, error) {
	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, nil, err
	}

	var req RGBMixInput
	err = json.Unmarshal(body, &req)
	if err != nil {
		return nil, nil, api.MakeBadRequestError(err)
	}

	// Validate
	if !fileaccess.IsValidObjectName(req.Name) {
		return nil, nil, api.MakeBadRequestError(fmt.Errorf("Invalid ID: %v", req.Name))
	}
	if len(req.Red.Element) > 0 || len(req.Green.Element) > 0 || len(req.Blue.Element) > 0 {
		return nil, nil, api.MakeBadRequestError(errors.New("RGB Mix definition with elements is deprecated"))
	}
	if len(req.Red.ExpressionID) <= 0 || len(req.Green.ExpressionID) <= 0 || len(req.Blue.ExpressionID) <= 0 {
		return nil, nil, api.MakeBadRequestError(errors.New("RGB Mix must have all expressions defined"))
	}

	if req.Tags == nil {
		req.Tags = []string{}
	}

	// Download the file
	items, err := readRGBMixData(params.Svcs, s3Path)
	if err != nil && !params.Svcs.FS.IsNotFoundError(err) {
		// Only return error if it's not about the file missing, because user may not have interacted with this dataset yet
		return nil, nil, err
	}

	return &items, &req, nil
}

func rgbMixPost(params handlers.ApiHandlerParams) (interface{}, error) {
	s3Path := filepaths.GetRGBMixPath(params.UserInfo.UserID)
	rgbMixes, req, err := setupRGBMixForSave(params, s3Path)
	if err != nil {
		return nil, err
	}

	// Save it & upload
	saveID := rgbMixIdPrefix + params.Svcs.IDGen.GenObjectID()
	_, exists := (*rgbMixes)[saveID]
	if exists {
		return nil, errors.New("Failed to generate unique ID")
	}

	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()
	(*rgbMixes)[saveID] = RGBMix{
		RGBMixInput: req,
		APIObjectItem: &pixlUser.APIObjectItem{
			Shared:              false,
			Creator:             params.UserInfo,
			CreatedUnixTimeSec:  timeNow,
			ModifiedUnixTimeSec: timeNow,
		},
	}

	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *rgbMixes)
}

func rgbMixPut(params handlers.ApiHandlerParams) (interface{}, error) {
	itemID := params.PathParams[idIdentifier]
	id, isSharedReq := utils.StripSharedItemIDPrefix(itemID)

	s3Path := filepaths.GetRGBMixPath(params.UserInfo.UserID)
	if isSharedReq {
		s3Path = filepaths.GetRGBMixPath(pixlUser.ShareUserID)
	}

	rgbMixes, req, err := setupRGBMixForSave(params, s3Path)
	if err != nil {
		return nil, err
	}

	existing, ok := (*rgbMixes)[id]
	if !ok {
		return nil, api.MakeNotFoundError(id)
	}

	if isSharedReq && params.UserInfo.UserID != existing.Creator.UserID {
		return nil, api.MakeBadRequestError(errors.New("cannot edit shared RGB mixes created by others"))
	}

	// Save it & upload
	(*rgbMixes)[id] = RGBMix{
		RGBMixInput: req,
		APIObjectItem: &pixlUser.APIObjectItem{
			Shared:              isSharedReq,
			Creator:             existing.Creator,
			CreatedUnixTimeSec:  existing.CreatedUnixTimeSec,
			ModifiedUnixTimeSec: params.Svcs.TimeStamper.GetTimeNowSec(),
		},
	}
	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *rgbMixes)
}

func rgbMixDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	// If deleting a shared item, we need to strip the prefix from the ID and also ensure that only the creator can delete
	itemID := params.PathParams[idIdentifier]
	s3Path := filepaths.GetRGBMixPath(params.UserInfo.UserID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetRGBMixPath(pixlUser.ShareUserID)
		itemID = strippedID
	}

	// Using path params, work out path
	items, err := readRGBMixData(params.Svcs, s3Path)
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

func rgbMixShare(params handlers.ApiHandlerParams) (interface{}, error) {
	// User is supplying an ID of an object to share. We should be able to find it in the users own data file
	// and put it in the shared file with a new ID, thereby implementing "share a copy"
	idToFind := params.PathParams[idIdentifier]
	ids, err := shareRGBMixes(params.Svcs, params.UserInfo.UserID, []string{idToFind})
	if err != nil {
		return nil, err
	}
	return ids[0], nil
}

func shareRGBMixes(svcs *services.APIServices, userID string, idsToShare []string) ([]string, error) {
	sharedIDs := []string{} // The shared IDs we will generate
	s3Path := filepaths.GetRGBMixPath(userID)
	userItems, err := readRGBMixData(svcs, s3Path)
	if err != nil {
		return sharedIDs, err
	}

	// NOTE: would make more sense to read the shared list here too but this would result in many tests
	// breaking as this was written with 1 id to share previously. This way works too we just need
	// to check for unique ids twice - once among newly generated ids, then among the shared ones
	// that get read in

	newlySharedItems := RGBMixLookup{}

	for _, idToFind := range idsToShare {
		itemToShare, ok := userItems[idToFind]

		if !ok {
			return []string{}, api.MakeNotFoundError(idToFind)
		}

		// Ensure that if it contains expressions, they are all SHARED expressions!
		// TODO: Remove this once we transition to a shared RGB mix containing the expression text baked into it
		if !strings.HasPrefix(itemToShare.Red.ExpressionID, "expr-") && !strings.HasPrefix(itemToShare.Red.ExpressionID, utils.SharedItemIDPrefix) ||
			!strings.HasPrefix(itemToShare.Green.ExpressionID, "expr-") && !strings.HasPrefix(itemToShare.Green.ExpressionID, utils.SharedItemIDPrefix) ||
			!strings.HasPrefix(itemToShare.Blue.ExpressionID, "expr-") && !strings.HasPrefix(itemToShare.Blue.ExpressionID, utils.SharedItemIDPrefix) {
			return nil, api.MakeBadRequestError(fmt.Errorf("When sharing RGB mix, it must only reference shared expressions"))
		}

		// Add it to the shared file and we're done
		sharedID := rgbMixIdPrefix + svcs.IDGen.GenObjectID()
		_, ok = newlySharedItems[sharedID]
		if ok {
			return nil, errors.New("Failed to generate unique share ID")
		}

		newlySharedItems[sharedID] = itemToShare
		sharedIDs = append(sharedIDs, sharedID)
	}

	// We've found it, download the shared file, so we can add it
	sharedS3Path := filepaths.GetRGBMixPath(pixlUser.ShareUserID)
	sharedItems, err := readRGBMixData(svcs, sharedS3Path)
	if err != nil {
		return nil, err
	}

	// Add the new items
	for id, item := range newlySharedItems {
		// Check id is unique against shared ones too
		_, ok := sharedItems[id]
		if ok {
			return nil, errors.New("Failed to generate unique share ID")
		}

		item.Shared = true

		// Set modified time, as we just shared it now
		item.ModifiedUnixTimeSec = svcs.TimeStamper.GetTimeNowSec()
		sharedItems[id] = item
	}

	return sharedIDs, svcs.FS.WriteJSON(svcs.Config.UsersBucket, sharedS3Path, sharedItems)
}
