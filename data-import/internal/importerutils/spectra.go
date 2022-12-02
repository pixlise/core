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
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/core/utils"
	gdsfilename "github.com/pixlise/core/v2/data-import/gds-filename"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
)

func ReadSpectraCSV(path string, jobLog logger.ILogger) (dataConvertModels.DetectorSampleByPMC, error) {
	data, err := ReadCSV(path, 0, ',', jobLog)
	if err != nil {
		return nil, err
	}

	values, err := parseSpectraCSVDataMultiTable(data, jobLog)
	if err != nil {
		return nil, fmt.Errorf("Spectra CSV: %s - %v", path, err)
	}

	return values, nil
}

var csvMetaTableHeaderColsNoYellow = []string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"}
var csvMetaTableHeaderCols = []string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "yellow_piece_temp", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"}

func parseSpectraCSVDataMultiTable(data [][]string, jobLog logger.ILogger) (dataConvertModels.DetectorSampleByPMC, error) {
	// CSV contains 2 sets of 3 tables (second set of 3 is optional):
	// 1. PMC/metadata table
	// 2. A spectra channels
	// 3. B spectra channels
	// 4/5/6. Dwell versions of the above tables

	// Here we split it into 2 lots of data (2 sets of 3 tables), then read them
	// and form one list of PMCs
	dataNormal, dataDwell := splitSpectraCSVTables(data)

	values, err := parseSpectraCSVData(dataNormal, "Normal", jobLog)
	if err != nil {
		return values, err
	}

	if len(dataDwell) <= 0 {
		return values, nil
	}

	// We have dwell data, so parse it
	valuesDwell, err := parseSpectraCSVData(dataDwell, "Dwell", jobLog)
	if err != nil {
		return valuesDwell, err
	}

	// Combine them with the normal values we just read
	return combineNormalDwellSpectra(values, valuesDwell)
}

func combineNormalDwellSpectra(normal dataConvertModels.DetectorSampleByPMC, dwell dataConvertModels.DetectorSampleByPMC) (dataConvertModels.DetectorSampleByPMC, error) {
	result := dataConvertModels.DetectorSampleByPMC{}
	for k, v := range normal {
		result[k] = v
	}
	for k, v := range dwell {
		existing, ok := result[k]
		if !ok {
			// Dwells should be for an existing PMC, not existing on their own
			return dataConvertModels.DetectorSampleByPMC{}, fmt.Errorf("Found dwell spectrum PMC: %v which has no corresponding normal spectrum", k)
		}
		result[k] = append(existing, v...)
	}
	return result, nil
}

func splitSpectraCSVTables(data [][]string) ([][]string, [][]string) {
	dwellIdx := -1
	for idx, row := range data[1:] {
		if utils.StringSlicesEqual(row, csvMetaTableHeaderCols) || utils.StringSlicesEqual(row, csvMetaTableHeaderColsNoYellow) {
			dwellIdx = idx
			break
		}
	}

	if dwellIdx > 0 {
		dwellIdx++ // Move on to actual header row
		return data[0:dwellIdx], data[dwellIdx:]
	}
	return data, [][]string{}
}

func parseSpectraCSVData(data [][]string, readType string, jobLog logger.ILogger) (dataConvertModels.DetectorSampleByPMC, error) {
	// This reads the 3 tables that describe the spectra of a set of PMCs:
	// 1. PMC and metadata for spectra
	//    Example: SCLK_A,SCLK_B,PMC,real_time_A,real_time_B,live_time_A,live_time_B,XPERCHAN_A,XPERCHAN_B,OFFSET_A,OFFSET_B
	// 2. A spectra channels (4096 columns)
	//    Header labels are: A_ch_1,A_ch_2,A_ch_3, etc
	// 3. B spectra channels (4096 columns)
	//    Header labels are: B_ch_1,B_ch_2,B_ch_3, etc

	metaColumnSaveRenames := map[string]string{
		"SCLK_A":            "SCLK",
		"SCLK_B":            "SCLK",
		"real_time_A":       "REALTIME",
		"real_time_B":       "REALTIME",
		"live_time_A":       "LIVETIME",
		"live_time_B":       "LIVETIME",
		"XPERCHAN_A":        "XPERCHAN",
		"XPERCHAN_B":        "XPERCHAN",
		"OFFSET_A":          "OFFSET",
		"OFFSET_B":          "OFFSET",
		"yellow_piece_temp": "yellow_piece_temp",
	}

	// There are 2 types, the older intermediate format had no yellow_piece_temp
	expMetaCols := csvMetaTableHeaderCols

	if !utils.StringSlicesEqual(data[0], expMetaCols) {
		// Not new, maybe old?
		if utils.StringSlicesEqual(data[0], csvMetaTableHeaderColsNoYellow) {
			// Yes, old format
			expMetaCols = csvMetaTableHeaderColsNoYellow
		} else {
			// Plain just don't know...
			return nil, fmt.Errorf("Unexpected columns in metadata table: %v", data[0])
		}
	}

	result := dataConvertModels.DetectorSampleByPMC{}
	rowPMCs := []int32{}

	channelTable := ""
	channelTableFirstRowIdx := -1
	channelCount := 0

	readingXYZTable := false

	xyzTableHeader := []string{"PMC", "x", "y", "z"}
	tableAHeaderSample := []string{"A_1", "A_2", "A_3", "A_4", "A_5"}
	tableBHeaderSample := []string{"B_1", "B_2", "B_3", "B_4", "B_5"}

	foundATable := false
	foundBTable := false
	ATableRowsRead := 0
	BTableRowsRead := 0

	for idx, row := range data[1:] {
		if len(row) > len(tableAHeaderSample) && utils.StringSlicesEqual(row[0:len(tableAHeaderSample)], tableAHeaderSample) {
			// Found the A table!
			foundATable = true
			readingXYZTable = false
			channelTable = "A"
			channelCount = len(row)
			channelTableFirstRowIdx = idx + 1

			// Allocate spectra values for all
			for pmc := range result {
				for specIdx := range result[pmc] {
					result[pmc][specIdx].Spectrum = make([]int64, channelCount)
				}
			}
		} else if len(row) > len(tableBHeaderSample) && utils.StringSlicesEqual(row[0:len(tableBHeaderSample)], tableBHeaderSample) {
			// Found the B table!
			if !foundATable {
				return nil, fmt.Errorf("row %v - Found B table without seeing A table first", idx+1)
			}
			foundBTable = true
			readingXYZTable = false
			channelTable = "B"
			if channelCount != len(row) {
				return nil, fmt.Errorf("row %v - differing channel count found, A was %v, B is %v", idx+1, channelCount, len(row))
			}
			channelTableFirstRowIdx = idx + 1
		} else if channelTableFirstRowIdx > 0 {
			// We're reading spectra channels!
			if len(row) != channelCount {
				return nil, fmt.Errorf("row %v, Expected %v channel values, found %v", idx+1, channelCount, len(row))
			}

			pmc := rowPMCs[idx-channelTableFirstRowIdx]

			spectraIdx := 0
			if channelTable == "A" {
				ATableRowsRead++
			} else if channelTable == "B" {
				spectraIdx = 1
				BTableRowsRead++
			} else {
				return nil, fmt.Errorf("row %v, Unexpected current table: %v", idx+1, channelTable)
			}

			for colIdx, col := range row {
				valI, err := strconv.Atoi(col)
				if err != nil {
					return nil, fmt.Errorf("row %v, col %v - failed to read value, got: %v", idx+1, colIdx+1, col)
				}
				result[pmc][spectraIdx].Spectrum[colIdx] = int64(valI)
			}
		} else if !readingXYZTable {
			// If we've hit the XYZ table, we ignore that...
			if utils.StringSlicesEqual(row, xyzTableHeader) {
				readingXYZTable = true
				continue
			}

			// We're reading metadata
			// TODO: how do we tell if it's dwell vs normal?
			metaA := dataConvertModels.MetaData{"DETECTOR_ID": dataConvertModels.StringMetaValue("A"), "READTYPE": dataConvertModels.StringMetaValue(readType)}
			metaB := dataConvertModels.MetaData{"DETECTOR_ID": dataConvertModels.StringMetaValue("B"), "READTYPE": dataConvertModels.StringMetaValue(readType)}

			if len(row) != len(data[0]) {
				return nil, fmt.Errorf("row %v - expected %v metadata items in row, got: %v", idx+1, len(data[0]), len(row))
			}

			pmc := int32(0)
			for colIdx, colValue := range row {
				colName := data[0][colIdx]

				if colName == "PMC" {
					pmcI, err := strconv.Atoi(colValue)
					if err != nil {
						return nil, fmt.Errorf("row %v - expected PMC, got: %v", idx+1, colValue)
					}
					pmc = int32(pmcI)

					metaA[colName] = dataConvertModels.IntMetaValue(pmc)
					metaB[colName] = dataConvertModels.IntMetaValue(pmc)
				} else {
					// Read the meta value, bin sort into A vs B

					// We should have a save name defined for every column...
					saveName, ok := metaColumnSaveRenames[colName]
					// if not, that's a warning but we only print it for the first data row, otherwise we'd be printing this 100s of times
					if !ok && idx == 1 {
						jobLog.Infof("row %v - No meta column rename found for: %v\n", idx+1, colName)
					}

					if !ok {
						saveName = colName
					}

					// SCLK expects ints, the rest expect floats
					saveMetaValue := dataConvertModels.MetaValue{}

					if saveName == "SCLK" {
						iSCLK, err := strconv.Atoi(colValue)
						if err != nil {
							return nil, fmt.Errorf("row %v - expected SCLK, got: %v", idx+1, colValue)
						}
						saveMetaValue = dataConvertModels.IntMetaValue(int32(iSCLK))
					} else {
						fValue, err := strconv.ParseFloat(colValue, 32)
						if err != nil {
							return nil, fmt.Errorf("row %v - %v expected float, got: %v", idx+1, colName, colValue)
						}
						saveMetaValue = dataConvertModels.FloatMetaValue(float32(fValue))
					}

					if strings.HasSuffix(colName, "_A") {
						metaA[saveName] = saveMetaValue
					} else if strings.HasSuffix(colName, "_B") {
						metaB[saveName] = saveMetaValue
					} else if colName == "yellow_piece_temp" {
						// yellow_piece_temp is a new column, we don't need it, so ignore that
						// Took out printing, not needed once per row...
						//jobLog.Infof("row %v - Ignoring column name: %v\n", idx+1, colName)
					} else {
						// Don't know what this is, perhaps a new column introduced?
						return nil, fmt.Errorf("row %v - Unexpected meta column name: %v", idx+1, colName)
					}
				}
			}

			rowPMCs = append(rowPMCs, pmc)
			if _, ok := result[pmc]; ok {
				return nil, fmt.Errorf("Found duplicate PMC: %v in metadata table", pmc)
			}

			result[pmc] = []dataConvertModels.DetectorSample{
				dataConvertModels.DetectorSample{
					Meta:     metaA,
					Spectrum: []int64{},
				},
				dataConvertModels.DetectorSample{
					Meta:     metaB,
					Spectrum: []int64{},
				},
			}
		}
	}

	// If we didn't see one of the tables, this is our last chance to complain
	if !foundATable || !foundBTable {
		return nil, fmt.Errorf("Did not find both A and B tables")
	}
	if ATableRowsRead == 0 || ATableRowsRead != BTableRowsRead {
		return nil, fmt.Errorf("A table had %v rows, B had %v", ATableRowsRead, BTableRowsRead)
	}
	return result, nil
}

func ReadBulkMaxSpectra(inPath string, files []string, jobLog logger.ILogger) (dataConvertModels.DetectorSampleByPMC, error) {
	result := dataConvertModels.DetectorSampleByPMC{}

	for _, file := range files {
		// Parse metadata for file
		csvMeta, err := gdsfilename.ParseFileName(file)
		if err != nil {
			return nil, err
		}

		// Make sure it's one of the products we're expecting
		readType := ""
		if csvMeta.ProdType == "RBS" {
			readType = "BulkSum"
		} else if csvMeta.ProdType == "RMS" {
			readType = "MaxValue"
		} else {
			return nil, fmt.Errorf("Unexpected bulk/max MSA product type: %v", csvMeta.ProdType)
		}

		pmc, err := csvMeta.PMC()
		if err != nil {
			return nil, err
		}

		csvPath := path.Join(inPath, file)
		jobLog.Infof("  Reading %v MSA: %v", readType, csvPath)
		lines, err := ReadFileLines(csvPath, jobLog)
		if err != nil {
			return nil, fmt.Errorf("Failed to load %v: %v", csvPath, err)
		}

		// Parse the MSA data
		spectrumList, err := ReadMSAFileLines(lines, false, false, false)
		if err != nil {
			return nil, fmt.Errorf("Failed to parse %v: %v", csvPath, err)
		}

		// Set the read type, detector & PMC
		for c := range spectrumList {
			detector := "A"
			if c > 0 {
				detector = "B"
			}
			spectrumList[c].Meta["READTYPE"] = dataConvertModels.StringMetaValue(readType)
			spectrumList[c].Meta["DETECTOR_ID"] = dataConvertModels.StringMetaValue(detector)
			spectrumList[c].Meta["PMC"] = dataConvertModels.IntMetaValue(pmc)
		}

		if _, ok := result[pmc]; !ok {
			result[pmc] = []dataConvertModels.DetectorSample{}
		}
		result[pmc] = append(result[pmc], spectrumList...)
	}

	return result, nil
}
