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

// Exposes the interface of the dataset importer aka converter and selecting one automatically based on what
// files are in the folder being imported. The converter supports various formats as delivered by GDS or test
// instruments and this is inteded to be extendable further to other lab instruments and devices in future.
package dataConverter

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"time"

	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"

	datasetModel "github.com/pixlise/core/v3/core/dataset"
	datasetArchive "github.com/pixlise/core/v3/data-import/dataset-archive"
	"github.com/pixlise/core/v3/data-import/internal/data-converters/combined"
	converter "github.com/pixlise/core/v3/data-import/internal/data-converters/interface"
	"github.com/pixlise/core/v3/data-import/internal/data-converters/jplbreadboard"
	"github.com/pixlise/core/v3/data-import/internal/data-converters/pixlfm"
	"github.com/pixlise/core/v3/data-import/internal/data-converters/soff"
	"github.com/pixlise/core/v3/data-import/output"
	diffractionDetector "github.com/pixlise/diffraction-peak-detection/v2/detection"
)

// All dataset conversions are started through here. This can contain multiple implementations
// for different scenarios, but internally it all runs the same way

// ImportFromArchive - Importing from dataset archive area. Calls ImportFromLocalFileSystem
// Returns:
// WorkingDir
// Saved dataset summary structure
// What changed (as a string), so caller can know what kind of notification to send (if any)
// IsUpdate flag
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
) (string, datasetModel.SummaryFileData, string, bool, error) {

	savedSummary := datasetModel.SummaryFileData{}

	workingDir, err := ioutil.TempDir("", "archive")
	if err != nil {
		return workingDir, savedSummary, "", false, err
	}

	// Read previously saved dataset summary file, so we have something to compare against to see what changes
	// we will need to notify on
	oldSummary, errOldSummary := datasetModel.ReadDataSetSummary(remoteFS, datasetBucket, datasetID)
	if err != nil {
		// NOTE: we don't die here, we may be importing for the first time! Just log and continue
		//return workingDir, savedSummary, "", false, err
		log.Infof("Failed to import previous dataset summary file - assuming we're a new dataset...")
	}

	// Firstly, we download from the archive
	archive := datasetArchive.NewDatasetArchiveDownloader(remoteFS, localFS, log, datasetBucket, manualUploadBucket)
	localDownloadPath, localUnzippedPath, zipCount, err := archive.DownloadFromDatasetArchive(datasetID, workingDir)
	if err != nil {
		return workingDir, savedSummary, "", false, err
	}

	// If no zip files were loaded, maybe this dataset is a manually uploaded one, try to import from there instead
	if zipCount == 0 {
		log.Infof("No zip files found in archive, dataset may have been manually uploaded. Trying to download...")
		localDownloadPath, localUnzippedPath, err = archive.DownloadFromDatasetUploads(datasetID, workingDir)
		if err != nil {
			return workingDir, savedSummary, "", false, err
		}
	}

	// No obvious place to make this change right now, but pseudo-intensities have changed in flight software
	// and this is likely to go live in late 2023.
	pseudoVersion := ""
	/*if datasetID > ... {
		pseudoVersion = "-2023"
	}*/

	localRangesPath, err := archive.DownloadPseudoIntensityRangesFile(configBucket, localDownloadPath, pseudoVersion)
	if err != nil {
		return workingDir, savedSummary, "", false, err
	}

	log.Infof("Downloading user customisation files...")

	err = archive.DownloadUserCustomisationsForDataset(datasetID, localUnzippedPath)
	if err != nil {
		return workingDir, savedSummary, "", false, err
	}

	// Now that we have data down, we can run the importer from local file system
	_, err = ImportFromLocalFileSystem(
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
		return workingDir, savedSummary, "", false, err
	}

	// Decide what notifications (if any) to send
	updatenotificationtype := "unknown"

	if errOldSummary == nil { // don't do this if the old summary couldn't be read!
		savedSummary, err = datasetModel.ReadDataSetSummary(remoteFS, datasetBucket, datasetID)
		if err != nil {
			return workingDir, savedSummary, "", false, err
		}

		updatenotificationtype, err = getUpdateType(savedSummary, oldSummary)
		if err != nil {
			return workingDir, savedSummary, "", false, err
		}
	}

	return workingDir, savedSummary, updatenotificationtype, !justArchived && zipCount > 1, err
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
	importer, err := SelectImporter(localFS, remoteFS, datasetBucket, localImportPath, log)

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

	// Apply any overrides we may have
	customMetaFields, err := readLocalCustomMeta(log, localImportPath)
	if err != nil {
		return "", err
	}

	if len(customMetaFields.Title) > 0 && customMetaFields.Title != " " {
		log.Infof("Applying custom title: %v", customMetaFields.Title)
		data.Meta.Title = customMetaFields.Title
	}

	if len(customMetaFields.DefaultContextImage) > 0 {
		log.Infof("Applying custom default context image: %v", customMetaFields.DefaultContextImage)
		data.DefaultContextImage = customMetaFields.DefaultContextImage
	}

	// Form the output path
	outPath := filepath.Join(outputPath, data.DatasetID)

	log.Infof("Writing dataset file...")
	saver := output.PIXLISEDataSaver{}
	err = saver.Save(*data, contextImageSrcPath, outPath, time.Now().Unix(), log)
	if err != nil {
		return "", fmt.Errorf("Failed to write dataset file: %v. Error: %v", outPath, err)
	}

	log.Infof("Running diffraction DB generator...")
	err = createPeakDiffractionDB(filepath.Join(outPath, filepaths.DatasetFileName), filepath.Join(outPath, filepaths.DiffractionDBFileName), log)

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
	localSummaryPath := filepath.Join(outPath, filepaths.DatasetSummaryFileName)
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

// SelectImporter - Looks in specified path and determines what importer to use. Requires remoteFS for new case of importing combined
// datasets where it may need to download other files to complete the job
func SelectImporter(localFS fileaccess.FileAccess, remoteFS fileaccess.FileAccess, datasetBucket string, importPath string, log logger.ILogger) (converter.DataConverter, error) {
	// Check if it's a combined dataset
	combinedFiles, _ /*imageFileNames*/, _ /*combinedFile1Meta*/, _ /*combinedFile2Meta*/, err := combined.GetCombinedBeamFiles(importPath, log)
	if len(combinedFiles) > 0 && err == nil {
		// It's a combined dataset, interpret it as such
		return combined.MakeCombinedDatasetImporter(SelectImporter, remoteFS, datasetBucket), nil
	}

	// Check if it's a PIXL FM style dataset
	pathType, err := pixlfm.DetectPIXLFMStructure(importPath)
	if len(pathType) > 0 && err == nil {
		// We know it's a PIXL FM type dataset... it'll later be determined which one
		return pixlfm.PIXLFM{}, nil
	}

	// Check if it's SOFF
	soffFile, err := soff.GetSOFFDescriptionFile(importPath)
	if err != nil {
		return nil, err
	}

	if len(soffFile) > 0 {
		return &soff.SOFFImport{}, nil
	}

	// Try to read a detector.json - manually uploaded datasets will contain this to direct our operation...
	detPath := filepath.Join(importPath, "detector.json")
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

	datasetPeaks, err := diffractionDetector.ScanDataset(protoParsed)
	if err != nil {
		jobLog.Errorf("Error Encoundered During Scanning: %v", err)
		return err
	}

	jobLog.Infof("  Completed scan successfully")

	if savepath != "" {
		jobLog.Infof("  Saving diffraction db file: %v", savepath)
		diffractionPB := diffractionDetector.BuildDiffractionProtobuf(protoParsed, datasetPeaks)
		err := diffractionDetector.SaveDiffractionProtobuf(diffractionPB, savepath)
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
				sourceFile := filepath.Base(sourcePath)
				uploadPath := path.Join(destPath, datasetID, sourceFile)

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

func getUpdateType(newSummary datasetModel.SummaryFileData, oldSummary datasetModel.SummaryFileData) (string, error) {
	diff, err := output.SummaryDiff(newSummary, oldSummary)
	if err != nil {
		return "unknown", err
	}
	if diff.MaxSpectra > 0 || diff.BulkSpectra > 0 || diff.DwellSpectra > 0 || diff.NormalSpectra > 0 {
		return "spectra", nil
	} else if diff.ContextImages > 0 {
		return "image", nil
	} else if diff.DriveID > 0 || diff.Site != "" || diff.Target != "" || diff.Title != "" {
		return "housekeeping", nil
	}
	return "unknown", nil
}
