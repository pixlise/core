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

package quantModel

import (
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/services"
)

const quantBlessedFile = "blessed-quant.json"

type BlessFileItem struct {
	Version      int    `json:"version"`
	BlessUnixSec int64  `json:"blessedAt"`
	UserID       string `json:"userId"`
	UserName     string `json:"userName"`
	JobID        string `json:"jobId"`
}

type BlessFile struct {
	History []BlessFileItem `json:"history"`
}

// Downloads & parses the blessed quants file.
// Returns:
// - the parsed contents
// - the blessed quant job info (BlessItem)
// - the path (in case we want to update the same file)
// - error or nil
func GetBlessedQuantFile(svcs *services.APIServices, datasetID string) (BlessFile, *BlessFileItem, string, error) {
	blessFilePath := filepaths.GetSharedQuantPath(datasetID, quantBlessedFile)

	blessFile := BlessFile{}
	err := svcs.FS.ReadJSON(svcs.Config.UsersBucket, blessFilePath, &blessFile, false)
	if err != nil {
		if !svcs.FS.IsNotFoundError(err) {
			return blessFile, nil, blessFilePath, err
		}
		// else it WAS a "not found" error, in which case we continue - the first blessing will always find this scenario
	}

	// Find the blessed quant job ID (one with highest version)
	highestVersion := 0
	var blessItem *BlessFileItem = nil

	for _, item := range blessFile.History {
		if item.Version > highestVersion {
			highestVersion = item.Version
			blessItem = &BlessFileItem{
				Version:      item.Version,
				BlessUnixSec: item.BlessUnixSec,
				UserID:       item.UserID,
				UserName:     item.UserName,
				JobID:        item.JobID,
			}
		}
	}

	return blessFile, blessItem, blessFilePath, nil
}
