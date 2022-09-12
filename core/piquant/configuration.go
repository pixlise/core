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

package piquant

import (
	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/api/services"
	"github.com/pixlise/core/v2/core/api"
)

type PiquantConfig struct {
	Description         string `json:"description"`
	ConfigFile          string `json:"configFile"`
	OpticEfficiencyFile string `json:"opticEfficiencyFile"`
	CalibrationFile     string `json:"calibrationFile"`
	StandardsFile       string `json:"standardsFile"`
}

// Legacy storage in S3, due to - in field names
type piquantConfigS3 struct {
	Description         string `json:"description"`
	ConfigFile          string `json:"config-file"`
	OpticEfficiencyFile string `json:"optic-efficiency"`
	CalibrationFile     string `json:"calibration-file"`
	StandardsFile       string `json:"standards-file"`
}

func GetPIQUANTConfig(svcs *services.APIServices, configName string, version string) (PiquantConfig, error) {
	result := PiquantConfig{}

	cfg := piquantConfigS3{} // Note using the S3 version of the struct due to legacy dashed JSON var names
	s3Path := filepaths.GetDetectorConfigPath(configName, version, filepaths.PiquantConfigFileName)
	err := svcs.FS.ReadJSON(svcs.Config.ConfigBucket, s3Path, &cfg, false)
	if err != nil && svcs.FS.IsNotFoundError(err) {
		return result, api.MakeNotFoundError(configName)
	}

	// Return the result, converted to the "resulting" struct
	result = PiquantConfig{
		Description:         cfg.Description,
		ConfigFile:          cfg.ConfigFile,
		OpticEfficiencyFile: cfg.OpticEfficiencyFile,
		CalibrationFile:     cfg.CalibrationFile,
		StandardsFile:       cfg.StandardsFile,
	}

	return result, nil
}
