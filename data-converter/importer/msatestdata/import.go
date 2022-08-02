// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package msatestdata

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pixlise/core/core/logger"
	"github.com/pixlise/core/data-converter/converterModels"

	"github.com/pixlise/core/data-converter/importer"
)

// We expect a JSON with these values in test datasets to provide us all required parameters
type importParams struct {
	DatasetID                 string  `json:"datasetid"`            // Dataset ID to output (affects output path and goes in summary file)
	Title                     string  `json:"title"`                // Title for this dataset
	TargetID                  string  `json:"targetid"`             // Target id to include in output
	Target                    string  `json:"target"`               // Target name include in output
	SiteID                    int32   `json:"siteid"`               // Site id to include in output
	Site                      string  `json:"site"`                 // Site name to include in output
	Group                     string  `json:"group"`                // Group the dataset will belong to
	BeamFile                  string  `json:"beamfile"`             // Beam location CSV path
	MsaBeamParams             string  `json:"beamparams"`           // Beam generation params if no beam location file
	HousekeepingFile          string  `json:"housekeeping"`         // Housekeeping CSV path
	ContextImgDir             string  `json:"contextimgdir"`        // Dir to find context images in
	MsaDir                    string  `json:"msadir"`               // Dir to load MSA files from
	PseudoIntensityCSVPath    string  `json:"pseudointensitycsv"`   // Pseudointensity CSV path
	IgnoreMSAFiles            string  `json:"ignoremsa"`            // MSA files to ignore
	SingleDetectorMSAs        bool    `json:"singledetectormsa"`    // Expecting single detector (1 column) MSA files
	GenPMCs                   bool    `json:"genpmcs"`              // Generate PMCs because it's an older test dataset without any
	ReadTypeOverride          string  `json:"readtype"`             // What to read MSAs as (normal vs dwell) because files arent named that way
	DetectorADuplicate        bool    `json:"detaduplicate"`        // Duplication of detector A to B, because test MSA only had 1 set of spectra
	GenBulkMax                bool    `json:"genbulkmax"`           // Generate bulk sum/max channel (because test dataset didnt come with one)
	DetectorConfig            string  `json:"detectorconfig"`       // Detector config that created this dataset, passed to PIQUANT when quantifying
	BulkQuantFile             string  `json:"bulkquantfile"`        // Bulk quantification file (for tactical datasets)
	XPerChanA                 float32 `json:"ev_xperchan_a"`        // eV calibration eV/channel (detector A)
	OffsetA                   float32 `json:"ev_offset_a"`          // eV calibration eV start offset (detector A)
	XPerChanB                 float32 `json:"ev_xperchan_b"`        // eV calibration eV/channel (detector B)
	OffsetB                   float32 `json:"ev_offset_b"`          // eV calibration eV start offset (detector B)
	ExcludeNormalDwellSpectra bool    `json:"exclude_normal_dwell"` // Hack for tactical datasets - load all MSAs to gen bulk sum, but dont save them in output
	SOL                       string  `json:"sol"`                  // Might as well be able to specify SOL. Needed for first spectrum dataset on SOL13
}

type MSATestData struct {
}

// Import - Implementing Importer interface, expects importPath to point to a JSON file with import params
func (m MSATestData) Import(importJSONPath string, pseudoIntensityRangesPath string, jobLog logger.ILogger) (*converterModels.OutputData, string, error) {
	if path.Ext(importJSONPath) != ".json" {
		return nil, "", errors.New("expected import path to point to import parameter JSON file")
	}

	var params importParams
	err := importer.ReadJSON(importJSONPath, &params, jobLog)
	if err != nil {
		return nil, "", err
	}

	if len(params.DatasetID) <= 0 {
		return nil, "", errors.New("Import parameter file did not specify a DatasetID")
	}
	if len(params.Group) <= 0 {
		return nil, "", errors.New("Import parameter file did not specify a Group")
	}

	// So the import path itself is the dir the json file sits in
	importPath := path.Dir(importJSONPath)

	// Process the context images first, so if we're assigning an image to a PMC, we know what PMC it is
	// NOTE: this only matters for msa test datasets where the context image file names contain the PMC. In the
	// case where there is no PMC info in the file name, we just assign it to PMC 1 anyway.
	contextImageSrcDir := path.Join(importPath, params.ContextImgDir)
	contextImgsPerPMC, err := processContextImages(contextImageSrcDir, jobLog)
	if err != nil {
		return nil, "", err
	}

	minContextPMC := getMinimumContextPMC(contextImgsPerPMC)

	var hkData converterModels.HousekeepingData
	var beamLookup = make(converterModels.BeamLocationByPMC)

	if params.MsaBeamParams == "" && params.BeamFile != "" {
		fmt.Printf("  Reading Beam Locations: \"%v\", using minimum context image PMC detected: %v\n", params.BeamFile, minContextPMC)
		beamLookup, err = importer.ReadBeamLocationsFile(path.Join(importPath, params.BeamFile), false, minContextPMC, jobLog)
		if err != nil {
			return nil, "", err
		}
	} else if params.MsaBeamParams != "" {
		//beamLookup = converterModels.BeamLocations{}
	}

	// Housekeeping file
	if params.HousekeepingFile != "" {
		fmt.Printf("  Reading Housekeeping: %v\n", params.HousekeepingFile)
		hkData, err = importer.ReadHousekeepingFile(path.Join(importPath, params.HousekeepingFile), 0, jobLog)
		if err != nil {
			return nil, "", err
		}

		//logIfMoreFound(m, "housekeeping", 1)
	}

	// Pseudointensity data - if we have the CSV, load it and ranges file, otherwise nothing
	var pseudoIntensityRanges []converterModels.PseudoIntensityRange = nil
	var pseudoIntensityData converterModels.PseudoIntensities = nil
	if len(params.PseudoIntensityCSVPath) > 0 {
		// Can't have data & no range description...
		if len(pseudoIntensityRangesPath) <= 0 {
			return nil, "", errors.New("If passing pseudo-intensity CSV file, pseudo-intensity ranges file must also be provided")
		}

		pseudoIntensityRanges, err = importer.ReadPseudoIntensityRangesFile(pseudoIntensityRangesPath, jobLog)
		if err != nil {
			return nil, "", err
		}
		pseudoIntensityData, err = importer.ReadPseudoIntensityFile(path.Join(importPath, params.PseudoIntensityCSVPath), false, jobLog)
		if err != nil {
			return nil, "", err
		}
	}

	allMSAFiles, err := listMSAFilesToProcess(path.Join(importPath, params.MsaDir), params.IgnoreMSAFiles, jobLog)
	if err != nil {
		return nil, "", err
	}

	verifyreadtype := true
	if params.ReadTypeOverride != "" {
		verifyreadtype = false
	}

	fmt.Printf("  Reading %v spectrum files...\n", len(allMSAFiles))
	spectrafiles, _ := getSpectraFiles(allMSAFiles, verifyreadtype)
	spectraLookup, err := makeSpectraLookup(path.Join(importPath, params.MsaDir), spectrafiles, params.SingleDetectorMSAs, params.GenPMCs, params.ReadTypeOverride, params.DetectorADuplicate, jobLog)
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
			spectraLookup = converterModels.DetectorSampleByPMC{pmc: data}
		} else {
			spectraLookup[pmc] = data
		}
	}

	importer.LogIfMoreFoundMSA(spectraLookup, "MSA/spectrum", 2)
	// Not really relevant, what would we show? It's a list of meta, how many is too many?
	//importer.LogIfMoreFoundHousekeeping(hkData, "Housekeeping", 1)

	// Build internal representation of the data that we can pass to the output code
	meta := converterModels.FileMetaData{
		TargetID: params.TargetID,
		Target:   params.Target,
		SiteID:   params.SiteID,
		Site:     params.Site,
		Title:    params.Title,
		SOL:      params.SOL,
	}

	data := &converterModels.OutputData{
		DatasetID:      params.DatasetID,
		Group:          params.Group,
		Meta:           meta,
		DetectorConfig: params.DetectorConfig,
		BulkQuantFile:  params.BulkQuantFile,
		PseudoRanges:   pseudoIntensityRanges,
		PerPMCData:     map[int32]*converterModels.PMCData{},
	}

	data.SetPMCData(beamLookup, hkData, spectraLookup, contextImgsPerPMC, pseudoIntensityData)
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
