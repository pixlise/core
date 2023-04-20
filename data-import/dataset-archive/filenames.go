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

package datasetArchive

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strings"
	"time"

	"github.com/pixlise/core/v3/core/utils"
)

// Implements functions that deal with file names or listings stored in S3

func getOrderedArchiveFiles(archivedFiles []string) ([]string, error) {
	filesByTimeStamp := map[int]string{}
	fileTimestamps := []string{}

	if len(archivedFiles) > 0 {
		// Form timestamp->file name map
		for _, fileName := range archivedFiles {
			_ /*expecting this to match already due to dir listing*/, timeStamp, err := DecodeArchiveFileName(fileName)
			if err != nil {
				return []string{}, err
			}

			filesByTimeStamp[timeStamp] = fileName
		}

		timeStamps := make([]int, 0, len(filesByTimeStamp))
		for ts := range filesByTimeStamp {
			timeStamps = append(timeStamps, ts)
		}
		sort.Ints(timeStamps)

		for _, timeStamp := range timeStamps {
			fileTimestamps = append(fileTimestamps, filesByTimeStamp[timeStamp])
		}
	}

	return fileTimestamps, nil
}

func DecodeArchiveFileName(fileName string) (string, int, error) {
	// We're expecting archived files to be named along the lines of: 161677829-12-06-2022-06-41-00.zip
	// Where the first part is the dataset ID (hence the prefix above working to list them) and then a time stamp
	splits := strings.SplitN(fileName, "-", 2)
	if len(splits) != 2 {
		return "", 0, errors.New("DecodeArchiveFileName unexpected file name: " + fileName)
	}
	// splits[0] is the dataset ID, splits[1] is "the rest"
	datasetID := splits[0]

	// Remove file extension:
	strTimestamp := splits[1]
	ext := path.Ext(strTimestamp)
	strTimestamp = strTimestamp[0 : len(strTimestamp)-len(ext)]

	layout := "02-01-2006-15-04-05"
	timestamp, err := time.Parse(layout, strTimestamp)
	if err != nil {
		return "", 0, fmt.Errorf("DecodeArchiveFileName \"%v\" error: %v", fileName, err)
	}

	return datasetID, int(utils.AbsI64(timestamp.Unix())), nil
}

// Expecting paths of the form: /dataset-addons/datasetID/custom-meta.json AND /dataset-addons/datasetID/MATCHED/something.png or .json
// Returns file name, type dir (MATCHED in above example) or error
func decodeManualUploadPath(filePath string) (string, []string, error) {
	fileName := path.Base(filePath)

	// If path starts with a /, skip that
	filePath = strings.TrimLeft(filePath, "/")
	pathParts := strings.Split(filePath, "/")
	if len(pathParts) > 3 {
		pathParts = pathParts[2 : len(pathParts)-1]
	} else if len(pathParts) == 3 && pathParts[2] == "custom-meta.json" {
		pathParts = pathParts[2 : len(pathParts)-1]
	} else {
		return "", []string{}, errors.New("Manual upload path invalid: " + filePath)
	}

	return fileName, pathParts, nil
}
