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
	"os"
	"path"
	"path/filepath"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
)

func LogIfMoreFoundMSA(m dataConvertModels.DetectorSampleByPMC, typename string, morethan int, log logger.ILogger) {
	for k, v := range m {
		if len(v) > morethan {
			log.Infof("PMC %d has %d %s entries", k, len(v), typename)
		}
	}
}

// Copies files to bucket
// If preserveStructure is true, preserves directory structure from sourcePath.
// If preserveStructure is false, copies all files flat (just filename, no subdirectories).
func CopyToBucket(remoteFS fileaccess.FileAccess, datasetID string, sourcePath string, destBucket string, destPath string, preserveStructure bool, log logger.ILogger) error {
	var uploadError error

	err := filepath.Walk(sourcePath, func(currentPath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(currentPath)
			if err != nil {
				log.Errorf("Failed to read file for upload: %v", currentPath)
				uploadError = err
			} else {
				var uploadPath string

				if preserveStructure {
					// Calculate relative path from sourcePath to preserve directory structure
					relPath, err := filepath.Rel(sourcePath, currentPath)
					if err != nil {
						log.Errorf("Failed to calculate relative path: %v", err)
						uploadError = err
						return nil
					}
					// Upload to: destPath/datasetID/relPath (e.g., Images/BigTiff/Multi_page24bpp/page_0.dzi)
					uploadPath = path.Join(destPath, datasetID, filepath.ToSlash(relPath))
				} else {
					// Flat structure: just use the base filename
					sourceFile := filepath.Base(currentPath)
					uploadPath = path.Join(destPath, datasetID, sourceFile)
				}

				log.Infof("-Uploading: %v", currentPath)
				log.Infof("---->to s3://%v/%v", destBucket, uploadPath)
				err = remoteFS.WriteObject(destBucket, uploadPath, data)

				if err != nil {
					log.Errorf("Failed to upload to s3://%v/%v: %v", destBucket, uploadPath, err)
					uploadError = err
				}
			}
		}
		return nil
	})

	if err != nil {
		return err
	}

	return uploadError
}
