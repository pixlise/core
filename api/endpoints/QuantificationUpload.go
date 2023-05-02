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

package endpoints

import (
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/pixlise/core/v3/api/handlers"
	"github.com/pixlise/core/v3/core/api"
	"github.com/pixlise/core/v3/core/quantModel"
)

// Users can also upload a compatible CSV file which we can convert into a quantification that's usable inside PIXLISE
// We expect the body to contain the CSV, but first few lines are expected to contain other info:
// Name=The quant name
// Comments=The comments\nWith new lines\nEncoded like so
// CSV
// <csv title line>
// <csv column headers>
// <csv row 0>
// ...
// <csv row n>

func quantificationUpload(params handlers.ApiHandlerParams) (interface{}, error) {
	datasetID := params.PathParams[datasetIdentifier]

	// Read in body
	body, err := ioutil.ReadAll(params.Request.Body)
	if err != nil {
		return nil, err
	}

	// Check if it looks remotely valid
	strBody := string(body)
	csvRows := strings.Split(strBody, "\n")

	const formatError = `Bad upload format. Expecting format:
Name=The quant name
Comments=The comments\nWith new lines\nEncoded like so
CSV
<csv title line>
<csv column headers>
csv rows`

	// First rows are reserved to contain some values in weird format only for this POST... Check that they came in that format:
	if len(csvRows) < 6 {
		return nil, api.MakeBadRequestError(errors.New(formatError))
	}

	const quantNamePrefix = "Name="
	const commentPrefix = "Comments="

	if !strings.HasPrefix(csvRows[0], quantNamePrefix) || !strings.HasPrefix(csvRows[1], commentPrefix) || csvRows[2] != "CSV" {
		return nil, api.MakeBadRequestError(errors.New(formatError))
	}

	quantNameStart := len(quantNamePrefix)
	quantNameEnd := len(csvRows[0])
	// Limit it
	if quantNameEnd > 100 {
		quantNameEnd = 100
	}

	commentStart := len(commentPrefix)
	commentEnd := len(csvRows[1])
	// Limit it
	if commentEnd > 1000 {
		commentEnd = 1000
	}

	quantName := csvRows[0][quantNameStart:quantNameEnd]
	comments := csvRows[1][commentStart:commentEnd]

	colLookup, err := parseCSVColumns(csvRows[3:])
	if err != nil {
		return nil, api.MakeBadRequestError(err)
	}

	csvBody := strings.Join(csvRows[3:], "\n")

	quantMode := quantModel.quantModeCombinedManualUpload

	// We know the filename column exists due to parseCSVColumns above
	if isABQuant(csvRows, colLookup["filename"]) {
		quantMode = quantModel.quantModeABManualUpload
	}

	return quantModel.ImportQuantCSV(params.Svcs, datasetID, params.UserInfo, csvBody, "user-supplied", "upload", quantName, quantMode, comments)
}

func parseCSVColumns(csvRows []string) (map[string]int, error) {
	colMap := map[string]int{}

	if len(csvRows) <= 2 {
		return map[string]int{}, errors.New("CSV must contain more than 2 lines")
	}

	// Expect certain columns
	cols := strings.Split(csvRows[1], ",")

	// Build a map so it's easier to look up

	hasWeightCol := false
	for c, col := range cols {
		colClean := strings.Trim(col, " \t")
		colMap[colClean] = c

		if strings.HasSuffix(colClean, "_%") {
			hasWeightCol = true
		}
	}

	if !hasWeightCol {
		return map[string]int{}, errors.New("CSV did not contain any _% columns")
	}

	// An example of valid:
	// PMC, CaO_%, SiO2_%, FeO-T_%, CaO_int, SiO2_int, FeO-T_int, CaO_err, SiO2_err, FeO-T_err, total_counts, livetime, chisq, eVstart, eV/ch, res, iter, filename, Events, Triggers, SCLK, RTT
	// We require AT LEAST:
	reqCols := []string{"PMC", "livetime", "filename", "SCLK", "RTT"} // and one _% column
	for _, col := range reqCols {
		if _, ok := colMap[col]; !ok {
			return map[string]int{}, fmt.Errorf("CSV missing column: \"%v\"", col)
		}
	}

	return colMap, nil
}

func isABQuant(csvRows []string, filenameColumnIdx int) bool {
	if len(csvRows) < 3 {
		return false
	}

	// Check near first, middle and near-last rows to see if we find A and B detectors
	earlyRow := strings.Split(csvRows[2], ",")
	earlyIsCombined := false

	midRow := strings.Split(csvRows[(2+len(csvRows)-2)/2], ",")
	midIsCombined := false

	lastRow := strings.Split(csvRows[len(csvRows)-1], ",")
	lastIsCombined := false

	if len(earlyRow) > filenameColumnIdx {
		if strings.HasSuffix(earlyRow[filenameColumnIdx], "_Combined") {
			earlyIsCombined = true
		}
	}

	if len(midRow) > filenameColumnIdx {
		if strings.HasSuffix(midRow[filenameColumnIdx], "_Combined") {
			midIsCombined = true
		}
	}

	if len(lastRow) > filenameColumnIdx {
		if strings.HasSuffix(lastRow[filenameColumnIdx], "_Combined") {
			lastIsCombined = true
		}
	}

	return !earlyIsCombined && !midIsCombined && !lastIsCombined
}
