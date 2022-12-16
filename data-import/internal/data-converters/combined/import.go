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
	"sort"
	"strconv"
	"strings"

	"github.com/pixlise/core/v2/core/fileaccess"
	"github.com/pixlise/core/v2/core/logger"
	gdsfilename "github.com/pixlise/core/v2/data-import/gds-filename"
	converter "github.com/pixlise/core/v2/data-import/internal/data-converters/interface"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
	"github.com/pixlise/core/v2/data-import/internal/importerutils"
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
	combinedBeamFiles, imageFileNames, _ /*firstFileMeta*/, secondFileMeta, err := GetCombinedBeamFiles(importPath, log)
	if err != nil {
		return nil, "", err
	}

	for _, file := range combinedBeamFiles {
		log.Infof(" Found: %v", file)
	}

	subDatasets := map[string]*dataConvertModels.OutputData{}

	for c, meta := range secondFileMeta {
		datasetID, err := meta.RTT()
		if err != nil {
			return nil, "", fmt.Errorf("Failed to read dataset RTT/ID from file name: %v", combinedBeamFiles[c])
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

	// Now we combine the datasets
	data, datasetIDs, offsets, err := combineDatasets(subDatasets, log)
	if err != nil {
		return nil, "", err
	}

	// Finally, we inject our "combined" beam location info and images
	err = injectCombinedImagery(importPath, combinedBeamFiles, imageFileNames, secondFileMeta, datasetIDs, offsets, data, log)
	contextImagePath := importPath // We expect images to be in the same dir we're reading the CSVs from

	return data, contextImagePath, err
}

func GetCombinedBeamFiles(importPath string, log logger.ILogger) ([]string, []string, []gdsfilename.FileNameMeta, []gdsfilename.FileNameMeta, error) {
	localFS := &fileaccess.FSAccess{}

	fileNames := []string{}
	imageFileNames := []string{}
	firstFileMeta := []gdsfilename.FileNameMeta{}
	secondFileMeta := []gdsfilename.FileNameMeta{}

	items, err := localFS.ListObjects(importPath, "")
	if err != nil {
		return fileNames, imageFileNames, firstFileMeta, secondFileMeta, err
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
			imageFileNames = append(imageFileNames, parts[1]+".png")
			firstFileMeta = append(firstFileMeta, firstFile)
			secondFileMeta = append(secondFileMeta, secondFile)
		}
	}

	return fileNames, imageFileNames, firstFileMeta, secondFileMeta, nil
}

func combineDatasets(datasets map[string]*dataConvertModels.OutputData, log logger.ILogger) (*dataConvertModels.OutputData, []string, []int32, error) {
	resultDatasetID := ""
	resultTitle := ""
	detectorConfig := "PIXL"
	group := "PIXL-FM"

	minSiteID := int32(0)
	minSCLK := int32(0)
	sol := ""
	minDriveID := int32(0)

	sourceMetas := []dataConvertModels.FileMetaData{}

	pmcSourceRTTs := map[int32]string{}

	pseudoIntensityRanges := []dataConvertModels.PseudoIntensityRange{}

	beamLookup := dataConvertModels.BeamLocationByPMC{}

	hkHeaderIndexes := map[string]int{}
	hkData := dataConvertModels.HousekeepingData{
		Header:           []string{},
		Data:             map[int32][]dataConvertModels.MetaValue{},
		PerPMCHeaderIdxs: map[int32][]int32{},
	}

	spectraLookup := dataConvertModels.DetectorSampleByPMC{}

	contextImgsPerPMC := map[int32]string{}

	var pseudoIntensityData dataConvertModels.PseudoIntensities = nil

	// Order by PMC ranges, we want to have the first datasets PMCs as they are, then subsequent scans starting at 10k boundaries
	// so preferably we may get PMCs at: 1st scan < 10k, second 10k-20k, 3rd scan 30-40k etc.
	datasetIDs, offsets := makeDatasetPMCOffsets(datasets)

	// Run through and form a list of ALL points
	first := true
	for c, datasetID := range datasetIDs {
		offset := offsets[c]
		dataset := datasets[datasetID]

		// Add to housekeeping header strings
		for _, header := range dataset.HousekeepingHeaders {
			_, ok := hkHeaderIndexes[header]
			if !ok {
				hkHeaderIndexes[header] = len(hkData.Header)
				hkData.Header = append(hkData.Header, header)
			}
		}

		if !first {
			if dataset.Meta.SiteID < minSiteID {
				minSiteID = dataset.Meta.SiteID
			}

			if dataset.Meta.SCLK < minSCLK {
				minSCLK = dataset.Meta.SCLK
			}

			/*if dataset.Meta.SOL < minSol {
				minSol = dataset.Meta.SOL
			}*/

			if dataset.Meta.DriveID < minDriveID {
				minDriveID = dataset.Meta.DriveID
			}

			resultDatasetID += "_"
			resultTitle += "+"
		} else {
			minSiteID = dataset.Meta.SiteID
			minSCLK = dataset.Meta.SCLK
			sol = dataset.Meta.SOL
			minDriveID = dataset.Meta.DriveID
			pseudoIntensityRanges = dataset.PseudoRanges
		}

		resultDatasetID += datasetID
		resultTitle += dataset.Meta.Title

		sourceMetas = append(sourceMetas, dataset.Meta)

		for pmc, pmcData := range dataset.PerPMCData {
			pmcOffset := pmc + offset

			// Store it with its source
			pmcSourceRTTs[pmcOffset] = datasetID

			if len(pmcData.ContextImageSrc) > 0 {
				contextImgsPerPMC[pmcOffset] = pmcData.ContextImageSrc
			}

			// Pseudo-intensities (only init if needed)
			if len(pmcData.PseudoIntensities) > 0 {
				if pseudoIntensityData == nil {
					pseudoIntensityData = dataConvertModels.PseudoIntensities{}
				}
				pseudoIntensityData[pmcOffset] = pmcData.PseudoIntensities
			}

			// Beam location - optional in what we're reading, only add if exists
			if pmcData.Beam != nil {
				beamLookup[pmcOffset] = *pmcData.Beam
			}

			// Spectra for this PMC
			if pmcData.DetectorSpectra != nil {
				spectraLookup[pmcOffset] = pmcData.DetectorSpectra
			}

			// Housekeeping data, converting from datasets own lookup to our combined lookup (for header string)
			if len(pmcData.Housekeeping) > 0 {
				hkData.Data[pmcOffset] = []dataConvertModels.MetaValue{}

				// Run through each housekeeping value, looking up the new index (into the housekeeping header strings) and store it
				for i, val := range pmcData.Housekeeping {
					headerName := dataset.HousekeepingHeaders[i]
					newIdx, ok := hkHeaderIndexes[headerName]
					if !ok {
						log.Errorf("Failed to find index of housekeeping header: %v", headerName)
					} else {
						hkData.Data[pmcOffset] = append(hkData.Data[pmcOffset], val)

						// NOTE: here we store these indexes, because different files may reference different housekeeping value names or they
						// might be in a different order. This allows us to specify it directly to the output code that writes to the protobuf file
						hkData.PerPMCHeaderIdxs[pmcOffset] = append(hkData.PerPMCHeaderIdxs[pmcOffset], int32(newIdx))
					}
				}
			}
		}

		first = false
	}

	meta := dataConvertModels.FileMetaData{
		RTT:      resultDatasetID,
		SCLK:     minSCLK,
		SOL:      sol,
		SiteID:   minSiteID,
		Site:     "",
		DriveID:  minDriveID,
		TargetID: "",
		Target:   "",
		Title:    "Combined " + resultTitle,
	}

	data := &dataConvertModels.OutputData{
		DatasetID:      resultDatasetID,
		Group:          group,
		Meta:           meta,
		Sources:        sourceMetas,
		DetectorConfig: detectorConfig,
		//BulkQuantFile: "", <-- no bulk quant for tactical...
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		RGBUImages:           []dataConvertModels.ImageMeta{},
		DISCOImages:          []dataConvertModels.ImageMeta{},
		MatchedAlignedImages: []dataConvertModels.MatchedAlignedImageMeta{},
	}

	data.SetPMCData(beamLookup, hkData, spectraLookup, contextImgsPerPMC, pseudoIntensityData, pmcSourceRTTs)
	if data.DefaultContextImage != "" {
		log.Infof("Setting context image to: " + data.DefaultContextImage)
	}

	return data, datasetIDs, offsets, nil
}

type datasetPMCRange struct {
	minPMC         int32
	maxPMC         int32
	datasetID      string
	assignedOffset int32
}

func makeDatasetPMCOffsets(datasets map[string]*dataConvertModels.OutputData) ([]string, []int32) {
	pmcRanges := []datasetPMCRange{}

	for datasetID, dataset := range datasets {
		saveRange := datasetPMCRange{
			minPMC:    int32(0x7fffffff),
			maxPMC:    int32(0),
			datasetID: datasetID,
		}

		for pmc := range dataset.PerPMCData {
			if pmc < saveRange.minPMC {
				saveRange.minPMC = pmc
			}
			if pmc > saveRange.maxPMC {
				saveRange.maxPMC = pmc
			}
		}

		pmcRanges = append(pmcRanges, saveRange)
	}

	// Now sort them
	sort.Slice(pmcRanges, func(i, j int) bool {
		return pmcRanges[i].minPMC < pmcRanges[j].minPMC
	})

	// Now add offsets of 10,000 each, but increase as needed
	for c, item := range pmcRanges {
		if c > 0 {
			prevOffset := pmcRanges[c-1].assignedOffset

			// Set the offset
			item.assignedOffset = prevOffset + 10000

			// Make sure there is no overlap now between item and the last one
			for pmcRanges[c-1].maxPMC+prevOffset > item.minPMC+item.assignedOffset {
				item.assignedOffset += 10000
			}

			// Save it back in
			pmcRanges[c] = item
		}
	}

	resultDatasetIDs := []string{}
	resultOffsets := []int32{}

	for _, item := range pmcRanges {
		resultDatasetIDs = append(resultDatasetIDs, item.datasetID)
		resultOffsets = append(resultOffsets, item.assignedOffset)
	}

	return resultDatasetIDs, resultOffsets
}

func injectCombinedImagery(importPath string, combinedBeamFiles []string, imageFileNames []string, beamMeta []gdsfilename.FileNameMeta, datasetIDs []string, pmcOffsets []int32, combinedData *dataConvertModels.OutputData, log logger.ILogger) error {
	// Verify each image exists and read all coordinates for that image into our PMCs
	for c, imageFile := range imageFileNames {
		_, err := os.Stat(path.Join(importPath, imageFile))
		if err != nil {
			return fmt.Errorf("File not found: %v. Error: %v", imageFile, err)
		}

		datasetID, err := beamMeta[c].RTT()
		if err != nil {
			return fmt.Errorf("Error getting RTT for sub-dataset of: %v. Error: %v", combinedBeamFiles[c], err)
		}

		// Find the PMC offset for this dataset...
		pmcOffset := int32(0)
		found := false
		for i, id := range datasetIDs {
			if id == datasetID {
				pmcOffset = pmcOffsets[i]
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("Failed to find PMC offset for dataset: %v", datasetID)
		}

		err = readBeamCoordinates(path.Join(importPath, combinedBeamFiles[c]), datasetID, pmcOffset, combinedData, log)
		if err != nil {
			return err
		}
	}

	// Ensure output file has the combined image set
	combinedData.DefaultContextImage = imageFileNames[0]

	// Also, for now, clear all other images
	for pmc, data := range combinedData.PerPMCData {
		data.ContextImageSrc = ""
		data.ContextImageDst = ""

		// Ensure nothing has "other" beam coords
		if data.Beam != nil && len(data.Beam.IJ) > 1 {
			return fmt.Errorf("No beam found for PMC: %v", pmc)
		}
	}

	// Add a new fake PMC 0 which has the context image set as its image
	pmc0, ok := combinedData.PerPMCData[0]
	if ok {
		if len(pmc0.ContextImageSrc) > 0 || len(pmc0.ContextImageDst) > 0 {
			return fmt.Errorf("Error: Dataset already has an unexpected PMC 0 with context image attached: %v", pmc0)
		}
	} else {
		combinedData.PerPMCData[0] = &dataConvertModels.PMCData{}
		pmc0 = combinedData.PerPMCData[0]
	}

	pmc0.ContextImageSrc = combinedData.DefaultContextImage
	pmc0.ContextImageDst = combinedData.DefaultContextImage

	return nil
}

func readBeamCoordinates(filePath string, datasetRTT string, pmcOffset int32, combinedData *dataConvertModels.OutputData, log logger.ILogger) error {
	// Read all lines, match to beam PMC and add IJ for it
	// NOTE: these csv files don't come with column headers
	fields, err := importerutils.ReadCSV(filePath, -1, ',', log)
	if err != nil {
		return err
	}

	// We've got the fields, parse them, assuming column 0 is PMC, followed by i, j relative to the image
	for row, lineFields := range fields {
		if len(lineFields) != 3 {
			return fmt.Errorf("Row %v: Did not contain 3 fields\n", row)
		}

		pmc, err := strconv.Atoi(lineFields[0])
		if err != nil {
			return fmt.Errorf("Row %v: Expected integer for first field (PMC), got: %v\n", row, lineFields[0])
		}

		i, err := strconv.ParseFloat(lineFields[1], 64)
		if err != nil {
			return fmt.Errorf("Row %v: Expected integer for second field (i), got: %v\n", row, lineFields[1])
		}

		j, err := strconv.ParseFloat(lineFields[2], 64)
		if err != nil {
			return fmt.Errorf("Row %v: Expected integer for second field (j), got: %v\n", row, lineFields[2])
		}

		// Look up the PMC, if doesn't exist, bail
		data, ok := combinedData.PerPMCData[int32(pmc)+pmcOffset]
		if !ok {
			return fmt.Errorf("Row %v: PMC %v does not exist in dataset(s)\n", row, pmc)
		}

		if data.Beam == nil {
			return fmt.Errorf("Row %v: PMC %v does have any beam data in dataset(s)\n", row, pmc)
		}

		data.Beam.IJ = map[int32]dataConvertModels.BeamLocationProj{
			0: {
				I: float32(i),
				J: float32(j),
			},
		}
	}

	return nil
}
