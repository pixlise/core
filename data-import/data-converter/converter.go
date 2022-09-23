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
	"os"
	"path"
	"time"

	"github.com/pixlise/core/v2/api/filepaths"
	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"

	datasetModel "github.com/pixlise/core/v2/core/dataset"
	datasetArchive "github.com/pixlise/core/v2/data-import/dataset-archive"
	"github.com/pixlise/core/v2/data-import/internal/data-converters/jplbreadboard"
	"github.com/pixlise/core/v2/data-import/internal/data-converters/pixlfm"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
	"github.com/pixlise/core/v2/data-import/output"
	diffractionDetection "github.com/pixlise/core/v2/diffraction-detector"
)

// All dataset conversions are started through here. This can contain multiple implementations
// for different scenarios, but internally it all runs the same way

// ImportFromArchive - Importing from dataset archive area. Calls ImportFromLocalFileSystem
func ImportFromArchive(
	localFS fileaccess.FileAccess,
	remoteFS fileaccess.FileAccess,
	configBucket string,
	manualUploadBucket string,
	datasetBucket string,
	datasetID string,
	log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	// Firstly, we download from the archive
	archive := datasetArchive.NewDatasetArchiveDownloader(remoteFS, localFS, log, datasetBucket, manualUploadBucket)
	archive.DownloadFromDatasetArchive(datasetID)
}

// ImportFromManualUpload - Importing from manually uploaded area. Calls ImportFromLocalFileSystem
//func ImportFromManualUpload(datasetBucket string, datasetID string, jobLog logger.ILogger) (*dataConvertModels.OutputData, string, error) {
//}

// ImportFromLocalFileSystem - As the name says, imports from directory on local file system
// Returns the dataset ID (in case it was modified during conversion), and an error if there was one
func ImportFromLocalFileSystem(
	localFS fileaccess.FileAccess,
	remoteFS fileaccess.FileAccess, // For uploading result
	localImportPath string, // Path on local file system with directory ready to import
	localPseudoIntensityRangesPath string, // Path on local file system
	datasetBucket string, // Where we import to
	log logger.ILogger) (string, error) {

	// Pick an importer by inspecting the directory we're about to import from
	importer, err := selectImporter(localImportPath)

	if err != nil {
		return "", err
	}

	// Create an output directory
	outputPath := fileaccess.MakeEmptyLocalDirectory(path.Dir(localImportPath)), "output")

	log.Infof("Running dataset converter...")
	data, contextImageSrcPath, err := importer.Import(localImportPath, localPseudoIntensityRangesPath, log)
	if err != nil {
		return "", fmt.Errorf("Import failed: %v", err)
	}

	// Apply any customisations/overrides:
	if len(config.Name) > 1 { // 1 for spaces?
		data.DatasetID = config.Name
	}

	overrideDetector := getOverrideDetectorForSol(data.Meta.SOL)
	if len(overrideDetector) > 0 {
		data.DetectorConfig = overrideDetector
	}

	data.Group = getDatasetGroup(data.DetectorConfig)

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
	err = copyToBucket(data.DatasetID, outputPath, datasetBucket, filepaths.RootDatasets)
	if err != nil {
		return "", fmt.Errorf("Error when copying dataset to bucket: %v. Error: %v", i.datasetBucket, err)
	}

	return data.DatasetID, nil
}

type DataConverter interface {
	Import(importJSONPath string, pseudoIntensityRangesPath string, jobLog logger.ILogger) (*dataConvertModels.OutputData, string, error)
}

// selectImporter - Looks in specified path and determines what importer to use
func selectImporter(importPath string) (DataConverter, error) {
	// If we find a "config.json", assume it's a FM dataset from the pipeline
	_, err := os.Stat(path.Join(importPath, "config.json"))
	if err != nil {
		return pixlfm.PIXLFM{}, nil
	}

	// If we find an "import.json", assume it's a JPL breadboard dataset
	_, err = os.Stat(path.Join(importPath, "import.json"))
	if err != nil {
		return jplbreadboard.MSATestData{}, nil
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

	jobLog.Infof("  Opened %v, got RTT: %v, title %v. Scanning for diffraction peaks...", path, protoParsed.Rtt, protoParsed.Title)

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
