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

package importer

import (
	"fmt"
	"strconv"

	"gitlab.com/pixlise/pixlise-go-api/core/logger"
	"gitlab.com/pixlise/pixlise-go-api/data-converter/converterModels"
)

func ReadPseudoIntensityFile(path string, expectHeaderRow bool, jobLog logger.ILogger) (converterModels.PseudoIntensities, error) {
	firstRealRowIdx := 0
	if expectHeaderRow {
		firstRealRowIdx = 1
	}

	data, err := ReadCSV(path, firstRealRowIdx, ',', jobLog)
	if err != nil {
		return nil, err
	}

	values, err := parsePsuedoIntensityData(data)
	if err != nil {
		return nil, fmt.Errorf("Pseudo-intensity CSV: %s - %v", path, err)
	}

	return values, nil
}

func parsePsuedoIntensityData(data [][]string) (converterModels.PseudoIntensities, error) {
	// CSV contains 2 tables: one at the start with PMC and xyz data, and another with the pseudointensities
	// Assume first row is the PMC table start, otherwise fail
	if data[0][0] != "PMC" {
		return nil, fmt.Errorf("expected first table to contain PMCs in first column, found: %v", data[0][0])
	}

	// As we parse, we first:
	// Read all PMCs, as we need to know what PMC the next tables rows refer to
	pmcs := []int32{}

	// If we find the header row of the 2nd table, we read into:
	result := converterModels.PseudoIntensities{}

	dataTableFirstRowIdx := -1

	for idx, row := range data[1:] {
		if dataTableFirstRowIdx > 0 {
			// We're reading pseudointensities...
			pseudoIntensities := []float32{}

			for colIdx, col := range row {
				val, err := strconv.ParseFloat(col, 32)
				if err != nil {
					return nil, fmt.Errorf("row %v, col %v - expected pseudointensity value, got: %v", idx+1, colIdx+1, col)
				}

				pseudoIntensities = append(pseudoIntensities, float32(val))
			}

			pmc := pmcs[idx-dataTableFirstRowIdx]
			result[pmc] = pseudoIntensities
		} else if len(row) > 3 && row[0] == "pi1" && row[1] == "pi2" && row[2] == "pi3" && row[3] == "pi4" {
			// Found the data table!
			dataTableFirstRowIdx = idx + 1
		} else {
			// We're reading PMCs
			pmcI, err := strconv.Atoi(row[0])
			if err != nil {
				return nil, fmt.Errorf("row %v - expected PMC, got: %v", idx+1, row[0])
			}

			pmcs = append(pmcs, int32(pmcI))
		}
	}

	return result, nil
}
