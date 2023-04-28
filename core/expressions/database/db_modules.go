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

package expressionDB

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v3/core/expressions/modules"
	"github.com/pixlise/core/v3/core/expressions/zenodo"
	"github.com/pixlise/core/v3/core/pixlUser"
	"github.com/pixlise/core/v3/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func (e *ExpressionDB) getModuleVersions(moduleID string) ([]modules.DataModuleVersion, error) {
	allVersions := []modules.DataModuleVersion{}

	filter := bson.D{primitive.E{Key: "moduleid", Value: moduleID}}

	opts := options.Find()
	cursor, err := e.ModuleVersions.Find(context.TODO(), filter, opts)
	if err != nil {
		return allVersions, err
	}

	err = cursor.All(context.TODO(), &allVersions)
	return allVersions, err
}

// ListModules - Lists all modules, returning a map of Module ID->Module, which contains a list of all
// module versions (with their tags). Note, this does
func (e *ExpressionDB) ListModules(retrieveUpdatedUserInfo bool) (modules.DataModuleWireLookup, error) {
	result := modules.DataModuleWireLookup{}
	if e.Modules == nil {
		return result, errors.New("ListModules: Mongo not connected")
	}

	// List all of the modules
	filter := bson.D{}
	opts := options.Find()
	cursor, err := e.Modules.Find(context.TODO(), filter, opts)

	if err != nil {
		return result, err
	}

	allModules := []modules.DataModule{}
	err = cursor.All(context.TODO(), &allModules)
	if err != nil {
		return result, err
	}

	// And for each module, we list all versions. Note, that we're returning a map of modules by module ID
	for _, moduleItem := range allModules {
		versions, err := e.getModuleVersions(moduleItem.ID)

		if err != nil {
			return result, fmt.Errorf("Failed to query versions for module %v. Error: %v", moduleItem.ID, err)
		}

		// If we didn't get any versions returned, this is an error!
		if len(versions) <= 0 {
			return result, fmt.Errorf("No versions for module %v", moduleItem.ID)
		}

		wireVersions := []modules.DataModuleVersionWire{}

		for _, ver := range versions {
			wireVersions = append(wireVersions, modules.DataModuleVersionWire{
				Version:          modules.SemanticVersionToString(ver.Version),
				Tags:             ver.Tags,
				Comments:         ver.Comments,
				TimeStampUnixSec: ver.TimeStampUnixSec,
			})
		}

		// Deep copy the module, otherwise we end up overwriting pointers...
		var modCopy modules.DataModule = moduleItem

		if retrieveUpdatedUserInfo {
			updatedCreator, creatorErr := e.Svcs.Users.GetCurrentCreatorDetails(modCopy.Origin.Creator.UserID)
			if creatorErr != nil {
				e.Svcs.Log.Infof("Failed to lookup user details for ID: %v (Name: %v), creator name in module: %v. Error: %v", modCopy.Origin.Creator.UserID, modCopy.Origin.Creator.Name, moduleItem.ID, creatorErr)
			} else {
				modCopy.Origin.Creator = updatedCreator
			}
		}

		result[moduleItem.ID] = modules.DataModuleWire{
			DataModule: &modCopy,
			Versions:   wireVersions,
		}
	}

	return result, nil
}

func (e *ExpressionDB) getModule(moduleID string, retrieveUpdatedUserInfo bool) (modules.DataModule, error) {
	result := modules.DataModule{}
	modResult := e.Modules.FindOne(context.TODO(), bson.M{"_id": moduleID})

	if modResult.Err() != nil {
		return result, modResult.Err()
	}

	// Read the module item
	err := modResult.Decode(&result)
	if err != nil {
		return result, err
	}

	if retrieveUpdatedUserInfo {
		// Get latest user details from Mongo
		updatedCreator, creatorErr := e.Svcs.Users.GetCurrentCreatorDetails(result.Origin.Creator.UserID)
		if creatorErr != nil {
			e.Svcs.Log.Infof("Failed to lookup user details for ID: %v (Name: %v), creator name in expression: %v. Error: %v", result.Origin.Creator.UserID, result.Origin.Creator.Name, result.ID, creatorErr)
		} else {
			result.Origin.Creator = updatedCreator
		}
	}

	return result, err
}

func (e *ExpressionDB) getModuleVersion(moduleID string, version modules.SemanticVersion) (modules.DataModuleVersion, error) {
	// NOTE: This was initially built with a query:
	// filter := bson.D{primitive.E{Key: "moduleid", Value: moduleID}, primitive.E{Key: "version", Value: version}}
	// But now ID is composed of these fields so it's more direct to query by ID
	result := modules.DataModuleVersion{}
	id := moduleID + "-v" + modules.SemanticVersionToString(version)
	verResult := e.ModuleVersions.FindOne(context.TODO(), bson.M{"_id": id})

	if verResult.Err() != nil {
		return result, verResult.Err()
	}

	// Read the module item
	err := verResult.Decode(&result)
	return result, err
}

func (e *ExpressionDB) GetModule(moduleID string, version *modules.SemanticVersion, retrieveUpdatedUserInfo bool) (modules.DataModuleSpecificVersionWire, error) {
	if e.Modules == nil {
		return modules.DataModuleSpecificVersionWire{}, errors.New("GetModule: Mongo not connected")
	}

	mod, err := e.getModule(moduleID, retrieveUpdatedUserInfo)

	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}

	// if version is not supplied, get latest
	if version == nil {
		ver, err := e.getLatestVersion(moduleID)
		if err != nil {
			return modules.DataModuleSpecificVersionWire{}, err
		}

		// Query this one!
		version = &ver
	}

	ver, err := e.getModuleVersion(moduleID, *version)

	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, fmt.Errorf("Failed to get version: %v for module: %v. Error: %v", modules.SemanticVersionToString(*version), moduleID, err)
	}

	result := modules.DataModuleSpecificVersionWire{
		DataModule: &mod,
		Version: modules.DataModuleVersionSourceWire{
			SourceCode: ver.SourceCode,
			DataModuleVersionWire: &modules.DataModuleVersionWire{
				Version:          modules.SemanticVersionToString(ver.Version),
				Tags:             ver.Tags,
				Comments:         ver.Comments,
				TimeStampUnixSec: ver.TimeStampUnixSec,
			},
		},
	}

	return result, err
}

func (e *ExpressionDB) CreateModule(
	input modules.DataModuleInput,
	creator pixlUser.UserInfo,
) (modules.DataModuleSpecificVersionWire, error) {
	nowUnix := e.Svcs.TimeStamper.GetTimeNowSec()
	modId := e.Svcs.IDGen.GenObjectID()

	mod := modules.DataModule{
		ID:       modId,
		Name:     input.Name,
		Comments: input.Comments,
		Origin: pixlUser.APIObjectItem{
			Shared:              true,
			Creator:             creator,
			CreatedUnixTimeSec:  nowUnix,
			ModifiedUnixTimeSec: nowUnix,
		},
	}

	// Write the module itself
	insertResult, err := e.Modules.InsertOne(context.TODO(), mod)
	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}
	if insertResult.InsertedID != modId {
		e.Svcs.Log.Errorf("CreateModule (module): Expected Mongo insert to return ID %v, got %v", modId, insertResult.InsertedID)
	}

	// Write out the first version
	saveVer := modules.SemanticVersion{Major: 0, Minor: 0, Patch: 1}
	verId := modId + "-v" + modules.SemanticVersionToString(saveVer)
	ver := modules.DataModuleVersion{
		ID:               verId,
		ModuleID:         modId,
		SourceCode:       input.SourceCode,
		Version:          saveVer,
		Tags:             input.Tags,
		Comments:         "Initial version",
		TimeStampUnixSec: nowUnix,
	}

	insertResult, err = e.ModuleVersions.InsertOne(context.TODO(), ver)
	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}
	if insertResult.InsertedID != verId {
		e.Svcs.Log.Errorf("CreateModule (version): Expected Mongo insert to return ID %v, got %v", verId, insertResult.InsertedID)
	}

	// We return it differently
	verWire := modules.DataModuleVersionSourceWire{
		SourceCode: input.SourceCode,
		DataModuleVersionWire: &modules.DataModuleVersionWire{
			Version:          modules.SemanticVersionToString(ver.Version),
			Tags:             ver.Tags,
			Comments:         ver.Comments,
			TimeStampUnixSec: ver.TimeStampUnixSec,
		},
	}

	result := modules.DataModuleSpecificVersionWire{
		DataModule: &mod,
		Version:    verWire,
	}

	return result, err
}

func (e *ExpressionDB) getLatestVersion(moduleID string) (modules.SemanticVersion, error) {
	result := modules.SemanticVersion{}

	ctx := context.TODO()
	cursor, err := e.ModuleVersions.Aggregate(ctx, bson.A{
		bson.D{{"$match", bson.D{{"moduleid", moduleID}}}},
		bson.D{
			{"$sort",
				bson.D{
					{"version.major", -1},
					{"version.minor", -1},
					{"version.patch", -1},
				},
			},
		},
		bson.D{{"$limit", 1}},
		bson.D{{"$project", bson.D{{"version", 1}}}},
	})

	if err != nil {
		return result, err
	}

	defer cursor.Close(ctx)
	ver := modules.DataModuleVersion{}
	for cursor.Next(ctx) {
		err = cursor.Decode(&ver)
	}

	result = ver.Version
	//ver := bson.D{}
	//err = cursor.Decode(&ver)

	return result, err
}

func (e *ExpressionDB) AddModuleVersion(moduleID string, input modules.DataModuleVersionInput) (modules.DataModuleSpecificVersionWire, error) {
	if e.Modules == nil {
		return modules.DataModuleSpecificVersionWire{}, errors.New("AddModuleVersion: Mongo not connected")
	}

	// Check that the version update field is a valid value
	if !utils.StringInSlice(input.VersionUpdate, []string{"", "patch", "minor", "major"}) {
		return modules.DataModuleSpecificVersionWire{}, fmt.Errorf("Invalid version update field: %v", input.VersionUpdate)
	}

	// Check that the module exists
	mod, err := e.getModule(moduleID, false)

	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, fmt.Errorf("Failed to add new version to non-existant module %v. %v", moduleID, err)
	}

	ver, err := e.getLatestVersion(moduleID)
	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}

	// Increment the version as needed
	if input.VersionUpdate == "major" {
		ver.Major++
		ver.Minor = 0
		ver.Patch = 0
	} else if input.VersionUpdate == "minor" {
		ver.Minor++
		ver.Patch = 0
	} else {
		ver.Patch++
	}

	// Write out the new version
	verId := moduleID + "-v" + modules.SemanticVersionToString(ver)
	nowUnix := e.Svcs.TimeStamper.GetTimeNowSec()
	verRec := modules.DataModuleVersion{
		ID:               verId,
		ModuleID:         moduleID,
		SourceCode:       input.SourceCode,
		Version:          ver,
		Tags:             input.Tags,
		Comments:         input.Comments,
		TimeStampUnixSec: nowUnix,
	}

	insertResult, err := e.ModuleVersions.InsertOne(context.TODO(), verRec)
	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}
	if insertResult.InsertedID != verId {
		e.Svcs.Log.Errorf("CreateModule (version): Expected Mongo insert to return ID %v, got %v", verId, insertResult.InsertedID)
	}

	// We return it differently
	verWire := modules.DataModuleVersionSourceWire{
		SourceCode: input.SourceCode,
		DataModuleVersionWire: &modules.DataModuleVersionWire{
			Version:          modules.SemanticVersionToString(verRec.Version),
			Tags:             verRec.Tags,
			Comments:         verRec.Comments,
			TimeStampUnixSec: verRec.TimeStampUnixSec,
		},
	}

	result := modules.DataModuleSpecificVersionWire{
		DataModule: &mod,
		Version:    verWire,
	}

	deposition, err := zenodo.PublishModuleToZenodo(result)
	if err != nil {
		e.Svcs.Log.Errorf("Failed to release Zenodo update for module: %v. Error: %v", moduleID, err)
		return result, err
	}

	fmt.Println("Deposition: ", deposition)

	return result, nil
}
