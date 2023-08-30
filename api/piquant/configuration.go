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

// Storage/versioning and retrieval of PIQUANT configuration files and the currently selected PIQUANT pod version to be run
package piquant

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

// Legacy storage in S3, due to - in field names
type piquantConfigS3 struct {
	Description         string `json:"description"`
	ConfigFile          string `json:"config-file"`
	OpticEfficiencyFile string `json:"optic-efficiency"`
	CalibrationFile     string `json:"calibration-file"`
	StandardsFile       string `json:"standards-file"`
}

func GetPIQUANTConfig(svcs *services.APIServices, configName string, version string) (*protos.PiquantConfig, error) {
	cfg := piquantConfigS3{} // Note using the S3 version of the struct due to legacy dashed JSON var names

	s3Path := filepaths.GetDetectorConfigPath(configName, version, filepaths.PiquantConfigFileName)
	err := svcs.FS.ReadJSON(svcs.Config.ConfigBucket, s3Path, &cfg, false)
	if err != nil && svcs.FS.IsNotFoundError(err) {
		return nil, errorwithstatus.MakeNotFoundError(configName)
	}

	// Return the result, converted to the "resulting" struct
	result := &protos.PiquantConfig{
		Description:         cfg.Description,
		ConfigFile:          cfg.ConfigFile,
		OpticEfficiencyFile: cfg.OpticEfficiencyFile,
		CalibrationFile:     cfg.CalibrationFile,
		StandardsFile:       cfg.StandardsFile,
	}

	return result, nil
}

func GetDetectorConfig(name string, db *mongo.Database) (*protos.DetectorConfig, error) {
	coll := db.Collection(dbCollections.DetectorConfigsName)

	result := coll.FindOne(context.TODO(), bson.M{"_id": name})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(name)
		}
		return nil, result.Err()
	}

	cfg := protos.DetectorConfig{}
	err := result.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &cfg, nil
}
