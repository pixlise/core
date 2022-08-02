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

package importer

import (
	"fmt"
	"strconv"

	"github.com/pixlise/core/core/logger"
	"github.com/pixlise/core/data-converter/converterModels"
	protos "github.com/pixlise/core/generated-protos"
)

func ReadHousekeepingFile(path string, headerRowCount int, jobLog logger.ILogger) (converterModels.HousekeepingData, error) {
	rows, err := ReadCSV(path, headerRowCount, ',', jobLog)
	if err != nil {
		return converterModels.HousekeepingData{}, err
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
func convertHousekeepingData(header []string, pmcColIdx int, data [][]string, colTypes []protos.Experiment_MetaDataType) (converterModels.HousekeepingData, error) {
	result := converterModels.HousekeepingData{header, map[int32][]converterModels.MetaValue{}}

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

		save := []converterModels.MetaValue{}
		pmc := int32(-1)

		// Parse each value as intended into the given array
		for colIdx, value := range row {
			if colIdx == pmcColIdx {
				iValue, _ := strconv.Atoi(value)
				pmc = int32(iValue)
			} else {
				switch colTypes[colIdx] {
				case protos.Experiment_MT_STRING:
					save = append(save, converterModels.StringMetaValue(value))
				case protos.Experiment_MT_INT:
					iValue, _ := strconv.Atoi(value)
					save = append(save, converterModels.IntMetaValue(int32(iValue)))
				case protos.Experiment_MT_FLOAT:
					fValue, _ := strconv.ParseFloat(value, 64)
					save = append(save, converterModels.FloatMetaValue(float32(fValue)))
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
