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
	"strings"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
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
