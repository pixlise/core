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
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/core/utils"
)

type DatasetArchiveDownloader struct {
	localFS            fileaccess.FileAccess
	remoteFS           fileaccess.FileAccess
	log                logger.ILogger
	datasetBucket      string
	manualUploadBucket string
}

func NewDatasetArchiveDownloader(
	remoteFS fileaccess.FileAccess,
	localFS fileaccess.FileAccess,
	log logger.ILogger,
	datasetBucket string,
	manualUploadBucket string) *DatasetArchiveDownloader {
	return &DatasetArchiveDownloader{
		localFS:            localFS,
		remoteFS:           remoteFS,
		log:                log,
		datasetBucket:      datasetBucket,
		manualUploadBucket: manualUploadBucket,
	}
}

// Returns output path, how many zips loaded from archive, and error if any
func (dl *DatasetArchiveDownloader) DownloadFromDatasetArchive(datasetID string) (string, int, error) {
	// Create a directories to process data in
	dl.log.Debugf("Preparing download dataset %v...", datasetID)

	downloadRoot, err := ioutil.TempDir("", "archive")
	if err != nil {
		err = fmt.Errorf("Failed to generate directory for importer downloads: %v", err)
		dl.log.Errorf("%v", err)
		return "", 0, err
	}
	downloadPath := path.Join(downloadRoot, "downloads")
	unzippedPath := path.Join(downloadRoot, "unzipped")
	outputPath := path.Join(downloadRoot, "output")

	// Make sure both exist and are empty
	prepDirs := []string{downloadPath, unzippedPath, outputPath}

	for _, dir := range prepDirs {
		err = os.MkdirAll(dir, os.ModePerm)
		if err != nil {
			err = fmt.Errorf("Failed to create directory %v for importer: %v", dir, err)
			dl.log.Errorf("%v", err)
			return outputPath, 0, err
		}

		err = dl.localFS.EmptyObjects(dir)
		if err != nil {
			err = fmt.Errorf("Failed to clear directory %v for importer: %v", dir, err)
			dl.log.Errorf("%v", err)
			return outputPath, 0, err
		}
	}

	// Download all zip files from archive for this dataset ID, and extract them as required
	dl.log.Debugf("Downloading archived zip files...")

	zipCount, err := dl.downloadArchivedZipsForDataset(datasetID, downloadPath, unzippedPath)
	if err != nil {
		err = fmt.Errorf("Failed to download archived zip files for dataset ID: %v. Error: %v", datasetID, err)
		dl.log.Errorf("%v", err)
		return outputPath, zipCount, err
	}

	// Download any additional files users may have manually added, eg custom config (dataset name), custom images, RGBU images
	dl.log.Debugf("Downloading user customisation files...")

	err = dl.downloadUserCustomisationsForDataset(datasetID, unzippedPath)
	if err != nil {
		err = fmt.Errorf("Failed to download user customisations for dataset ID: %v. Error: %v", datasetID, err)
		dl.log.Errorf("%v", err)
		return outputPath, zipCount, err
	}

	dl.log.Debugf("Dataset %v downloaded from archive", datasetID)
	return outputPath, zipCount, nil
}

// Fetches from given bucket/path, writes to given savePath, ensures any intermediate directories in savePath exist
func (dl *DatasetArchiveDownloader) fetchFile(bucketFrom string, pathFrom string, savePath string) error {
	dl.log.Debugf("-Save: s3://%v/%v", bucketFrom, pathFrom)
	dl.log.Debugf("-->to: %v", savePath)

	bytes, err := dl.remoteFS.ReadObject(bucketFrom, pathFrom)
	if err != nil {
		return err
	}

	return dl.localFS.WriteObject(savePath, "", bytes)
}

// Returns 2 things:
// Number of zips loaded
// Error if there was one
func (dl *DatasetArchiveDownloader) downloadArchivedZipsForDataset(datasetID string, downloadPath string, unzippedPath string) (int, error) {
	// Download all zip files that have the dataset ID prefixed in their file name
	// Unzip them in timestamp order into downloadPath
	archiveSearchPath := path.Join(filepaths.RootArchive, datasetID)
	dl.log.Infof("Searching for archived files in: s3://%v/%v", dl.datasetBucket, archiveSearchPath)

	archivedFiles, err := dl.remoteFS.ListObjects(dl.datasetBucket, archiveSearchPath)
	if err != nil {
		return 0, err
	}

	orderedArchivedFiles, err := getOrderedArchiveFiles(archivedFiles)

	if err != nil {
		// Stop here if we find a bad file
		return 0, err
	}

	fileCount := 0

	for _, filePath := range orderedArchivedFiles {
		fileName := path.Base(filePath)
		if !strings.HasSuffix(fileName, ".zip") {
			return 0, errors.New("Expected zip file, got: " + fileName)
		}

		savePath := path.Join(downloadPath, fileName)
		err = dl.fetchFile(dl.datasetBucket, filePath, savePath)

		if err != nil {
			return 0, err
		}

		// Unzip the file
		unzippedFileNames, err := utils.UnzipDirectory(savePath, unzippedPath)
		if err != nil {
			return 0, err
		}

		fileCount += len(unzippedFileNames)
	}

	dl.log.Infof("Downloaded %v zip files, unzipped %v files", len(orderedArchivedFiles), fileCount)
	return len(orderedArchivedFiles), nil
}

func (dl *DatasetArchiveDownloader) downloadUserCustomisationsForDataset(datasetID string, downloadPath string) error {
	// Download all files for the given dataset ID from user manual upload bucket/path
	// into downloadPath
	uploadedFiles, err := dl.remoteFS.ListObjects(dl.manualUploadBucket, path.Join(filepaths.DatasetCustomRoot, datasetID))
	if err != nil {
		return err
	}

	for _, uploadedPath := range uploadedFiles {
		fileName, middleDirs, err := decodeManualUploadPath(uploadedPath)

		if err != nil {
			return err
		}

		// We need to form a path starting at downloadPath that preserves the file structure of what's in the bucket
		// Here it forms something like <downloadPath>/file.png OR <downloadPath>/MATCHED/file.png
		parts := append([]string{downloadPath}, middleDirs...)
		parts = append(parts, fileName)
		savePath := path.Join(parts...)

		err = dl.fetchFile(dl.manualUploadBucket, uploadedPath, savePath)
		if err != nil {
			return err
		}
	}

	return nil
}
