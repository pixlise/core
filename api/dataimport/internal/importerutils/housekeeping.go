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
	"strconv"

	"github.com/pixlise/core/v4/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func ReadHousekeepingFile(path string, headerRowCount int, jobLog logger.ILogger) (dataConvertModels.HousekeepingData, error) {
	rows, err := ReadCSV(path, headerRowCount, ',', jobLog)
	if err != nil {
		return dataConvertModels.HousekeepingData{}, err
	}

	headers, pmcColIdx, colTypes, tableRowCount := scanHousekeepingData(rows)
	return convertHousekeepingData(headers, pmcColIdx, rows[1:tableRowCount], colTypes)
}

// Scans the data to work out how many rows there are to read in the first table (there are others we don't read)
// and what datatypes are in each row
func scanHousekeepingData(data [][]string) ([]string, int, []protos.Experiment_MetaDataType, int) {
	firstRowItemCount := len(data[0])
	rowCount := len(data)
	dataTypes := []protos.Experiment_MetaDataType{}
	headers := []string{}
	pmcColIdx := -1

	for rowIdx, row := range data {
		if len(row) != firstRowItemCount {
			// Stop scanning, we've found a row that differs in length (or potentially an empty
			// row between tables) so must be a new table
			rowCount = rowIdx
			break
		}

		for colIdx, value := range row {
			if rowIdx == 0 {
				// Assume first is the column headers, save them
				if value == "PMC" {
					pmcColIdx = colIdx
				} else {
					headers = append(headers, value)
				}
			} else {
				// Data row...
				valType := pickDataType(value)
				if rowIdx == 1 {
					// We're just saving whatever type we find
					dataTypes = append(dataTypes, valType)
				} else {
					// If the value is more "permissive", use that
					if valType != dataTypes[colIdx] {
						// Allow changing column from int->float
						if (valType == protos.Experiment_MT_FLOAT && dataTypes[colIdx] == protos.Experiment_MT_INT) ||
							// If anything is a string in the column, we have to just change all to string
							(valType == protos.Experiment_MT_STRING) {
							dataTypes[colIdx] = valType
						}
					}
				}
			}
		}
	}

	return headers, pmcColIdx, dataTypes, rowCount
}

func pickDataType(value string) protos.Experiment_MetaDataType {
	// It came in as a string, but if we can convert it to int or float, use that as the type
	_, err := strconv.Atoi(value)
	if err == nil {
		return protos.Experiment_MT_INT
	}

	_, err = strconv.ParseFloat(value, 64)
	if err == nil {
		return protos.Experiment_MT_FLOAT
	}

	// stick to string
	return protos.Experiment_MT_STRING
}

// Conversion of data to our variant types, given data type for each column done in a previous scan
func convertHousekeepingData(header []string, pmcColIdx int, data [][]string, colTypes []protos.Experiment_MetaDataType) (dataConvertModels.HousekeepingData, error) {
	result := dataConvertModels.HousekeepingData{
		Header:           header,
		Data:             map[int32][]dataConvertModels.MetaValue{},
		PerPMCHeaderIdxs: map[int32][]int32{},
	}

	// If the PMC col index isn't set, we've got something wrong
	if pmcColIdx < 0 || pmcColIdx > (len(header)+1) {
		return result, fmt.Errorf("PMC column index (%v) is invalid, is the column missing?", pmcColIdx)
	}

	// Make sure pmc column ended up found as integers
	if colTypes[pmcColIdx] != protos.Experiment_MT_INT {
		return result, fmt.Errorf("PMC column (%v) did not parse as integers, got: %v", pmcColIdx, colTypes[pmcColIdx])
	}

	for rowIdx, row := range data {
		if len(row) != len(colTypes) {
			return result, fmt.Errorf("Row %v: Invalid row item count, expected %v, got %v", rowIdx+1, len(colTypes), len(row))
		}

		save := []dataConvertModels.MetaValue{}
		pmc := int32(-1)

		// Parse each value as intended into the given array
		for colIdx, value := range row {
			if colIdx == pmcColIdx {
				iValue, _ := strconv.Atoi(value)
				pmc = int32(iValue)
			} else {
				switch colTypes[colIdx] {
				case protos.Experiment_MT_STRING:
					save = append(save, dataConvertModels.StringMetaValue(value))
				case protos.Experiment_MT_INT:
					iValue, _ := strconv.Atoi(value)
					save = append(save, dataConvertModels.IntMetaValue(int32(iValue)))
				case protos.Experiment_MT_FLOAT:
					fValue, _ := strconv.ParseFloat(value, 64)
					save = append(save, dataConvertModels.FloatMetaValue(float32(fValue)))
				default:
					return result, fmt.Errorf("Row %v: Invalid type defined: %v", rowIdx+1, colTypes[colIdx])
				}
			}
		}

		// save it!
		if pmc < 0 {
			return result, fmt.Errorf("Row %v: Invalid PMC: %v", rowIdx+1, pmc)
		}

		result.Data[pmc] = save
	}

	return result, nil
}
