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
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"

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

	localFS := fileaccess.FSAccess{}
	fileList, _ := localFS.ListObjects(sourcePath, "")
	count := 0
	totalCount := len(fileList)

	jobs := make(chan uploadJob, totalCount)
	results := make(chan uploadResult, totalCount)

	// Start workers
	numUploaders := 1
	if totalCount > 10 {
		numUploaders = 5
		for w := 1; w <= numUploaders; w++ {
			go uploadWorker(w, jobs, results)
		}
	}

	err := filepath.Walk(sourcePath, func(currentPath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
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

			jobs <- uploadJob{
				currentPath:  currentPath,
				uploadPath:   uploadPath,
				destBucket:   destBucket,
				log:          log,
				remoteFS:     remoteFS,
				currentCount: count + 1,
				totalCount:   totalCount,
			}

			count++
		}
		return nil
	})

	if err != nil {
		return err
	}

	if numUploaders > 1 && uploadError == nil {
		close(jobs)

		// Check each upload for an error
		fails := 0
		failedPaths := []string{}
		var firstError error
		for c := 0; c < totalCount; c++ {
			result := <-results
			if result.err != nil {
				if firstError == nil {
					firstError = result.err
				}
				fails++

				if len(failedPaths) < 10 {
					failedPaths = append(failedPaths, result.uploadPath)
				}
			}
		}

		if fails > 0 {
			uploadError = fmt.Errorf("Failed to upload %v files. Paths (up to 10): %v. First error: %v", fails, strings.Join(failedPaths, ","), firstError)
		}
	}

	return uploadError
}

type uploadJob struct {
	currentPath string
	uploadPath  string

	destBucket string

	totalCount   int
	currentCount int

	log logger.ILogger

	remoteFS fileaccess.FileAccess
}

type uploadResult struct {
	uploadPath string
	err        error
}

func uploadWorker(id int, jobs <-chan uploadJob, results chan<- uploadResult) {
	for j := range jobs {
		data, err := os.ReadFile(j.currentPath)
		if err != nil {
			j.log.Errorf("Failed to read file for upload: %v", j.currentPath)
		} else {
			j.log.Infof("-Uploading [%v/%v]: %v", j.currentCount, j.totalCount, j.currentPath)
			j.log.Infof("---->to s3://%v/%v", j.destBucket, j.uploadPath)
			err = j.remoteFS.WriteObject(j.destBucket, j.uploadPath, data)

			if err != nil {
				j.log.Errorf("Failed to upload to s3://%v/%v: %v", j.destBucket, j.uploadPath, err)
			}
		}

		results <- uploadResult{err: err, uploadPath: j.uploadPath}
	}
}
