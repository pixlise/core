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
