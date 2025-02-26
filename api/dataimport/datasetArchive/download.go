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
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
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

// Returns:
// Downloads path (raw zip files go here),
// Unzipped files path (archive zips unzipped here),
// How many zips loaded from archive
// Error (if any)
func (dl *DatasetArchiveDownloader) DownloadFromDatasetArchive(datasetID string, workingDir string) (string, string, []string, error) {
	// Create a directories to process data in
	dl.log.Debugf("Preparing to download archived dataset %v...", datasetID)

	downloadPath, err := fileaccess.MakeEmptyLocalDirectory(workingDir, "download")
	if err != nil {
		err = fmt.Errorf("Failed to generate directory for importer downloads: %v", err)
		//dl.log.Errorf("%v", err)
		return "", "", []string{}, err
	}
	unzippedPath, err := fileaccess.MakeEmptyLocalDirectory(workingDir, "unzipped")
	if err != nil {
		err = fmt.Errorf("Failed to generate directory for importer unzips: %v", err)
		//dl.log.Errorf("%v", err)
		return "", "", []string{}, err
	}

	// Download all zip files from archive for this dataset ID, and extract them as required
	dl.log.Debugf("Downloading archived zip files...")

	zipFilesOrdered, err := dl.downloadArchivedZipsForDataset(datasetID, downloadPath, unzippedPath)
	if err != nil {
		err = fmt.Errorf("Failed to download archived zip files for dataset ID: %v. Error: %v", datasetID, err)
		//dl.log.Errorf("%v", err)
		return downloadPath, unzippedPath, zipFilesOrdered, err
	}

	dl.log.Debugf("Dataset %v downloaded %v zip files from archive", datasetID, len(zipFilesOrdered))
	return downloadPath, unzippedPath, zipFilesOrdered, nil
}

func (dl *DatasetArchiveDownloader) DownloadPseudoIntensityRangesFile(configBucket string, downloadPath string, version string) (string, error) {
	// Download the ranges file
	dl.log.Debugf("Downloading pseudo-intensity ranges...")

	fileName := "StandardPseudoIntensities" + version + ".csv"
	localRangesPath := filepath.Join(downloadPath, fileName)
	err := dl.fetchFile(configBucket, path.Join(filepaths.RootDatasetConfig, fileName), localRangesPath)
	if err != nil {
		dl.log.Errorf("%v", err)
		return "", err
	}

	return localRangesPath, err
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
func (dl *DatasetArchiveDownloader) downloadArchivedZipsForDataset(datasetID string, downloadPath string, unzippedPath string) ([]string, error) {
	// Download all zip files that have the dataset ID prefixed in their file name
	// Unzip them in timestamp order into downloadPath
	archiveSearchPath := path.Join(filepaths.RootArchive, datasetID)

	// NOTE: For importing datasets from FM, we don't want a / at the end, but for importing from uploaded data, we do!
	// Uploaded datasets may have the same prefix at the start (eg user uploads dataset AA then later uploads A) so if
	// we don't have a trailing / when reading dataset A, we'd get the files from AA and it'll fail. For this reason
	// we have a second attempt after this with no / if no files were found
	if !strings.HasSuffix(archiveSearchPath, "/") {
		archiveSearchPath = archiveSearchPath + "/"
	}

	dl.log.Infof("Searching for archived files in: s3://%v/%v", dl.datasetBucket, archiveSearchPath)

	archivedFiles, err := dl.remoteFS.ListObjects(dl.datasetBucket, archiveSearchPath)
	if err != nil {
		return []string{}, err
	}

	// If nothing has been found try search again without a trailing /
	if len(archivedFiles) <= 0 {
		archiveSearchPath = path.Join(filepaths.RootArchive, datasetID)

		dl.log.Infof("Searching again for archived files in: s3://%v/%v", dl.datasetBucket, archiveSearchPath)

		archivedFiles, err = dl.remoteFS.ListObjects(dl.datasetBucket, archiveSearchPath)
		if err != nil {
			return []string{}, err
		}
	}

	orderedArchivedFiles, err := getOrderedArchiveFiles(archivedFiles)

	if err != nil {
		// Stop here if we find a bad file
		return []string{}, err
	}

	fileCount := 0

	for _, filePath := range orderedArchivedFiles {
		fileName := path.Base(filePath)
		if !strings.HasSuffix(fileName, ".zip") {
			return []string{}, errors.New("Expected zip file, got: " + fileName)
		}

		savePath := filepath.Join(downloadPath, fileName)
		err = dl.fetchFile(dl.datasetBucket, filePath, savePath)

		if err != nil {
			return []string{}, err
		}

		dl.log.Debugf("Unzipping: \"%v\"", savePath)

		// Unzip the file
		unzippedFileNames, err := utils.UnzipDirectory(savePath, unzippedPath, false)
		if err != nil {
			return []string{}, err
		}

		fileCount += len(unzippedFileNames)

		// Delete the source zip file so we don't keep expanding the space we're using
		err = os.RemoveAll(savePath)
		if err != nil {
			dl.log.Errorf("Failed to delete zip file after unzipping: \"%v\". Error: %v", savePath, err)
			// Don't die for this
			err = nil
		} else {
			dl.log.Debugf("Deleted zip file after unzipping: \"%v\"", savePath)
		}
	}

	lastFileName := ""
	if len(orderedArchivedFiles) > 0 {
		lastFileName = orderedArchivedFiles[len(orderedArchivedFiles)-1]
	}

	dl.log.Infof("Downloaded %v zip files, unzipped %v files. Last file name: %v", len(orderedArchivedFiles), fileCount, lastFileName)
	return orderedArchivedFiles, nil
}

func (dl *DatasetArchiveDownloader) DownloadUserCustomisationsForDataset(datasetID string, downloadPath string) error {
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
		savePath := filepath.Join(parts...)

		err = dl.fetchFile(dl.manualUploadBucket, uploadedPath, savePath)
		if err != nil {
			return err
		}
	}

	return nil
}

// Downloads from user uploaded dataset zip area. Expects the following files to exist:
// - creator.json - describing who uploaded the dataset, and when
// - detector.json - describing what detector, hence what dataset type this is
// Other files depending on what type of detector:
// BREADBOARD:
// - import.json - import parameters for the jpl breadboard importer
// - spectra.zip - all .MSA files
//
// Returns:
// Downloads path (raw zip files go here),
// Unzipped files path (archive zips unzipped here),
// Error (if any)
func (dl *DatasetArchiveDownloader) DownloadFromDatasetUploads(datasetID string, workingDir string) (string, string, error) {
	// Create a directories to process data in
	dl.log.Debugf("Preparing to download manually-uploaded dataset %v...", datasetID)

	downloadPath, err := fileaccess.MakeEmptyLocalDirectory(workingDir, "download")
	if err != nil {
		err = fmt.Errorf("Failed to generate directory for importer downloads: %v", err)
		dl.log.Errorf("%v", err)
		return "", "", err
	}
	unzippedPath, err := fileaccess.MakeEmptyLocalDirectory(workingDir, "unzipped")
	if err != nil {
		err = fmt.Errorf("Failed to generate directory for importer unzips: %v", err)
		dl.log.Errorf("%v", err)
		return "", "", err
	}

	// Download all files for this dataset...
	pathsToDownload, err := dl.remoteFS.ListObjects(dl.manualUploadBucket, path.Join(filepaths.DatasetUploadRoot, datasetID))
	if err != nil {
		err = fmt.Errorf("Failed to list files for download from user upload area: %v", err)
		dl.log.Errorf("%v", err)
		return "", "", err
	}

	for _, filePath := range pathsToDownload {
		// Zip files go to download area and get unzipped into unzip dir, non-zips go straight to unzip dir
		savePath := path.Base(filePath)
		zipName := ""
		if strings.HasSuffix(filePath, ".zip") {
			savePath = filepath.Join(downloadPath, savePath)
			zipName = path.Base(filePath)
			zipName = zipName[0 : len(zipName)-4] // Snip off the .zip
		} else {
			savePath = filepath.Join(unzippedPath, savePath)
		}

		err = dl.fetchFile(dl.manualUploadBucket, filePath, savePath)
		if err != nil {
			err = fmt.Errorf("Failed to download file: %v", err)
			dl.log.Errorf("%v", err)
			return "", "", err
		}

		if len(zipName) > 0 {
			// Unzip it!
			zipDest := filepath.Join(unzippedPath, zipName)
			_, err := utils.UnzipDirectory(savePath, zipDest, false) // We used to flatten paths for uploads, but no longer, we support FM format so need subdirs
			if err != nil {
				err = fmt.Errorf("Failed to unzip %v: %v", savePath, err)
				dl.log.Errorf("%v", err)
				return "", "", err
			}

			// Delete the source zip file so we don't keep expanding the space we're using
			err = os.RemoveAll(savePath)
			if err != nil {
				dl.log.Errorf("Failed to delete zip file after unzipping: \"%v\". Error: %v", savePath, err)
				// Don't die for this
				err = nil
			} else {
				dl.log.Infof("Deleted zip file after unzipping: \"%v\"", savePath)
			}
		}
	}

	dl.log.Debugf("Dataset %v downloaded %v files from manual upload area", datasetID, len(pathsToDownload))
	return downloadPath, unzippedPath, nil
}
