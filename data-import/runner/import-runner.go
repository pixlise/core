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

package runner

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/awsutil"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/core/utils"
	dataConverter "github.com/pixlise/core/v2/data-import/data-converter"
	"github.com/pixlise/core/v2/data-import/data-converters/jplbreadboard"
	"github.com/pixlise/core/v2/data-import/data-converters/pixlfm"
	"github.com/pixlise/core/v2/data-import/internal/importerutils"
	"github.com/pixlise/core/v2/data-import/output"
)

// Processes any trigger that we have, and calls import dataset with the right parameters
//
// Triggers Possible:
//
// SNS Message from Dataset Edit page Save button (API endpoint: PUT dataset/meta/datasetID):
//   {"datasetaddons":{"dir": "dataset-addons/datasetID/custom-meta.json", "log": "dataimport-a1b2c3d4e5f6g7h8"}}
// Used when user clicks save button
//
// SNS Message which contains an S3 trigger for a zip that landed from OCS world
//   After downloading and unzipping the file it is deleted
//
// NOTE: New data zip files can be dropped into the "rawdata" buckets (one for staging, and one for prod)
//       and that's where the SNS wrapped S3 triggers will come from
// NOTE2: User breadboard dataset uploads will have to be saved in the manual upload bucket (per env) and we'll
//        need to trigger a dataset generation from there!
//
// FOR EXAMPLE MESSAGES: see trigger.go and the unit tests for it

func RunDatasetImport(triggerMessageBody []byte, configBucket string, datasetBucket string, manualBucket string, envName string) error {
	if len(envName) <= 0 {
		return errors.New("ENVIRONMENT_NAME not configured")
	}

	var err error

	// If we're just being asked to re-generate a dataset, we end up with a dataset ID
	datasetID := ""

	// Log ID to use - this forms part of the log stream in cloudwatch
	logID := ""

	// But if we're being triggered due to new data arriving, these will be filled out
	sourceFilePath := ""
	sourceBucket := ""

	sourceBucket, sourceFilePath, datasetID, logID, err = decodeImportTrigger(triggerMessageBody)

	if err != nil {
		return err
	}

	// Initialise stuff
	sess, err := awsutil.GetSession()
	if err != nil {
		return err
	}

	log, err := logger.InitCloudWatchLogger(sess, "/dataset-importer/"+envName, datasetID+"-"+logID, logger.LogDebug, 30, 3)
	if err != nil {
		return err
	}

	svc, err := awsutil.GetS3(sess)
	if err != nil {
		return err
	}

	localFS := fileaccess.FSAccess{}
	remoteFS := fileaccess.MakeS3Access(svc)

	importer := dataImporter{
		remoteFS:           remoteFS,
		localFS:            &localFS,
		log:                log,
		datasetBucket:      datasetBucket,
		configBucket:       configBucket,
		manualUploadBucket: manualBucket,
	}

	updateExisting, updateType, datasetName, err := importer.importData(datasetID, sourceBucket, sourceFilePath)
	if err != nil {
		return err
	}

	// It worked! Trigger notifications
	log.Infof("Triggering Notifications...")
	if updateType != "trivial" {
		updatenotificationtype, err := getUpdateNotificationType(datasetID, datasetBucket, remoteFS)
		if err != nil {
			return err
		}

		ns := makeNotificationStack(remoteFS, log)
		err = triggerNotifications(configBucket, datasetName, remoteFS, updateExisting, updatenotificationtype, ns, log)
		if err != nil {
			return err
		}
	}

	return nil
}

type dataImporter struct {
	localFS            fileaccess.FileAccess
	remoteFS           fileaccess.FileAccess
	log                logger.ILogger
	datasetBucket      string
	configBucket       string
	manualUploadBucket string
}

// Returns 4 things:
// update flag (bool) - true if this dataset already had 1 or more files in the archive
// updateType, read from config.json
// dataset name (string)
// error if there was one
func (i *dataImporter) importData(datasetID string, sourceBucket string, sourceFilePath string) (bool, string, string, error) {
	var err error
	var update bool

	// Validate config
	if len(i.datasetBucket) <= 0 || len(i.configBucket) <= 0 || len(i.manualUploadBucket) <= 0 {
		err = errors.New("One or more environment variables not set")
		i.log.Errorf("%v", err)
		return update, "", "", err
	}

	// If we're triggered by a file arriving, add it to the archive
	archived := false
	if len(sourceBucket) > 0 && len(sourceFilePath) > 0 {
		i.log.Debugf("Archiving source file: \"s3://%v/%v\"", sourceBucket, sourceFilePath)

		// Work out the file name
		fileName := path.Base(sourceFilePath)

		err = i.remoteFS.CopyObject(sourceBucket, sourceFilePath, i.datasetBucket, path.Join(filepaths.RootArchive, fileName))
		if err != nil {
			err = fmt.Errorf("Failed to archive incoming file: \"s3://%v/%v\"", sourceBucket, sourceFilePath)
			i.log.Errorf("%v", err)
			return update, "", "", err
		}
		archived = true
	} else if len(sourceBucket) > 0 || len(sourceFilePath) > 0 {
		// We need BOTH to be set to something for this to work, only one of them is set
		err = fmt.Errorf("Trigger message must specify bucket AND path, received bucket=%v, path=%v", sourceBucket, sourceFilePath)
		i.log.Errorf("%v", err)
		return update, "", "", err
	}

	// Create a directories to process data in
	i.log.Debugf("Preparing download directory...")
	downloadRoot := ""
	downloadRoot, err = ioutil.TempDir("", "archive")
	if err != nil {
		err = fmt.Errorf("Failed to generate directory for importer downloads: %v", err)
		i.log.Errorf("%v", err)
		return update, "", "", err
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
			i.log.Errorf("%v", err)
			return update, "", "", err
		}

		err = i.localFS.EmptyObjects(dir)
		if err != nil {
			err = fmt.Errorf("Failed to clear directory %v for importer: %v", dir, err)
			i.log.Errorf("%v", err)
			return update, "", "", err
		}
	}

	// Download all zip files from archive for this dataset ID, and extract them as required
	i.log.Debugf("Downloading archived zip files...")

	zipCount, err := i.downloadArchivedZipsForDataset(datasetID, downloadPath, unzippedPath)
	if err != nil {
		err = fmt.Errorf("Failed to download archived zip files for dataset ID: %v. Error: %v", datasetID, err)
		i.log.Errorf("%v", err)
		return update, "", "", err
	}

	// If we just archived a zip file AND there were others stored already
	// OR if we didn't archive a zip file...
	// we are updating the dataset!
	if archived {
		update = zipCount > 1
	} else {
		update = true
	}

	// Download the ranges file
	i.log.Debugf("Downloading pseudo-intensity ranges...")

	localRangesPath := path.Join(downloadPath, "StandardPseudoIntensities.csv")
	err = i.fetchFile(i.configBucket, "DatasetConfig/StandardPseudoIntensities.csv", localRangesPath)
	if err != nil {
		i.log.Errorf("%v", err)
		return update, "", "", err
	}

	// Download any additional files users may have manually added, eg custom config (dataset name), custom images, RGBU images
	i.log.Debugf("Downloading user customisation files...")

	err = i.downloadUserCustomisationsForDataset(datasetID, unzippedPath)
	if err != nil {
		err = fmt.Errorf("Failed to download user customisations for dataset ID: %v. Error: %v", datasetID, err)
		i.log.Errorf("%v", err)
		return update, "", "", err
	}

	i.log.Debugf("Downloads complete, running importer")
	datasetName, importConfig, err := i.importDataFiles(unzippedPath, localRangesPath, outputPath)
	return update, importConfig.UpdateType, datasetName, err
}

// Fetches from given bucket/path, writes to given savePath, ensures any intermediate directories in savePath exist
func (i *dataImporter) fetchFile(bucketFrom string, pathFrom string, savePath string) error {
	i.log.Debugf("-Save: s3://%v/%v", bucketFrom, pathFrom)
	i.log.Debugf("-->to: %v", savePath)

	bytes, err := i.remoteFS.ReadObject(bucketFrom, pathFrom)
	if err != nil {
		return err
	}

	return i.localFS.WriteObject(savePath, "", bytes)
}

// Returns 2 things:
// Number of zips loaded
// Error if there was one
func (i *dataImporter) downloadArchivedZipsForDataset(datasetID string, downloadPath string, unzippedPath string) (int, error) {
	// Download all zip files that have the dataset ID prefixed in their file name
	// Unzip them in timestamp order into downloadPath
	archiveSearchPath := path.Join(filepaths.RootArchive, datasetID)
	i.log.Infof("Searching for archived files in: s3://%v/%v", i.datasetBucket, archiveSearchPath)

	archivedFiles, err := i.remoteFS.ListObjects(i.datasetBucket, archiveSearchPath)
	if err != nil {
		return 0, err
	}

	orderedArchivedFiles, err := importerutils.GetOrderedArchiveFiles(archivedFiles)

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
		err = i.fetchFile(i.datasetBucket, filePath, savePath)

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

	i.log.Infof("Downloaded %v zip files, unzipped %v files", len(orderedArchivedFiles), fileCount)
	return len(orderedArchivedFiles), nil
}

func (i *dataImporter) downloadUserCustomisationsForDataset(datasetID string, downloadPath string) error {
	// Download all files for the given dataset ID from user manual upload bucket/path
	// into downloadPath
	uploadedFiles, err := i.remoteFS.ListObjects(i.manualUploadBucket, path.Join(filepaths.DatasetCustomRoot, datasetID))
	if err != nil {
		return err
	}

	for _, uploadedPath := range uploadedFiles {
		fileName, middleDirs, err := importerutils.DecodeManualUploadPath(uploadedPath)

		if err != nil {
			return err
		}

		// We need to form a path starting at downloadPath that preserves the file structure of what's in the bucket
		// Here it forms something like <downloadPath>/file.png OR <downloadPath>/MATCHED/file.png
		parts := append([]string{downloadPath}, middleDirs...)
		parts = append(parts, fileName)
		savePath := path.Join(parts...)

		err = i.fetchFile(i.manualUploadBucket, uploadedPath, savePath)
		if err != nil {
			return err
		}
	}

	return nil
}

// We look at the Sol to work out if it's a test dataset or a real one, to specify a custom detector if needed
func getOverrideDetectorForSol(sol string, config importZipConfig) string {
	if config.Detector != "" {
		// Return a custom detector string.
		return config.Detector
	} else if sol[0] >= '0' && sol[0] <= '9' {
		// Usual Sol number and no custom string, don't override.
		return ""
	} else if sol[0] == 'D' || sol[0] == 'C' {
		return ""
	} else {
		// Sol starts with a character, non-standard, use the EM detector.
		return "PIXL-EM-E2E"
	}
}

func getDatasetGroup(detector string, config importZipConfig) string {
	if config.Group != "" {
		return config.Group
	} else if detector == "PIXL-EM-E2E" {
		return "PIXL-EM"
	} else {
		return "PIXL-FM"
	}
}

// NOT SURE why this function exists, was in the dataset importer - original one didn't fail if no files, so it seems
// like it's optional files that may be there...
func (i *dataImporter) copyAPIXToOutput(importPath string, outputPath string) error {
	// List all files in APIX dir
	apixPaths, err := i.localFS.ListObjects(importPath, "APIX")
	if err != nil {
		return err
	}

	for _, apixFile := range apixPaths {
		fileName := path.Base(apixFile)
		err = os.Rename(apixFile, path.Join(outputPath, fileName))
		if err != nil {
			return err
		}
	}

	return nil
}

/*
// copyAdditionalDirectories - Copy in additional directories
func (i *dataImporter) copyManuallyUploadedImageDirectories(importPath string, outputPath string) error {
	dirs := []string{"RGBU", "DISCO", "MATCHED"}
	for _, dir := range dirs {
		imgPath := path.Join(importPath, dir)
		if _, err := os.Stat(imgPath); !os.IsNotExist(err) {
			i.log.Infof("Copying %v to output directory...", dir)
			err := ccopy.Copy(imgPath, path.Join(outputPath, dir))

			if err != nil {
				return err
			}
		}
	}

	return nil
}
*/
// Copies files to bucket
// NOTE: Assumes flat list of files, no folder structure!
func (i *dataImporter) copyToBucket(datasetID string, sourcePath string, destBucket string, destPath string) error {
	var uploadError error

	err := filepath.Walk(sourcePath, func(sourcePath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(sourcePath)
			if err != nil {
				i.log.Errorf("Failed to read file for upload: %v", sourcePath)
				uploadError = err
			} else {
				uploadPath := path.Join(destPath, datasetID, path.Base(sourcePath))

				i.log.Infof("-Uploading: %v", sourcePath)
				i.log.Infof("---->to s3://%v/%v", destBucket, uploadPath)
				err = i.remoteFS.WriteObject(destBucket, uploadPath, data)

				if err != nil {
					i.log.Errorf("Failed to upload to s3://%v/%v: %v", destBucket, uploadPath, err)
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

type importZipConfig struct {
	Name       string `json:"name"`
	Detector   string `json:"detector"`
	Group      string `json:"group"`
	UpdateType string `json:"updateType"`
}

// Returns 3 things:
// Dataset name
// importer config file
// errors, if any
func (i *dataImporter) importDataFiles(importPath string, localRangesPath string, outputPath string) (string, importZipConfig, error) {
	// Read config
	var config importZipConfig
	err := i.localFS.ReadJSON("", path.Join(importPath, "config.json"), &config, false)
	if err != nil {
		return "", config, fmt.Errorf("Failed to load importer config file: %v", err)
	}

	// Determine what kind of dataset this is... pick an importer that will work
	importers := map[string]dataConverter.DataConverter{"test-msa": jplbreadboard.MSATestData{}, "pixl-fm": pixlfm.PIXLFM{}}
	importerName := "pixl-fm"

	// Run dataset converter to generate a dataset.bin and dataset summary, along with all required image files
	importer, ok := importers[importerName]
	if !ok {
		return "", config, errors.New("Importer not found: " + importerName)
	}

	i.log.Infof("Copying APIX files to output directory...")
	err = i.copyAPIXToOutput(importPath, outputPath)
	if err != nil {
		// TODO: remove this, or what? Old code wasn't failing on this...
		//return fmt.Errorf("Failed to copy APIX files to output directory: %v", err)
	}

	i.log.Infof("Running dataset converter...")
	data, contextImageSrcPath, err := importer.Import(importPath, localRangesPath, i.log)
	if err != nil {
		return "", config, fmt.Errorf("Import failed: %v", err)
	}

	// Apply any customisations/overrides:
	if len(config.Name) > 1 { // 1 for spaces?
		data.DatasetID = config.Name
	}

	overrideDetector := getOverrideDetectorForSol(data.Meta.SOL, config)
	if len(overrideDetector) > 0 {
		data.DetectorConfig = overrideDetector
	}

	data.Group = getDatasetGroup(data.DetectorConfig, config)

	// Form the output path
	outPath := path.Join(outputPath, data.DatasetID)

	i.log.Infof("Writing dataset file...")
	saver := output.PIXLISEDataSaver{}
	err = saver.Save(*data, contextImageSrcPath, outPath, time.Now().Unix(), i.log)
	if err != nil {
		return "", config, fmt.Errorf("Failed to write dataset file: %v. Error: %v", outPath, err)
	}

	i.log.Infof("Running diffraction DB generator...")
	err = createPeakDiffractionDB(path.Join(outPath, filepaths.DatasetFileName), path.Join(outPath, filepaths.DiffractionDBFileName), i.log)

	if err != nil {
		return "", config, fmt.Errorf("Failed to run diffraction DB generator. Error: %v", err)
	}

	/* Dataset converter does this already... writes to the wrong place anyway (output expected to be flat list of files, this creates ../MATCHED dir)
	i.log.Infof("Outputting manually uploaded images...")
	err = i.copyManuallyUploadedImageDirectories(importPath, outputPath)
	if err != nil {
		return config, fmt.Errorf("Error when copying manually-uploaded images directories: %v", err)
	}
	*/
	// Finally, copy the whole thing to our target bucket
	i.log.Infof("Copying generated dataset to bucket: %v...", i.datasetBucket)
	err = i.copyToBucket(data.DatasetID, outputPath, i.datasetBucket, filepaths.RootDatasets)
	if err != nil {
		return "", config, fmt.Errorf("Error when copying dataset to bucket: %v. Error: %v", i.datasetBucket, err)
	}

	return data.DatasetID, config, nil
}
