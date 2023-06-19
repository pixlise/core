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
	"strconv"
	"strings"

	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/data-import/internal/dataConvertModels"
)

// ReadBeamLocationsFile - Reads beam location CSV. Old style (expectMultipleIJ=false) or new multi-image IJ coord CSVs
func ReadBeamLocationsFile(path string, expectMultipleIJ bool, mainImagePMC int32, jobLog logger.ILogger) (dataConvertModels.BeamLocationByPMC, error) {
	rowsToSkip := 0
	if !expectMultipleIJ {
		// If we're loading the old style test data, that had an extra header that we skip
		rowsToSkip = 1
	}
	// read CSV
	rows, err := ReadCSV(path, rowsToSkip, ',', jobLog)
	if err != nil {
		return nil, err
	}

	return parseBeamLocations(rows, expectMultipleIJ, mainImagePMC)
}

func parseBeamLocations(rows [][]string, expectMultipleIJ bool, mainImagePMC int32) (dataConvertModels.BeamLocationByPMC, error) {
	headerLookup, geom_corrIdx, err := parseBeamLocationHeaders(rows[0], expectMultipleIJ, mainImagePMC)
	if err != nil {
		return nil, err
	}

	// Read in each row and store based on the header lookup we made
	result := dataConvertModels.BeamLocationByPMC{}

	for line, row := range rows[1:] {
		pmc, locData, err := parseBeamLocationRow(row, headerLookup, geom_corrIdx)
		if err != nil {
			return nil, fmt.Errorf("line [%v] - ERROR: %v", line, err)
		}
		if _, ok := result[pmc]; ok {
			return nil, fmt.Errorf("line [%v] - ERROR: duplicate PMC %v", line, pmc)
		}

		result[pmc] = locData
	}

	return result, nil
}

type pmcColIdxs struct {
	pmc int32
	//data string
	iIdx int
	jIdx int
}

// Gives back the i/j column indexes for each context image image (identified by PMC), geom_corr column index (or -1 if not there), and an error
func parseBeamLocationHeaders(header []string, expectMultipleIJ bool, mainImagePMC int32) ([]pmcColIdxs, int32, error) {
	result := []pmcColIdxs{}
	geom_corrIdx := int32(-1)

	// Check that these items are first...
	expHeaders := []string{"PMC", "x", "y", "z"}

	if !expectMultipleIJ {
		// If we're not expecting multiple IJ per PMC, we're reading the old test data format, and here
		// we expect the ij columns too
		expHeaders = append(expHeaders, "image_i")
		expHeaders = append(expHeaders, "image_j")
	} else {
		// We import also a new geom_corr column, this is optional, but would only appear in newer beam locations where we have multi-ij's
		// This should be at the end of the expected header values...
		if len(header) > len(expHeaders) && header[len(expHeaders)] == "geom_corr" {
			geom_corrIdx = int32(len(expHeaders))
			expHeaders = append(expHeaders, "geom_corr")
		}

		// we DO expect multiple i/j so check that there is at least 1 set
		ijColCount := len(header) - len(expHeaders)
		if ijColCount <= 0 || (ijColCount%2) != 0 {
			return nil, geom_corrIdx, errors.New("Unexpected count of i/j columns")
		}
	}

	// Make sure the first ones match our expected
	for idx, expColName := range expHeaders {
		if idx >= len(header) || header[idx] != expColName {
			return nil, geom_corrIdx, errors.New("Expected column " + expColName + ", got: " + header[idx])
		}
	}

	if expectMultipleIJ {
		// Rest of the columns should be for ij's per PMC, so work out what they are
		ijColCount := len(header) - len(expHeaders)
		if ijColCount <= 0 || (ijColCount%2) != 0 {
			return nil, geom_corrIdx, errors.New("Unexpected count of i/j columns")
		}
		// Run through them, expecting them to be in i/j order
		for idx := len(expHeaders); idx < len(header); idx += 2 {
			pmc, datatype, coord, err := splitColumnHeader(header[idx])
			if err != nil {
				return nil, geom_corrIdx, err
			}

			pmc2, datatype2, coord2, err := splitColumnHeader(header[idx+1])
			if err != nil {
				return nil, geom_corrIdx, err
			}

			if pmc != pmc2 || datatype != datatype2 || coord != "i" || coord2 != "j" {
				return nil, geom_corrIdx, errors.New("Unexpected column header " + header[idx+1] + " after " + header[idx])
			}

			// We only read the MCC coordinates
			if datatype == "MCC" {
				result = append(result, pmcColIdxs{pmc, idx, idx + 1})
			}
		}
	} else {
		// We verified it's a single i/j file (like in our older test data files), so we
		// simply set the columns as we saw them
		result = append(result, pmcColIdxs{mainImagePMC, 4, 5})
	}

	return result, geom_corrIdx, nil
}

func splitColumnHeader(header string) (int32, string, string, error) {
	// These are either of:
	// PMC_<num>_corr_i
	// PMC_<num>_corr_j
	// PMC_<num>_MCC_i
	// PMC_<num>_MCC_j

	bits := strings.Split(header, "_")
	if len(bits) == 4 && bits[0] == "PMC" && (bits[2] == "MCC" || bits[2] == "corr") && (bits[3] == "i" || bits[3] == "j") {
		// Try read the PMC
		pmcI, err := strconv.Atoi(bits[1])
		if err == nil {
			// It's right, return it!
			return int32(pmcI), bits[2], bits[3], nil
		}
	}

	return 0, "", "", fmt.Errorf("Unexpected column: %v", header)
}

func parseBeamLocationRow(row []string, headerLookup []pmcColIdxs, geom_corrIdx int32) (int32, dataConvertModels.BeamLocation, error) {
	locData := dataConvertModels.BeamLocation{}
	locData.IJ = map[int32]dataConvertModels.BeamLocationProj{}

	// Read the PMC,x,y,z columns
	pmcI, err := strconv.Atoi(row[0])
	if err != nil {
		return 0, locData, fmt.Errorf("Failed to read PMC: %v", row[0])
	}

	for c := 0; c <= 3; c++ {
		fVal, err := strconv.ParseFloat(row[c], 32)
		if err != nil {
			return 0, locData, fmt.Errorf("Failed to read x/y/z value: %v", row[c])
		}

		switch c {
		case 1:
			locData.X = float32(fVal)
		case 2:
			locData.Y = float32(fVal)
		case 3:
			locData.Z = float32(fVal)
		}
	}

	// Read geom column if needed
	if geom_corrIdx > -1 {
		fVal, err := strconv.ParseFloat(row[geom_corrIdx], 32)
		if err != nil {
			return 0, locData, fmt.Errorf("Failed to read geom_corr value: %v", row[geom_corrIdx])
		}

		locData.GeomCorr = float32(fVal)
	}
	// NOTE: If no geom_corr value, it ends up as 0 here. We don't want to save 0's, so we ensure later to NOT set it
	//       in the protobuf file if it's 0

	// Read the ij's
	for _, ijCol := range headerLookup {
		iVal, err := strconv.ParseFloat(row[ijCol.iIdx], 32)
		jVal, err2 := strconv.ParseFloat(row[ijCol.jIdx], 32)
		if err != nil || err2 != nil {
			return 0, locData, fmt.Errorf("Failed to read i,j values: %v,%v", row[ijCol.iIdx], row[ijCol.jIdx])
		}

		locData.IJ[ijCol.pmc] = dataConvertModels.BeamLocationProj{I: float32(iVal), J: float32(jVal)}
	}

	return int32(pmcI), locData, nil
}
