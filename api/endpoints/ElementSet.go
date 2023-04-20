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
// Element Set

type elementLines struct {
	AtomicNumber int8 `json:"Z"` // 118 still fits! Will we break past 127 any time soon? :)
	K            bool `json:"K"`
	L            bool `json:"L"`
	M            bool `json:"M"`
	Esc          bool `json:"Esc"`
}

// ByAtomicNumber Atomic Numbering
type ByAtomicNumber []elementLines

func (a ByAtomicNumber) Len() int           { return len(a) }
func (a ByAtomicNumber) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByAtomicNumber) Less(i, j int) bool { return a[i].AtomicNumber < a[j].AtomicNumber }

type elementSetInput struct {
	Name  string         `json:"name"`
	Lines []elementLines `json:"lines"`
}

type ElementSet struct {
	Name  string         `json:"name"`
	Lines []elementLines `json:"lines"`
	*pixlUser.APIObjectItem
}

func (a ElementSet) SetTimes(userID string, t int64) {
	if a.CreatedUnixTimeSec == 0 {
		a.CreatedUnixTimeSec = t
	}
	if a.ModifiedUnixTimeSec == 0 {
		a.ModifiedUnixTimeSec = t
	}
}

type ElementSetLookup map[string]ElementSet

type elementSetSummary struct {
	Name          string `json:"name"`
	AtomicNumbers []int8 `json:"atomicNumbers"`
	*pixlUser.APIObjectItem
}
type elementSetSummaryLookup map[string]elementSetSummary

func registerElementSetHandler(router *apiRouter.ApiObjectRouter) {
	const pathPrefix = "element-set"

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), elementSetList)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix), apiRouter.MakeMethodPermission("POST", permission.PermWriteDataAnalysis), elementSetPost)

	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("GET", permission.PermReadDataAnalysis), elementSetGet)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("PUT", permission.PermWriteDataAnalysis), elementSetPut)
	router.AddJSONHandler(handlers.MakeEndpointPath(pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("DELETE", permission.PermWriteDataAnalysis), elementSetDelete)

	router.AddShareHandler(handlers.MakeEndpointPath(shareURLRoot+"/"+pathPrefix, idIdentifier), apiRouter.MakeMethodPermission("POST", permission.PermWriteSharedElementSet), elementSetShare)
}

func readElementSetData(svcs *services.APIServices, s3Path string) (ElementSetLookup, error) {
	itemLookup := ElementSetLookup{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, s3Path, &itemLookup, true)
	return itemLookup, err
}

func getElementSetByID(svcs *services.APIServices, ID string, s3PathFrom string, markShared bool) (ElementSet, error) {
	result := ElementSet{}

	elemSets, err := readElementSetData(svcs, s3PathFrom)
	if err != nil {
		return result, err
	}

	// Find the named one and return it
	result, ok := elemSets[ID]
	if !ok {
		return result, api.MakeNotFoundError(ID)
	}

	result.Shared = markShared

	return result, nil
}

func getElementSetSummary(svcs *services.APIServices, s3PathFrom string, sharedFile bool, outMap *elementSetSummaryLookup) error {
	elemSets, err := readElementSetData(svcs, s3PathFrom)
	if err != nil {
		return err
	}

	// Run through and just return summary info
	for id, item := range elemSets {
		// Loop through all elements and make an element set summary
		summary := elementSetSummary{
			Name:          item.Name,
			AtomicNumbers: []int8{},
			APIObjectItem: &pixlUser.APIObjectItem{
				Shared:              sharedFile,
				Creator:             item.Creator,
				CreatedUnixTimeSec:  item.CreatedUnixTimeSec,
				ModifiedUnixTimeSec: item.ModifiedUnixTimeSec,
			},
		}
		for _, lineinfo := range item.Lines {
			summary.AtomicNumbers = append(summary.AtomicNumbers, lineinfo.AtomicNumber)
		}

		// We modify the ids of shared items, so if passed to GET/PUT/DELETE we know this refers to something that's shared
		saveID := id
		if sharedFile {
			saveID = utils.SharedItemIDPrefix + id
		}

		(*outMap)[saveID] = summary
	}

	return nil
}

func elementSetList(params handlers.ApiHandlerParams) (interface{}, error) {
	summaryLookup := elementSetSummaryLookup{}

	// Get user item summaries
	err := getElementSetSummary(params.Svcs, filepaths.GetElementSetPath(params.UserInfo.UserID), false, &summaryLookup)
	if err != nil {
		return nil, err
	}

	// Get shared item summaries (into same map)
	err = getElementSetSummary(params.Svcs, filepaths.GetElementSetPath(pixlUser.ShareUserID), true, &summaryLookup)
	if err != nil {
		return nil, err
	}

	// Read keys in alphabetical order, else we randomly fail unit test
	keys := []string{}
	for k := range summaryLookup {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Update all creator infos
	for _, k := range keys {
		item := summaryLookup[k]
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(item.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Infof("Failed to lookup user details for ID: %v, creator name in file: %v (element set listing). Error: %v", item.Creator.UserID, item.Creator.Name, creatorErr)
		} else {
			item.Creator = updatedCreator
		}
	}

	// Return the combined set
	return &summaryLookup, nil
}

func elementSetGet(params handlers.ApiHandlerParams) (interface{}, error) {
	// Depending on if the user ID starts with our shared marker, we load from different files...
	itemID := params.PathParams[idIdentifier]

	// Check if it's a shared one, if so, change our query variables
	s3Path := filepaths.GetElementSetPath(params.UserInfo.UserID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetElementSetPath(pixlUser.ShareUserID)
		itemID = strippedID
	}

	elemSet, err := getElementSetByID(params.Svcs, itemID, s3Path, isSharedReq)

	if err == nil {
		updatedCreator, creatorErr := params.Svcs.Users.GetCurrentCreatorDetails(elemSet.Creator.UserID)
		if creatorErr != nil {
			params.Svcs.Log.Infof("Failed to lookup user details for ID: %v, creator name in file: %v (element set GET). Error: %v", elemSet.Creator.UserID, elemSet.Creator.Name, creatorErr)
		} else {
			elemSet.Creator = updatedCreator
		}
	}

	return elemSet, err
}

func setupElementSetForSave(svcs *services.APIServices, body []byte, s3Path string) (*ElementSetLookup, *elementSetInput, error) {
	// Read in body
	var req elementSetInput
	err := json.Unmarshal(body, &req)
	if err != nil {
		return nil, nil, api.MakeBadRequestError(err)
	}

	// Validate
	if !fileaccess.IsValidObjectName(req.Name) {
		return nil, nil, api.MakeBadRequestError(fmt.Errorf("Invalid element set name: %v", req.Name))
	}
	if len(req.Lines) <= 0 {
		return nil, nil, api.MakeBadRequestError(fmt.Errorf("Cannot save empty element set"))
	}
	for _, item := range req.Lines {
		if item.AtomicNumber < 1 || item.AtomicNumber > 118 {
			return nil, nil, api.MakeBadRequestError(fmt.Errorf("Invalid atomic number: %v", item.AtomicNumber))
		}
		if item.K == false && item.L == false && item.M == false {
			return nil, nil, api.MakeBadRequestError(fmt.Errorf("At least one shell should be enabled"))
		}
	}

	// Save the atomic numbers sorted
	sort.Sort(ByAtomicNumber(req.Lines))

	// Download the file
	elemSets, err := readElementSetData(svcs, s3Path)
	if err != nil {
		return nil, nil, err
	}

	return &elemSets, &req, nil
}

func elementSetPost(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	s3Path := filepaths.GetElementSetPath(params.UserInfo.UserID)
	elemSets, req, err := setupElementSetForSave(params.Svcs, body, s3Path)
	if err != nil {
		return nil, err
	}

	itemID := params.Svcs.IDGen.GenObjectID()
	_, ok := (*elemSets)[itemID]
	if ok {
		return nil, fmt.Errorf("Failed to generate unique ID")
	}

	// Save it & upload
	timeNow := params.Svcs.TimeStamper.GetTimeNowSec()
	(*elemSets)[itemID] = ElementSet{
		req.Name,
		req.Lines,
		&pixlUser.APIObjectItem{
			Shared:              false,
			Creator:             params.UserInfo,
			CreatedUnixTimeSec:  timeNow,
			ModifiedUnixTimeSec: timeNow,
		},
	}
	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *elemSets)
}

func elementSetPut(params handlers.ApiHandlerParams) (interface{}, error) {
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	itemID := params.PathParams[idIdentifier]
	if _, isSharedReq := utils.StripSharedItemIDPrefix(itemID); isSharedReq {
		return nil, api.MakeBadRequestError(errors.New("Cannot edit shared items"))
	}

	s3Path := filepaths.GetElementSetPath(params.UserInfo.UserID)
	elemSets, req, err := setupElementSetForSave(params.Svcs, body, s3Path)
	if err != nil {
		return nil, err
	}

	existing, ok := (*elemSets)[itemID]
	if !ok {
		return nil, api.MakeNotFoundError(itemID)
	}

	// Save it & upload
	(*elemSets)[itemID] = ElementSet{
		req.Name,
		req.Lines,
		&pixlUser.APIObjectItem{
			Shared:              false,
			Creator:             existing.Creator,
			CreatedUnixTimeSec:  existing.CreatedUnixTimeSec,
			ModifiedUnixTimeSec: params.Svcs.TimeStamper.GetTimeNowSec(),
		},
	}
	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, *elemSets)
}

func elementSetDelete(params handlers.ApiHandlerParams) (interface{}, error) {
	// If deleting a shared item, we need to strip the prefix from the ID and also ensure that only the creator can delete
	itemID := params.PathParams[idIdentifier]
	s3Path := filepaths.GetElementSetPath(params.UserInfo.UserID)

	strippedID, isSharedReq := utils.StripSharedItemIDPrefix(itemID)
	if isSharedReq {
		s3Path = filepaths.GetElementSetPath(pixlUser.ShareUserID)
		itemID = strippedID
	}

	// Using path params, work out path
	elemSets, err := readElementSetData(params.Svcs, s3Path)
	if err != nil {
		return nil, err
	}

	sharedItem, ok := elemSets[itemID]
	if !ok {
		return nil, api.MakeNotFoundError(itemID)
	}

	if isSharedReq && sharedItem.Creator.UserID != params.UserInfo.UserID {
		return nil, api.MakeStatusError(http.StatusUnauthorized, fmt.Errorf("%v not owned by %v", itemID, params.UserInfo.UserID))
	}

	// Found it, delete & we're done
	delete(elemSets, itemID)

	return nil, params.Svcs.FS.WriteJSON(params.Svcs.Config.UsersBucket, s3Path, elemSets)
}

func elementSetShare(params handlers.ApiHandlerParams) (interface{}, error) {
	// User is supplying an ID of an object to share. We should be able to find it in the users own data file
	// and put it in the shared file with a new ID, thereby implementing "share a copy"
	idToFind := params.PathParams[idIdentifier]

	s3Path := filepaths.GetElementSetPath(params.UserInfo.UserID)
	itemToShare, err := getElementSetByID(params.Svcs, idToFind, s3Path, true)
	if err != nil {
		return nil, err
	}

	// We've found it, download the shared file, so we can add it
	sharedS3Path := filepaths.GetElementSetPath(pixlUser.ShareUserID)
	sharedItems, err := readElementSetData(params.Svcs, sharedS3Path)
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
