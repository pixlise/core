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

	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/expressions/modules"
	mongoDBConnection "github.com/pixlise/core/v2/core/mongo"
	"github.com/pixlise/core/v2/core/pixlUser"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MakeExpressionDB(
	envName string,
	svcs *services.APIServices,
) *ExpressionDB {
	exprDB := mongoDBConnection.GetDatabaseName("expressions", envName)

	db := svcs.Mongo.Database(exprDB)

	expressions := db.Collection("expressions")
	modules := db.Collection("modules")
	moduleVersions := db.Collection("moduleVersions")

	return &ExpressionDB{
		Svcs: svcs,

		Database:       db,
		Expressions:    expressions,
		Modules:        modules,
		ModuleVersions: moduleVersions,
	}
}

type ExpressionDB struct {
	Svcs *services.APIServices

	Database       *mongo.Database
	Expressions    *mongo.Collection
	Modules        *mongo.Collection
	ModuleVersions *mongo.Collection
}

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
func (e *ExpressionDB) ListModules() (modules.DataModuleWireLookup, error) {
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
				Version:          ver.Version,
				Tags:             ver.Tags,
				Comments:         ver.Comments,
				TimeStampUnixSec: ver.TimeStampUnixSec,
			})
		}

		// Deep copy the module, otherwise we end up overwriting pointers...
		var modCopy modules.DataModule = moduleItem
		result[moduleItem.ID] = modules.DataModuleWire{
			DataModule: &modCopy,
			Versions:   wireVersions,
		}
	}

	return result, nil
}

func (e *ExpressionDB) getModule(moduleID string) (modules.DataModule, error) {
	filter := bson.D{primitive.E{Key: "moduleID", Value: moduleID}}

	opts := options.FindOne()
	cursor := e.Modules.FindOne(context.TODO(), filter, opts)

	mod := modules.DataModule{}
	err := cursor.Decode(&mod)

	return mod, err
}

func (e *ExpressionDB) getModuleVersion(moduleID string, version string) (modules.DataModuleVersion, error) {
	filter := bson.D{primitive.E{Key: "moduleID", Value: moduleID}, primitive.E{Key: "version", Value: version}}

	opts := options.FindOne()
	cursor := e.ModuleVersions.FindOne(context.TODO(), filter, opts)

	ver := modules.DataModuleVersion{}
	err := cursor.Decode(&ver)

	return ver, err
}

func (e *ExpressionDB) GetModule(moduleID string, version string) (modules.DataModuleSpecificVersionWire, error) {
	if e.Modules == nil {
		return modules.DataModuleSpecificVersionWire{}, errors.New("GetModule: Mongo not connected")
	}

	mod, err := e.getModule(moduleID)

	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}

	ver, err := e.getModuleVersion(moduleID, version)

	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, fmt.Errorf("Failed to get version: %v for module: %v. Error: %v", version, moduleID, err)
	}

	result := modules.DataModuleSpecificVersionWire{
		DataModule: &mod,
		Version: modules.DataModuleVersionSourceWire{
			SourceCode: ver.SourceCode,
			DataModuleVersionWire: &modules.DataModuleVersionWire{
				Version:          ver.Version,
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
	_, err := e.Modules.InsertOne(context.TODO(), mod)
	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}

	// Write out the first version
	ver := modules.DataModuleVersion{
		ModuleID:         modId,
		SourceCode:       input.SourceCode,
		Version:          "0.1",
		Tags:             input.Tags,
		Comments:         "Initial version",
		TimeStampUnixSec: nowUnix,
	}

	_, err = e.ModuleVersions.InsertOne(context.TODO(), ver)
	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}

	// We return it differently
	verWire := modules.DataModuleVersionSourceWire{
		SourceCode: input.SourceCode,
		DataModuleVersionWire: &modules.DataModuleVersionWire{
			Version:          ver.Version,
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

func (e *ExpressionDB) AddModuleVersion(moduleID string, input modules.DataModuleInput) (modules.DataModuleSpecificVersionWire, error) {
	if e.Modules == nil {
		return modules.DataModuleSpecificVersionWire{}, errors.New("AddModuleVersion: Mongo not connected")
	}

	// Check that the module exists
	_, err := e.getModule(moduleID)

	if err != nil {
		return modules.DataModuleSpecificVersionWire{}, err
	}
	/*
		// Get latest version so we can increment
		ver, err := e.getModuleVersion(moduleID, version)

		if err == nil {
			return modules.DataModuleSpecificVersionWire{}, fmt.Errorf("Module %v version %v already exists", moduleID, version)
		}

		filter := bson.D{primitive.E{Key: "moduleID", Value: moduleID}}

		opts := options.FindOne()
		cursor := e.Modules.FindOne(context.TODO(), filter, opts)

		mod := modules.DataModule{}
		err := cursor.Decode(&mod)

		if err != nil {
			return modules.DataModuleSpecificVersionWire{}, err
		}

		// It does exist, check that this version doesn't yet exist
	*/
	return modules.DataModuleSpecificVersionWire{}, nil
}
