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

package importtime

import (
	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
)

type LastImportTimes struct {
	Times map[string]int `json:"times"`
}

func SaveDatasetImportUnixTimeSec(fs fileaccess.FileAccess, log logger.ILogger, configBucket string, datasetID string, unixTimeSec int) error {
	loads := LastImportTimes{Times: map[string]int{}}

	err := fs.ReadJSON(configBucket, filepaths.DatasetLastImportTimesPath, &loads, false)

	// If file doesn't exist, we must just be doing this for the first time, that's ok
	// Also, really, if file is corrupt, what can we do... we restart fresh, so we just log in this case!
	if err != nil && !fs.IsNotFoundError(err) {
		log.Errorf("Failed to read last dataset import times, fresh file will be written. Error was: %v", err)
	}

	// Set it in the map
	loads.Times[datasetID] = unixTimeSec

	// Write it back
	return fs.WriteJSONNoIndent(configBucket, filepaths.DatasetLastImportTimesPath, loads)
}

// GetDatasetImportUnixTimeSec - Returns the last unix time seconds timestamp that the dataset with given datasetID was imported
// or 0 if never
func GetDatasetImportUnixTimeSec(fs fileaccess.FileAccess, configBucket string, datasetID string) (int, error) {
	loads := LastImportTimes{Times: map[string]int{}}

	err := fs.ReadJSON(configBucket, filepaths.DatasetLastImportTimesPath, &loads, false)

	// If file doesn't exist, we must just be doing this for the first time, that's ok
	if err != nil && !fs.IsNotFoundError(err) {
		return 0, err
	}

	timestamp, ok := loads.Times[datasetID]
	if !ok {
		// Not found, so likely not loaded before
		return 0, nil
	}

	return timestamp, nil
}
