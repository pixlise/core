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

package detector

import (
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/api"
)

// Stored config
type DetectorConfig struct {
	MinElementAtomicNumber    int8    `json:"minElement"`
	MaxElementAtomicNumber    int8    `json:"maxElement"`
	XRFeVLowerBound           int32   `json:"xrfeVLowerBound"`
	XRFeVUpperBound           int32   `json:"xrfeVUpperBound"`
	XRFeVResolution           int32   `json:"xrfeVResolution"`
	WindowElementAtomicNumber int8    `json:"windowElement"`
	TubeElementAtomicNumber   int8    `json:"tubeElement"`
	DefaultParams             string  `json:"defaultParams"`
	MMBeamRadius              float32 `json:"mmBeamRadius"`
}

// ReadDetectorConfig - Reads detector configuration given a name. Name is something
// like "PIXL" or "Breadboard" - one of the config subdirectories
func ReadDetectorConfig(svcs *services.APIServices, configName string) (DetectorConfig, error) {
	resp := DetectorConfig{}
	s3Path := filepaths.GetDetectorConfigFilePath(configName)
	err := svcs.FS.ReadJSON(svcs.Config.ConfigBucket, s3Path, &resp, false)

	if err != nil && svcs.FS.IsNotFoundError(err) {
		return resp, api.MakeNotFoundError(configName)
	}

	return resp, err
}
