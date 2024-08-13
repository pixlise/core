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

package jplbreadboard

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"

	dataImportHelpers "github.com/pixlise/core/v4/api/dataimport/dataimportHelpers"
	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/api/dataimport/internal/importerutils"
	dataimportModel "github.com/pixlise/core/v4/api/dataimport/models"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

type MSATestData struct {
}

// Import - Implementing Importer interface, expects importPath to point to a directory containing importable files, with an import.json
//
//	containing fields specific to this importer
func (m MSATestData) Import(importPath string, pseudoIntensityRangesPath string, datasetID string, jobLog logger.ILogger) (*dataConvertModels.OutputData, string, error) {
	localFS := &fileaccess.FSAccess{}

	// Check if we can load the import instructions JSON file
	var params dataimportModel.BreadboardImportParams
	err := localFS.ReadJSON(importPath, "import.json", &params, false)
	if err != nil {
		// If there is no import.json file, we can use some suitable defaults, so just warn here
		//return nil, "", err
		jobLog.Infof("Warning: No import.json found, defaults will be used")

		// Set defaults
		params.MsaDir = "spectra" // We now assume we will have a spectra.zip extracted into a spectra dir!
		params.MsaBeamParams = "10,0,10,0"
		params.GenBulkMax = true
		params.GenPMCs = true
		params.ReadTypeOverride = "Normal"
		params.DetectorConfig = "Breadboard"
		params.Group = "JPL Breadboard"
		params.TargetID = "0"
		params.SiteID = 0

		// The rest we set to the dataset ID
		params.DatasetID = datasetID
		//params.Site = datasetID
		//params.Target = datasetID
		params.Title = datasetID
	} else {
		if len(params.DatasetID) <= 0 {
			return nil, "", errors.New("Import parameter file did not specify a DatasetID")
		}
		if len(params.Group) <= 0 {
			return nil, "", errors.New("Import parameter file did not specify a Group")
		}
		// Ensure expected ID matches what we're given
		if params.DatasetID != datasetID {
			return nil, "", fmt.Errorf("Expected dataset ID %v, read %v", datasetID, params.DatasetID)
		}
	}

	// Process the context images first, so if we're assigning an image to a PMC, we know what PMC it is
	// NOTE: this only matters for msa test datasets where the context image file names contain the PMC. In the
	// case where there is no PMC info in the file name, we just assign it to PMC 1 anyway.
	contextImgsPerPMC := map[int32]string{}
	contextImageSrcDir := ""
	if len(params.ContextImgDir) > 0 {
		contextImageSrcDir = filepath.Join(importPath, params.ContextImgDir)
		contextImgsPerPMC, err = processContextImages(contextImageSrcDir, jobLog, localFS)
		if err != nil {
			return nil, "", err
		}
	}

	minContextPMC := getMinimumContextPMC(contextImgsPerPMC)

	var hkData dataConvertModels.HousekeepingData
	var beamLookup = make(dataConvertModels.BeamLocationByPMC)

	if params.MsaBeamParams == "" && params.BeamFile != "" {
		jobLog.Infof("  Reading Beam Locations: \"%v\", using minimum context image PMC detected: %v\n", params.BeamFile, minContextPMC)
		beamLookup, err = dataImportHelpers.ReadBeamLocationsFile(filepath.Join(importPath, params.BeamFile), false, minContextPMC, []string{}, jobLog)
		if err != nil {
			return nil, "", err
		}
	} else if params.MsaBeamParams != "" {
		//beamLookup = dataConvertModels.BeamLocations{}
	}

	// Housekeeping file
	if params.HousekeepingFile != "" {
		jobLog.Infof("  Reading Housekeeping: %v\n", params.HousekeepingFile)
		hkData, err = importerutils.ReadHousekeepingFile(filepath.Join(importPath, params.HousekeepingFile), 0, jobLog)
		if err != nil {
			return nil, "", err
		}

		//logIfMoreFound(m, "housekeeping", 1)
	}

	// Pseudointensity data - if we have the CSV, load it and ranges file, otherwise nothing
	var pseudoIntensityRanges []dataConvertModels.PseudoIntensityRange = nil
	var pseudoIntensityData dataConvertModels.PseudoIntensities = nil
	if len(params.PseudoIntensityCSVPath) > 0 {
		// Can't have data & no range description...
		if len(pseudoIntensityRangesPath) <= 0 {
			return nil, "", errors.New("If passing pseudo-intensity CSV file, pseudo-intensity ranges file must also be provided")
		}

		pseudoIntensityRanges, err = importerutils.ReadPseudoIntensityRangesFile(pseudoIntensityRangesPath, jobLog)
		if err != nil {
			return nil, "", err
		}
		pseudoIntensityData, err = importerutils.ReadPseudoIntensityFile(filepath.Join(importPath, params.PseudoIntensityCSVPath), false, jobLog)
		if err != nil {
			return nil, "", err
		}
	}

	spectraPath := filepath.Join(importPath, params.MsaDir)
	allMSAFiles, err := localFS.ListObjects(spectraPath, "")
	if err != nil {
		return nil, "", err
	}

	verifyreadtype := true
	if params.ReadTypeOverride != "" {
		verifyreadtype = false
	}

	jobLog.Infof("  Reading %v files from spectrum directory...", len(allMSAFiles))
	spectrafiles, _ := getSpectraFiles(allMSAFiles, verifyreadtype, jobLog)

	jobLog.Infof("  Found %v usable spectrum files...", len(allMSAFiles))
	spectraLookup, err := makeSpectraLookup(spectraPath, spectrafiles, params.SingleDetectorMSAs, params.GenPMCs, params.ReadTypeOverride, params.DetectorADuplicate, jobLog)
	if err != nil {
		return nil, "", err
	}

	err = eVCalibrationOverride(&spectraLookup, params.XPerChanA, params.OffsetA, params.XPerChanB, params.OffsetB)
	if err != nil {
		return nil, "", err
	}

	var beamParams = make(map[string]float32)
	if params.MsaBeamParams != "" {
		labels := [4]string{"xscale", "xbias", "yscale", "ybias"}
		bits := strings.Split(params.MsaBeamParams, ",")
		for i, item := range bits {
			f, _ := strconv.ParseFloat(item, 32)
			beamParams[labels[i]] = float32(f)
		}
		beamLookup, err = makeBeamLocationFromSpectrums(spectraLookup, beamParams, minContextPMC)
		if err != nil {
			return nil, "", err
		}
	}

	if params.GenBulkMax {
		pmc, data := makeBulkMaxSpectra(spectraLookup, params.XPerChanA, params.OffsetA, params.XPerChanB, params.OffsetB)

		// If we're excluding all normal/dwell spectra, just include this one on its own
		if params.ExcludeNormalDwellSpectra {
			spectraLookup = dataConvertModels.DetectorSampleByPMC{pmc: data}
		} else {
			spectraLookup[pmc] = data
		}
	}

	importerutils.LogIfMoreFoundMSA(spectraLookup, "MSA/spectrum", 2, jobLog)
	// Not really relevant, what would we show? It's a list of meta, how many is too many?
	//importer.LogIfMoreFoundHousekeeping(hkData, "Housekeeping", 1)

	matchedAlignedImages, err := importerutils.ReadMatchedImages(filepath.Join(importPath, "MATCHED"), beamLookup, jobLog, localFS)

	if err != nil {
		return nil, "", err
	}

	// Build internal representation of the data that we can pass to the output code
	meta := dataConvertModels.FileMetaData{
		TargetID: params.TargetID,
		Target:   params.Target,
		SiteID:   params.SiteID,
		Site:     params.Site,
		Title:    params.Title,
		SOL:      params.SOL,
	}

	instr := protos.ScanInstrument_JPL_BREADBOARD
	if params.Group != "JPL Breadboard" {
		instr = protos.ScanInstrument_SBU_BREADBOARD // OK hack for now...
	}

	creator := params.CreatorUserId
	if len(creator) <= 0 {
		creator = specialUserIds.JPLImport
		if instr == protos.ScanInstrument_SBU_BREADBOARD {
			creator = specialUserIds.SBUImport
		}
	}

	data := &dataConvertModels.OutputData{
		DatasetID:            params.DatasetID,
		Instrument:           instr,
		Meta:                 meta,
		DetectorConfig:       params.DetectorConfig,
		BulkQuantFile:        params.BulkQuantFile,
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		MatchedAlignedImages: matchedAlignedImages,
		CreatorUserId:        creator,
	}

	data.SetPMCData(beamLookup, hkData, spectraLookup, contextImgsPerPMC, pseudoIntensityData, map[int32]string{})
	return data, contextImageSrcDir, nil
}

// Check what the minimum PMC is we have a context image for
func getMinimumContextPMC(contextImgsPerPMC map[int32]string) int32 {
	minContextPMC := int32(0)

	for contextPMC := range contextImgsPerPMC {
		if minContextPMC == 0 || contextPMC < minContextPMC {
			minContextPMC = contextPMC
		}
	}
	if minContextPMC == 0 {
		minContextPMC = 1
	}

	return minContextPMC
}
