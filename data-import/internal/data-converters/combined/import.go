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

package combined

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	gdsfilename "github.com/pixlise/core/v2/data-import/gds-filename"
	converter "github.com/pixlise/core/v2/data-import/internal/data-converters/interface"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
)

type CombinedDatasetImport struct {
	selectImporter converter.SelectImporterFunc
}

func MakeCombinedDatasetImporter(selectImporter converter.SelectImporterFunc) CombinedDatasetImport {
	return CombinedDatasetImport{
		selectImporter: selectImporter,
	}
}

// This expects a directory with other datasets inside it, stored in directories named by RTT, along with coordinate CSV/image files which reference those directories.
// An example outer directory:
// CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-PE__0614_0721483865_000RFS__03011722129925170003___J05.csv  <--  Coordinates for dataset RTT=212992517 relative to image SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01
// CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-SS__0614_0721474508_785RRS__0301172SRLC11360W108CGNJ01.csv  <--  Coordinates for dataset ID=SRLC11360 relative to image SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01
// SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01.png  <--  The image(?)
// SIF_0614_0721455441_734EBY_N0301172SRLC00643_0000LMJ01.png  <--  A stand-in for the image, EBY not RAS... all I could get for testing quickly
// SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-PE__0614_0721483865_000RFS__03011722129925170003___J05.png  <--  Just an example image, not likely to exist in future, but shows the coords on the underlying image
// SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-SS__0614_0721474508_785RRS__0301172SRLC11360W108CGNJ01.png  <--  Just an example image, not likely to exist in future, but shows the coords on the underlying image
// 212992517/<PIXL FM or SOFF format dataset>
// SRLC11360/<SHERLOC dataset in SOFF format(?)>

func (cmb CombinedDatasetImport) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, log logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	localFS := &fileaccess.FSAccess{}

	log.Infof("Importing combined dataset...")
	fileNames, _ /*firstFileMeta*/, secondFileMeta, err := GetCombinedBeamFiles(importPath, log)
	if err != nil {
		return nil, "", err
	}

	for _, file := range fileNames {
		log.Infof(" Found: %v", file)
	}
	"CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-PE__0614_0721483865_000RFS__03011722129925170003___J05.csv"
	[1]:
	"CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-SS__0614_0721474508_785RRS__0301172SRLC11360W108CGNJ01.csv"
	[2]:
	"CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-SS__0614_0721475157_600RRS__0301172SRLC11360W208CGNJ01.csv"
	[3]:
	"CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-SS__0614_0721475935_830RRS__0301172SRLC11370W108CGNJ01.csv"
	[4]:
	"CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-SS__0614_0721476584_900RRS__0301172SRLC11370W208CGNJ01.csv"
	[5]:
	"CW-SIF_0614_0721455441_734RAS_N0301172SRLC00643_0000LMJ01-SS__0614_072147
	// Get CSVs and verify that the relevant dataset subdirs are present
	subDatasets := map[string]*dataConvertModels.OutputData{}

	for c, meta := range secondFileMeta {
		datasetID, err := meta.RTT()
		if err != nil {
			return nil, "", fmt.Errorf("Failed to read dataset RTT/ID from file name: %v", fileNames[c])
		}

		log.Infof("Checking directory for dataset: %v", datasetID)

		importSubdirPath := path.Join(importPath, datasetID)
		_, err = os.Stat(importSubdirPath)
		if err != nil {
			return nil, "", fmt.Errorf("Missing subdirectory for dataset RTT/ID: %v", datasetID)
		}

		if strings.HasPrefix(datasetID, "SRLC") {
			log.Infof("SKIPPING dataset read for: %v", datasetID)
		}

		// Read in the dataset
		log.Infof("Checking dataset type: %v", datasetID)
		importer, err := cmb.selectImporter(localFS, importSubdirPath, log)
		if err != nil {
			return nil, "", err
		}

		log.Infof("Reading dataset: %v", datasetID)
		output, _ /*datasetIDRead*/, err := importer.Import(importSubdirPath, pseudoIntensityRangesPath, datasetID, log)
		if err != nil {
			return nil, "", fmt.Errorf("Failed to import dataset RTT/ID: %v. Error: %v", datasetID, err)
		}

		// Save this for later
		subDatasets[datasetID] = output
	}

	return nil, "", nil
}

func GetCombinedBeamFiles(importPath string, log logger.ILogger) ([]string, []gdsfilename.FileNameMeta, []gdsfilename.FileNameMeta, error) {
	localFS := &fileaccess.FSAccess{}

	fileNames := []string{}
	firstFileMeta := []gdsfilename.FileNameMeta{}
	secondFileMeta := []gdsfilename.FileNameMeta{}

	items, err := localFS.ListObjects(importPath, "")
	if err != nil {
		return fileNames, firstFileMeta, secondFileMeta, err
	}

	// Expecting at least 1 CSV starting with CW-, which contains 2 valid file names embedded into it, - separated
	for _, item := range items {
		if strings.HasPrefix(strings.ToUpper(item), "CW-") && strings.HasSuffix(strings.ToLower(item), ".csv") {
			// Split it at - and verify the 2 parts parse
			parts := strings.Split(item, "-")
			if len(parts) != 3 {
				log.Infof("Failed to parse file name: %v", item)
				continue
			}
			firstFile, err := gdsfilename.ParseFileName(parts[1] + ".___")
			if err != nil {
				log.Infof("Failed to parse first part of file name: %v", item)
				continue
			}

			secondFile, err := gdsfilename.ParseFileName(parts[2])
			if err != nil {
				log.Infof("Failed to parse second part of file name: %v", item)
				continue
			}

			fileNames = append(fileNames, item)
			firstFileMeta = append(firstFileMeta, firstFile)
			secondFileMeta = append(secondFileMeta, secondFile)
		}
	}

	return fileNames, firstFileMeta, secondFileMeta, nil
}
