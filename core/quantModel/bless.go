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

package quantModel

import (
	"github.com/pixlise/core/api/filepaths"
	"github.com/pixlise/core/api/services"
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
