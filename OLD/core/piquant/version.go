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
	"errors"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/pixlUser"
)

// Version config item
type PiquantVersionConfig struct {
	Version            string            `json:"version"`
	ChangedUnixTimeSec int64             `json:"changedUnixTimeSec"`
	Creator            pixlUser.UserInfo `json:"creator"`
}

// GetPiquantVersion - retrieves currently active PIQUANT version
// NOTE: If this is not in S3, we read the API config value, but this allows users to override it with S3
func GetPiquantVersion(svcs *services.APIServices) (PiquantVersionConfig, error) {
	ver := PiquantVersionConfig{}
	err := svcs.FS.ReadJSON(svcs.Config.ConfigBucket, filepaths.GetConfigFilePath(filepaths.PiquantVersionFileName), &ver, false)

	if err != nil {
		// Return the config var, if it's set
		if len(svcs.Config.PiquantDockerImage) <= 0 {
			return ver, errors.New("PIQUANT version not set")
		}

		ver = PiquantVersionConfig{
			Version:            svcs.Config.PiquantDockerImage,
			ChangedUnixTimeSec: 0,
			Creator:            pixlUser.UserInfo{},
		}
	}

	return ver, nil
}
