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

package importerutils

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
)

func ReadMatchedImages(matchedPath string, beamLookup dataConvertModels.BeamLocationByPMC, jobLog logger.ILogger, fs fileaccess.FileAccess) ([]dataConvertModels.MatchedAlignedImageMeta, error) {
	result := []dataConvertModels.MatchedAlignedImageMeta{}

	// Read all JSON files in the directory, if they reference a context image by file name great, otherwise error
	//files, err := GetDirListing(matchedPath, "json", jobLog)
	files, err := fs.ListObjects(matchedPath, "")

	if err != nil {
		jobLog.Infof("readMatchedImages: directory not found, SKIPPING")
		return result, nil
	}

	for _, jsonFile := range files {
		// Ensure they are .json files
		if strings.ToUpper(path.Ext(jsonFile)) != ".JSON" {
			continue
		}

		jsonPath := path.Join(matchedPath, jsonFile)
		// Read JSON file
		jsonBytes, err := ioutil.ReadFile(jsonPath)
		if err != nil {
			return result, err
		}

		var meta dataConvertModels.MatchedAlignedImageMeta
		err = json.Unmarshal(jsonBytes, &meta)
		if err != nil {
			return result, err
		}

		// Verify the beams exist (though if we have NO beams, we're a "disco" dataset, so skip this)
		if len(beamLookup) > 0 {
			if _, ok := beamLookup[meta.AlignedBeamPMC]; !ok {
				return result, fmt.Errorf("Matched image %v references beam locations for PMC which cannot be found: %v", jsonPath, meta.AlignedBeamPMC)
			}
		}

		// Work out the full path, will be needed when copying to output dir
		meta.MatchedImageFullPath = path.Join(matchedPath, meta.MatchedImageName)

		_, err = os.Stat(meta.MatchedImageFullPath)
		if err != nil {
			return result, fmt.Errorf("Matched image %v references image which cannot be found: %v", jsonPath, meta.MatchedImageName)
		}

		// And the offsets are valid. I doubt we'll be loading images much larger than maxSize:
		const maxSize = 10000.0
		if meta.XOffset < -maxSize || meta.XOffset > maxSize || meta.YOffset < -maxSize || meta.YOffset > maxSize {
			return result, fmt.Errorf("%v x/y offsets invalid", jsonPath)
		}

		// And the scale values are valid
		const maxScale = 100.0 // 100x greater/less resolution... not likely!
		if meta.XScale < 1/maxScale || meta.XScale > maxScale || meta.YScale < 1/maxScale || meta.YScale > maxScale {
			return result, fmt.Errorf("%v x/y scales invalid", jsonPath)
		}

		result = append(result, meta)
	}

	return result, nil
}
