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
	"bufio"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/pixlise/core/v2/core/logger"
	gdsfilename "github.com/pixlise/core/v2/data-import/gds-filename"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
)

func ReadCSV(filePath string, headerIdx int, sep rune, jobLog logger.ILogger) ([][]string, error) {
	csvFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	if headerIdx > 0 {
		n := 0
		for n < headerIdx {
			n = n + 1
			row1, err := bufio.NewReader(csvFile).ReadSlice('\n')
			if err != nil {
				return nil, err
			}
			_, err = csvFile.Seek(int64(len(row1)), io.SeekStart)
			if err != nil {
				return nil, err
			}
		}
	}

	r := csv.NewReader(csvFile)
	r.TrimLeadingSpace = true
	r.Comma = sep

	// Some of our CSV files contain multiple tables, that we detect during parsing, so instead of using
	// ReadAll() here, which blows up when the # cols differs, we read each line, and if we get the error
	// "wrong number of fields", we can ignore it and keep reading
	rows := [][]string{}
	var lineRecord []string
	for {
		lineRecord, err = r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			if csverr, ok := err.(*csv.ParseError); !ok && csverr.Err != csv.ErrFieldCount {
				return nil, err
			}
		}

		rows = append(rows, lineRecord)
	}

	if len(rows) <= 0 {
		return rows, fmt.Errorf("Read 0 rows from: %v", filePath)
	}
	return rows, nil
}

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
	meta, err := gdsfilename.MakeDatasetFileMeta(datasetMeta, log)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse file metadata: %v", err)
	}

	// Expecting RTT to be read
	if len(meta.RTT) <= 0 {
		return nil, errors.New("Failed to determine dataset RTT")
	}

	// Ensure it matches what we're expecting
	// We allow for missing 0's at the start because for a while we imported RTTs as ints, so older dataset RTTs
	// were coming in as eg 76481028, while we now read them as 076481028
	if meta.RTT != datasetIDExpected && meta.RTT != "0"+datasetIDExpected {
		return nil, fmt.Errorf("Expected dataset ID %v, read %v", datasetIDExpected, meta.RTT)
	}

	detectorConfig := "PIXL"
	group := "PIXL-FM"

	// Depending on the SOL we may override the group and detector, as we have some test datasets that came
	// from the EM and have special characters as first part of SOL
	if len(meta.SOL) > 0 && (meta.SOL[0] == 'D' || meta.SOL[0] == 'C') {
		detectorConfig = "PIXL-EM-E2E"
		group = "PIXL-EM"
	}

	data := &dataConvertModels.OutputData{
		DatasetID:      meta.RTT,
		Group:          group,
		Meta:           meta,
		DetectorConfig: detectorConfig,
		//BulkQuantFile: "", <-- no bulk quant for tactical... TODO: what do we do here, does a scientist do it and we publish it back through PDS?
		PseudoRanges:         pseudoIntensityRanges,
		PerPMCData:           map[int32]*dataConvertModels.PMCData{},
		RGBUImages:           rgbuImages,
		DISCOImages:          discoImages,
		MatchedAlignedImages: matchedAlignedImages,
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
