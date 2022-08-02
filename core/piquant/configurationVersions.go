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
	"strings"

	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
)

// PIQUANT config versioning

// REFACTOR: this file seems to have 2 similar functions, do we need this many?

// GetPiquantConfigVersions - retrieves all available versions of a given named PIQUANT config
// eg. Returns all versions for config called PIXL or Breadboard...
func GetPiquantConfigVersions(svcs *services.APIServices, configName string) []string {
	versionPaths, err := listPIQUANTConfigVersionPaths(svcs, configName)
	if err != nil {
		return []string{}
	}

	cfgPrefix := filepaths.GetDetectorConfigPath(configName, "", "") + "/"
	versions := getPIQUANTVersionsFromVersionsPaths(cfgPrefix, versionPaths)
	return versions
}

func listPIQUANTConfigVersionPaths(svcs *services.APIServices, configName string) ([]string, error) {
	versionPaths := []string{}

	s3Path := filepaths.GetDetectorConfigPath(configName, "", "") + "/"
	versionPaths, err := svcs.FS.ListObjects(svcs.Config.ConfigBucket, s3Path)
	if err != nil {
		svcs.Log.Errorf("Failed to list piquant configs in %v/%v: %v", svcs.Config.ConfigBucket, s3Path, err)
	}

	return versionPaths, err
}

func getPIQUANTVersionsFromVersionsPaths(knownPathPrefix string, paths []string) []string {
	// Expecting paths of the form: DetectorConfig/PetersSuperDetector/PiquantConfigs/V1/config.json
	// Only look at paths that are for a piquantConfigFileName (the file that ties together the detector config)
	// We return version file paths of the form: V1
	versions := []string{}
	const cfgSuffix = "/" + filepaths.PiquantConfigFileName

	for _, path := range paths {
		if strings.HasPrefix(path, knownPathPrefix) && strings.HasSuffix(path, cfgSuffix) {
			// Snip off the suffix and the path up to that point
			version := path[len(knownPathPrefix) : len(path)-len(cfgSuffix)]
			versions = append(versions, version)
		}
	}

	return versions
}
