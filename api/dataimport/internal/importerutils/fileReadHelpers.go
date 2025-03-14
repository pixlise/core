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

package importerutils

import (
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/api/specialUserIds"
	"github.com/pixlise/core/v4/core/gdsfilename"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// Given the stuff read from disk, this takes all the data and assembles it in the output structure
// This was hard-coded into the FM importer in past, but now that we have SOFF files they need to
// work the same way, so it's been pulled into here
func MakeFMDatasetOutput(
	beamLookup dataConvertModels.BeamLocationByPMC,
	hkData dataConvertModels.HousekeepingData,
	locSpectraLookup dataConvertModels.DetectorSampleByPMC,
	bulkMaxSpectraLookup dataConvertModels.DetectorSampleByPMC,
	contextImgsPerPMC map[int32]string,
	pseudoIntensityData dataConvertModels.PseudoIntensities,
	pseudoIntensityRanges []dataConvertModels.PseudoIntensityRange,
	matchedAlignedImages []dataConvertModels.MatchedAlignedImageMeta,
	rgbuImages []dataConvertModels.ImageMeta,
	discoImages []dataConvertModels.ImageMeta,
	whiteDiscoImage string,
	datasetMeta gdsfilename.FileNameMeta,
	datasetIDExpected string,
	overrideInstrument protos.ScanInstrument,
	overrideDetector string,
	beamVersion uint32,
	log logger.ILogger,
) (*dataConvertModels.OutputData, error) {
	// Now that all have been read, combine the bulk/max spectra into our lookup
	for pmc := range bulkMaxSpectraLookup {
		locSpectraLookup[pmc] = append(locSpectraLookup[pmc], bulkMaxSpectraLookup[pmc]...)
	}

	// Print out any weird ones
	LogIfMoreFoundMSA(locSpectraLookup, "MSA/spectrum", 2, log)
	// Not really relevant, what would we show? It's a list of meta, how many is too many?
	//importer.LogIfMoreFoundHousekeeping(hkData, "Housekeeping", 1)

	// Build internal representation of the data that we can pass to the output code
	// We now read the metadata from the housekeeping file name, as it's the only file we expect to always exist!
	meta, err := makeDatasetFileMeta(datasetMeta, log)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse file metadata: %v", err)
	}

	// Expecting RTT to be read
	if len(meta.RTT) <= 0 {
		return nil, errors.New("Failed to determine dataset RTT")
	}

	detectorConfig := "PIXL"
	instrument := protos.ScanInstrument_PIXL_FM

	// If we're being overridden, use the incoming values
	if overrideInstrument != protos.ScanInstrument_UNKNOWN_INSTRUMENT && len(overrideDetector) > 0 {
		detectorConfig = overrideDetector
		instrument = overrideInstrument
	} else {
		isEM := false

		// Ensure it matches what we're expecting
		// We allow for missing 0's at the start because for a while we imported RTTs as ints, so older dataset RTTs
		// were coming in as eg 76481028, while we now read them as 076481028
		// NOTE: it looks like EM datasets are generated with the RTT: 000000453, 000000454
		// so if this is the RTT we don't do the check
		if meta.RTT == "000000453" || meta.RTT == "000000454" {
			isEM = true
			if datasetIDExpected == meta.RTT {
				return nil, fmt.Errorf("Read RTT %v, need expected dataset ID to be different", meta.RTT)
			} else {
				// Set the RTT to the expected ID, we're importing some kind of test dataset
				// so use something else, otherwise we would overwrite them all the time
				meta.RTT = datasetIDExpected
			}
		} else if meta.RTT != datasetIDExpected && meta.RTT != "0"+datasetIDExpected {
			return nil, fmt.Errorf("Expected dataset ID %v, read %v", datasetIDExpected, meta.RTT)
		}

		// Depending on the SOL we may override the group and detector, as we have some test datasets that came
		// from the EM and have special characters as first part of SOL
		if isEM || len(meta.SOL) > 0 && (meta.SOL[0] == 'D' || meta.SOL[0] == 'C') {
			detectorConfig = "PIXL-EM-E2E"
			instrument = protos.ScanInstrument_PIXL_EM
		}
	}

	data := &dataConvertModels.OutputData{
		DatasetID:      meta.RTT,
		Instrument:     instrument,
		Meta:           meta,
		DetectorConfig: detectorConfig,
		//BulkQuantFile: "", <-- no bulk quant for tactical... TODO: what do we do here, does a scientist do it and we publish it back through PDS?
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		RGBUImages:           rgbuImages,
		DISCOImages:          discoImages,
		MatchedAlignedImages: matchedAlignedImages,
		CreatorUserId:        specialUserIds.PIXLISESystemUserId, // Auto-importing FM datasets, we don't show a creator... TODO: what about EM though??
		BeamVersion:          beamVersion,
	}

	data.SetPMCData(beamLookup, hkData, locSpectraLookup, contextImgsPerPMC, pseudoIntensityData, map[int32]string{})
	if data.DefaultContextImage != "" {
		log.Infof("Setting context image to: " + data.DefaultContextImage)
	}
	// If we have no default context image at this point, see if we can use one of the DISCO images
	if len(data.DefaultContextImage) <= 0 && len(discoImages) > 0 {
		if len(whiteDiscoImage) > 0 {
			data.DefaultContextImage = whiteDiscoImage
			log.Infof("White Disco Image Found. Setting Context Image: " + whiteDiscoImage)
		} else {
			log.Infof("Setting Context to first Disco Image: " + discoImages[0].FileName)
			data.DefaultContextImage = discoImages[0].FileName
		}
	}

	return data, nil
}

func makeDatasetFileMeta(fMeta gdsfilename.FileNameMeta, jobLog logger.ILogger) (dataConvertModels.FileMetaData, error) {
	result := dataConvertModels.FileMetaData{}

	sol, err := fMeta.SOL()
	if err != nil {
		//return result, nil
		jobLog.Infof("Dataset Metadata did not contain SOL: %v", err)
	}

	rtt, err := fMeta.RTT()
	if err != nil {
		return result, nil
	}

	sclk, err := fMeta.SCLK()
	if err != nil {
		//return result, nil
		jobLog.Infof("Dataset Metadata did not contain SCLK: %v", err)
	}

	site, err := fMeta.Site()
	if err != nil {
		return result, nil
	}

	drive, err := fMeta.Drive()
	if err != nil {
		return result, nil
	}

	result.SOL = sol
	result.RTT = rtt
	result.SCLK = sclk
	result.SiteID = site
	result.DriveID = drive
	result.TargetID = "?"
	result.Title = rtt
	return result, nil
}
