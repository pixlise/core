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

package dataConverter

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/dataset"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"

	datasetModel "github.com/pixlise/core/v2/core/dataset"
	datasetArchive "github.com/pixlise/core/v2/data-import/dataset-archive"
	"github.com/pixlise/core/v2/data-import/internal/data-converters/jplbreadboard"
	"github.com/pixlise/core/v2/data-import/internal/data-converters/pixlfm"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
	importerNotification "github.com/pixlise/core/v2/data-import/internal/notification"
	"github.com/pixlise/core/v2/data-import/output"
	diffractionDetection "github.com/pixlise/core/v2/diffraction-detector"
)

// All dataset conversions are started through here. This can contain multiple implementations
// for different scenarios, but internally it all runs the same way

// ImportFromArchive - Importing from dataset archive area. Calls ImportFromLocalFileSystem
// Returns:
// Dataset ID imported
// Error (if any)
func ImportDataset(
	localFS fileaccess.FileAccess,
	remoteFS fileaccess.FileAccess,
	configBucket string,
	manualUploadBucket string,
	datasetBucket string,
	datasetID string,
	log logger.ILogger,
	justArchived bool, // Set to true if a file was just saved to the archive prior to calling this. Affects notifications sent out
) (string, error) {

	workingDir, err := ioutil.TempDir("", "archive")
	if err != nil {
		return "", err
	}

	// Firstly, we download from the archive
	archive := datasetArchive.NewDatasetArchiveDownloader(remoteFS, localFS, log, datasetBucket, manualUploadBucket)
	localDownloadPath, localUnzippedPath, zipCount, err := archive.DownloadFromDatasetArchive(datasetID, workingDir)
	if err != nil {
		return "", err
	}

	// If no zip files were loaded, maybe this dataset is a manually uploaded one, try to import from there instead
	if zipCount == 0 {
		log.Infof("No zip files found in archive, dataset may have been manually uploaded. Trying to download...")
		localDownloadPath, localUnzippedPath, err = archive.DownloadFromDatasetUploads(datasetID, workingDir)
		if err != nil {
			return "", err
		}
	}

	localRangesPath, err := archive.DownloadPseudoIntensityRangesFile(configBucket, localDownloadPath)
	if err != nil {
		return "", err
	}

	log.Infof("Downloading user customisation files...")

	err = archive.DownloadUserCustomisationsForDataset(datasetID, localUnzippedPath)
	if err != nil {
		return "", err
	}

	// Now that we have data down, we can run the importer from local file system
	datasetIDImported, err := ImportFromLocalFileSystem(
		localFS,
		remoteFS,
		workingDir,
		localUnzippedPath,
		localRangesPath,
		datasetBucket,
		datasetID,
		log,
	)
	if err != nil {
		return "", err
	}

	// Decide what notifications (if any) to send
	err = sendNotificationsIfRequired(remoteFS, log, configBucket, datasetBucket, datasetIDImported, !justArchived && zipCount > 1)
	if err != nil {
		log.Errorf("Failed to send notification: %v", err)
	}

	return datasetIDImported, nil
}

// ImportFromLocalFileSystem - As the name says, imports from directory on local file system
// Returns:
// Dataset ID (in case it was modified during conversion)
// Error (if there was one)
func ImportFromLocalFileSystem(
	localFS fileaccess.FileAccess,
	remoteFS fileaccess.FileAccess, // For uploading result
	workingDir string, // Working dir, under which we may form our output dir
	localImportPath string, // Path on local file system with directory ready to import
	localPseudoIntensityRangesPath string, // Path on local file system
	datasetBucket string, // Where we import to
	datasetID string, // Dataset ID being imported. Some importers may need this, others (who have dataset ID in file names being imported) can verify it matches this expected one
	log logger.ILogger) (string, error) {

	// Pick an importer by inspecting the directory we're about to import from
	importer, err := selectImporter(localFS, localImportPath)

	if err != nil {
		return "", err
	}

	// Create an output directory
	outputPath, err := fileaccess.MakeEmptyLocalDirectory(workingDir, "output")

	if err != nil {
		return "", err
	}

	log.Infof("Running dataset converter...")
	data, contextImageSrcPath, err := importer.Import(localImportPath, localPseudoIntensityRangesPath, datasetID, log)
	if err != nil {
		return "", fmt.Errorf("Import failed: %v", err)
	}
	/*
		// Apply any customisations/overrides:
		if len(config.Name) > 1 { // 1 for spaces?
			data.DatasetID = config.Name
		}

		overrideDetector := getOverrideDetectorForSol(data.Meta.SOL)
		if len(overrideDetector) > 0 {
			data.DetectorConfig = overrideDetector
		}

		data.Group = getDatasetGroup(data.DetectorConfig)
	*/

	// Apply any overrides we may have
	customMetaFields, err := readLocalCustomMeta(log, localImportPath)
	if err != nil {
		return "", err
	} else if len(customMetaFields.Title) > 0 && customMetaFields.Title != " " {
		log.Infof("Applying custom title: %v", customMetaFields.Title)
		data.Meta.Title = customMetaFields.Title
	}

	// Form the output path
	outPath := path.Join(outputPath, data.DatasetID)

	log.Infof("Writing dataset file...")
	saver := output.PIXLISEDataSaver{}
	err = saver.Save(*data, contextImageSrcPath, outPath, time.Now().Unix(), log)
	if err != nil {
		return "", fmt.Errorf("Failed to write dataset file: %v. Error: %v", outPath, err)
	}

	log.Infof("Running diffraction DB generator...")
	err = createPeakDiffractionDB(path.Join(outPath, filepaths.DatasetFileName), path.Join(outPath, filepaths.DiffractionDBFileName), log)

	if err != nil {
		return "", fmt.Errorf("Failed to run diffraction DB generator. Error: %v", err)
	}

	// Finally, copy the whole thing to our target bucket
	log.Infof("Copying generated dataset to bucket: %v...", datasetBucket)
	err = copyToBucket(remoteFS, data.DatasetID, outputPath, datasetBucket, filepaths.RootDatasets, log)
	if err != nil {
		return "", fmt.Errorf("Error when copying dataset to bucket: %v. Error: %v", datasetBucket, err)
	}

	// NOTE: we also copy out the summary file to another location for it to be indexed by the dataset list generator
	summary := dataset.SummaryFileData{}
	localSummaryPath := path.Join(outPath, filepaths.DatasetSummaryFileName)
	err = localFS.ReadJSON(localSummaryPath, "", &summary, false)
	if err != nil {
		log.Errorf("Failed to find dataset summary file. Error: %v", err)
		// Don't die for this
	} else {
		err = remoteFS.WriteJSON(datasetBucket, filepaths.GetDatasetSummaryFilePath(datasetID), &summary)
		if err != nil {
			log.Errorf("Failed to write dataset summary file to summary location. Error: %v", err)
			// Don't die for this
		}
	}

	return data.DatasetID, nil
}

type DataConverter interface {
	Import(importJSONPath string, pseudoIntensityRangesPath string, datasetID string, jobLog logger.ILogger) (*dataConvertModels.OutputData, string, error)
}

// selectImporter - Looks in specified path and determines what importer to use
func selectImporter(localFS fileaccess.FileAccess, importPath string) (DataConverter, error) {
	// If we find a "config.json", assume it's a FM dataset from the pipeline
	pathType, err := pixlfm.DetectPIXLFMStructure(importPath)
	if len(pathType) > 0 && err == nil {
		// We know it's a PIXL FM type dataset... it'll later be determined which one
		return pixlfm.PIXLFM{}, nil
	}

	// Try to read a detector.json - manually uploaded datasets will contain this to direct our operation...
	detPath := path.Join(importPath, "detector.json")
	var detectorFile datasetArchive.DetectorChoice
	err = localFS.ReadJSON(detPath, "", &detectorFile, false)
	if err == nil {
		// We found it, work out based on what's in there
		if detectorFile.Detector == "JPL Breadboard" {
			return jplbreadboard.MSATestData{}, nil
		}
	}

	// TODO: Add other formats here!

	// Unknown
	return nil, errors.New("Failed to determine dataset type to import.")
}

// createPeakDiffractoinDB - Use the diffraction engine to calculate the diffraction peaks
func createPeakDiffractionDB(path string, savepath string, jobLog logger.ILogger) error {
	protoParsed, err := datasetModel.ReadDatasetFile(path)
	if err != nil {
		jobLog.Errorf("Failed to open dataset \"%v\": \"%v\"", path, err)
		return err
	}

	jobLog.Infof("  Opened %v, got RTT: %v, title: \"%v\". Scanning for diffraction peaks...", path, protoParsed.Rtt, protoParsed.Title)

	datasetPeaks, err := diffractionDetection.ScanDataset(protoParsed)
	if err != nil {
		jobLog.Errorf("Error Encoundered During Scanning: %v", err)
		return err
	}

	jobLog.Infof("  Completed scan successfully")

	if savepath != "" {
		jobLog.Infof("  Saving diffraction db file: %v", savepath)
		diffractionPB := diffractionDetection.BuildDiffractionProtobuf(protoParsed, datasetPeaks)
		err := diffractionDetection.SaveDiffractionProtobuf(diffractionPB, savepath)
		if err != nil {
			jobLog.Errorf("Error Encoundered During Saving: %v", err)
			return err
		}

		jobLog.Infof("  Diffraction db saved successfully")
	}

	return nil
}

// Copies files to bucket
// NOTE: Assumes flat list of files, no folder structure!
func copyToBucket(remoteFS fileaccess.FileAccess, datasetID string, sourcePath string, destBucket string, destPath string, log logger.ILogger) error {
	var uploadError error

	err := filepath.Walk(sourcePath, func(sourcePath string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(sourcePath)
			if err != nil {
				log.Errorf("Failed to read file for upload: %v", sourcePath)
				uploadError = err
			} else {
				uploadPath := path.Join(destPath, datasetID, path.Base(sourcePath))

				log.Infof("-Uploading: %v", sourcePath)
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

// Sends dataset import related notifications as required
func sendNotificationsIfRequired(remoteFS fileaccess.FileAccess, log logger.ILogger, configBucket string, datasetBucket string, datasetID string, isUpdate bool) error {
	// It worked! Trigger notifications
	log.Infof("Triggering Notifications...")
	/*if updateType != "trivial"*/ {
		updatenotificationtype, err := importerNotification.GetUpdateNotificationType(datasetID, datasetBucket, remoteFS)
		if err != nil {
			return err
		}

		ns := importerNotification.MakeNotificationStack(remoteFS, log)
		err = importerNotification.TriggerNotifications(configBucket, datasetID, remoteFS, isUpdate, updatenotificationtype, ns, log)
		if err != nil {
			return err
		}
	}
	return nil
}
