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
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/api"
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
