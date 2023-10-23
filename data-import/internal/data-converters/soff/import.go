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

package soff

import (
	"encoding/xml"
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	gdsfilename "github.com/pixlise/core/v3/data-import/gds-filename"
	"github.com/pixlise/core/v3/data-import/internal/dataConvertModels"
	"github.com/pixlise/core/v3/data-import/internal/importerutils"
)

type SOFFImport struct {
	log logger.ILogger
}

type identificationArea struct {
	Title        string `xml:"title"`
	ProductClass string `xml:"product_class"`
}

type fileAreaObservationalFile struct {
	FileName string `xml:"file_name"`
}

type fileAreaObservational struct {
	File           fileAreaObservationalFile `xml:"File"`
	TableDelimited []tableDelimited          `xml:"Table_Delimited"`
	EncodedImage   encodedImage              `xml:"Encoded_Image"`
}

type Offset struct {
	Unit  string `xml:"unit,attr"`
	Value int64  `xml:",chardata"`
}

type encodedImage struct {
	Offset Offset `xml:"offset"`
}

type tableDelimited struct {
	LocalIdentifier string `xml:"local_identifier"`
	Offset          Offset `xml:"offset"`
	Records         int64  `xml:"records"`
	Description     string `xml:"description"`
	RecordDelimiter string `xml:"record_delimiter"`
	FieldDelimiter  string `xml:"field_delimiter"`
	Fields          int64  `xml:"fields"`
	Groups          int64  `xml:"groups"`
}

type timeCoordinates struct {
	StartDateTime string `xml:"start_date_time"`
	StopDateTime  string `xml:"stop_date_time"`
}

type investigationArea struct {
	Name string `xml:"name"`
	Type string `xml:"type"`
}

type observingSystem struct {
	ObservingSystemComponents []investigationArea `xml:"Observing_System_Component"`
}

type observationArea struct {
	TimeCoordinates      timeCoordinates   `xml:"Time_Coordinates"`
	InvestigationArea    investigationArea `xml:"Investigation_Area"`
	ObservingSystem      observingSystem   `xml:"Observing_System"`
	TargetIdentification investigationArea `xml:"Target_Identification"`
}

type productObservational struct {
	IdentificationArea identificationArea      `xml:"Identification_Area"`
	FileArea           []fileAreaObservational `xml:"File_Area_Observational"`
	ObservationArea    observationArea         `xml:"Observation_Area"`
}

type fileOffset struct {
	fileName string
	offset   int64
}

func (s *SOFFImport) Import(importPath string, pseudoIntensityRangesPath string, datasetIDExpected string, jobLog logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	s.log = jobLog

	// Find ONE xml file
	soffFilePath, err := GetSOFFDescriptionFile(importPath)
	if err != nil {
		return nil, "", err
	}

	if len(soffFilePath) <= 0 {
		return nil, "", errors.New("No SOFF description file found in path: " + importPath)
	}

	soff, err := readSOFF(soffFilePath)
	if err != nil {
		return nil, "", err
	}

	// At this point we have the SOFF and know it's got the right tables, so import each table
	importPathAndOffsets := map[string]fileOffset{}

	imageFiles := []string{}

	for _, file := range soff.FileArea {
		for _, table := range file.TableDelimited {
			importPathAndOffsets[table.LocalIdentifier] = fileOffset{
				fileName: file.File.FileName,
				offset:   table.Offset.Value,
			}
		}

		// If there are images...
		if file.EncodedImage.Offset.Unit == "byte" {
			if file.EncodedImage.Offset.Value != 0 {
				return nil, "", fmt.Errorf("Image %v expected offset to be 0", file.File.FileName)
			}

			imageFiles = append(imageFiles, file.File.FileName)
		} else if len(file.EncodedImage.Offset.Unit) > 0 {
			return nil, "", fmt.Errorf("Unexpected image units: %v", file.EncodedImage.Offset.Unit)
		}
	}

	// Read each one
	beamLookup, err := importerutils.ReadBeamLocationsFile(filepath.Join(importPath, importPathAndOffsets["Xray_beam_positions"].fileName), true, 1, s.log)
	if err != nil {
		return nil, "", err
	}

	// NOTE: these are specified separately by the SOFF file. Currently we expect them to be in a single file
	// and ReadSpectraCSV knows how to break them up. It's possible the SOFF file may one day break this into
	// multiple files, at which point this will break unless we break this into 4 separate read functions!
	// For now, detect the situation and show an error so we find it if it does happen

	if importPathAndOffsets["histogram_housekeeping"].fileName != importPathAndOffsets["histogram_position"].fileName ||
		importPathAndOffsets["histogram_housekeeping"].fileName != importPathAndOffsets["histogram_A"].fileName ||
		importPathAndOffsets["histogram_housekeeping"].fileName != importPathAndOffsets["histogram_B"].fileName {
		return nil, "", errors.New("This parser only supports the histogram file containing all tables as per iSDS generated CSV files")
	}

	if importPathAndOffsets["histogram_B"].offset < importPathAndOffsets["histogram_A"].offset ||
		importPathAndOffsets["histogram_A"].offset < importPathAndOffsets["histogram_position"].offset ||
		importPathAndOffsets["histogram_position"].offset < importPathAndOffsets["histogram_housekeeping"].offset {
		return nil, "", errors.New("This parser expects the tables in the histogram file to be in order: housekeeping, position, A, B")
	}

	locSpectraLookup, err := importerutils.ReadSpectraCSV(filepath.Join(importPath, importPathAndOffsets["histogram_housekeeping"].fileName), s.log)
	if err != nil {
		return nil, "", err
	}

	specialHistogramFilePaths := []string{
		filepath.Join(importPath, importPathAndOffsets["bulk_sum_histogram"].fileName),
		filepath.Join(importPath, importPathAndOffsets["max_value_histogram"].fileName),
	}

	bulkMaxSpectraLookup, err := importerutils.ReadBulkMaxSpectra(specialHistogramFilePaths, s.log)
	if err != nil {
		return nil, "", err
	}

	hkData, err := importerutils.ReadHousekeepingFile(filepath.Join(importPath, importPathAndOffsets["housekeeping_frame"].fileName), 1, s.log)
	if err != nil {
		return nil, "", err
	}

	pseudoIntensityRanges, err := importerutils.ReadPseudoIntensityRangesFile(pseudoIntensityRangesPath, s.log)
	if err != nil {
		return nil, "", err
	}

	// "pseudointensity_map_metadata"
	pseudoIntensityData, err := importerutils.ReadPseudoIntensityFile(filepath.Join(importPath, importPathAndOffsets["pseudointensity_map"].fileName), false, s.log)
	if err != nil {
		return nil, "", err
	}

	contextImgsPerPMC := map[int32]string{}

	// Parse a PMC out of each image file, if there is one
	for _, imgFile := range imageFiles {
		upperImgFile := strings.ToUpper(imgFile)
		meta, err := gdsfilename.ParseFileName(upperImgFile)
		if err != nil {
			s.log.Errorf("Failed to parse image file name: \"%v\". Ignored.", imgFile)
		} else {
			pmc, err := meta.PMC()
			if err != nil {
				s.log.Infof("  WARNING: No PMC in context image file name: \"%v\"", imgFile)
			} else {
				contextImgsPerPMC[pmc] = imgFile
			}
		}
	}

	// We don't support reading RGBU and DISCO images from SOFF files
	rgbuImages := []dataConvertModels.ImageMeta{}
	discoImages := []dataConvertModels.ImageMeta{}
	matchedAlignedImages := []dataConvertModels.MatchedAlignedImageMeta{}
	/*
		localFS := &fileaccess.FSAccess{}
		matchedAlignedImages, err := importerutils.ReadMatchedImages(filepath.Join(importPath, "MATCHED"), beamLookup, s.log, localFS)
		if err != nil {
			return nil, "", err
		}
	*/
	// We now read the metadata from the housekeeping file name, as it's the only file we expect to always exist!
	housekeepingFileName := strings.ToUpper(importPathAndOffsets["housekeeping_frame"].fileName)
	housekeepingFileNameMeta, err := gdsfilename.ParseFileName(housekeepingFileName)
	if err != nil {
		return nil, "", fmt.Errorf("Failed to parse housekeeping file name: %v. Error: %v", housekeepingFileName, err)
	}

	data, err := importerutils.MakeFMDatasetOutput(
		beamLookup,
		hkData,
		locSpectraLookup,
		bulkMaxSpectraLookup,
		contextImgsPerPMC,
		pseudoIntensityData,
		pseudoIntensityRanges,
		matchedAlignedImages,
		rgbuImages,
		discoImages,
		"",
		housekeepingFileNameMeta,
		datasetIDExpected,
		"",
		"",
		jobLog,
	)

	return data, importPath, err
}

func GetSOFFDescriptionFile(importPath string) (string, error) {
	localFS := &fileaccess.FSAccess{}
	items, err := localFS.ListObjects(importPath, "")
	if err != nil {
		return "", err
	}

	for _, item := range items {
		if strings.HasSuffix(strings.ToLower(item), ".xml") {
			// We found an XML file, currently this is enough for us to assume SOFF.
			// NOTE: In future we may need to open it and verify, but for now none of our other importers see XML files!
			return filepath.Join(importPath, item), nil
		}
	}

	return "", nil
}

func readSOFF(xmlPath string) (*productObservational, error) {
	localFS := &fileaccess.FSAccess{}

	soffBytes, err := localFS.ReadObject(xmlPath, "")
	if err != nil {
		return nil, err
	}

	var soff productObservational
	err = xml.Unmarshal(soffBytes, &soff)
	if err != nil {
		return nil, err
	}

	err = isValidSOFF(soff)
	if err != nil {
		return nil, err
	}

	return &soff, err
}

func isValidSOFF(soff productObservational) error {
	// We're expecting the following Table_Delimited.LocalIdentifier to exist, if any don't, don't accept the file
	requiredTables := map[string]int{
		"housekeeping_frame":           0,
		"Xray_beam_positions":          0,
		"bulk_sum_histogram":           0,
		"max_value_histogram":          0,
		"histogram_housekeeping":       0,
		"histogram_position":           0,
		"histogram_A":                  0,
		"histogram_B":                  0,
		"pseudointensity_map_metadata": 0,
		"pseudointensity_map":          0,
	}

	for _, file := range soff.FileArea {
		for _, table := range file.TableDelimited {
			count, exists := requiredTables[table.LocalIdentifier]
			if !exists {
				return errors.New("Did not include table: " + table.LocalIdentifier)
			}
			if count > 0 {
				return errors.New("Duplicate table: " + table.LocalIdentifier)
			}

			if len(file.File.FileName) <= 0 {
				return errors.New("No file for table: " + table.LocalIdentifier)
			}

			// Increment
			requiredTables[table.LocalIdentifier]++
		}
	}

	// Ensure we encountered each table once
	for table, count := range requiredTables {
		if count < 1 {
			return errors.New("Missing table: " + table)
		}
	}

	return nil
}
