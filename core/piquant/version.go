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

package piquant

import (
	"errors"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/pixlUser"
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
