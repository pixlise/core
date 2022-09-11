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

package quantModel

import (
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	datasetModel "github.com/pixlise/core/v2/core/dataset"
	protos "github.com/pixlise/core/v2/generated-protos"
	"google.golang.org/protobuf/proto"
)

// This was converted over from a python program, so it may not be implemented in the most "go-esque" way
// as it has to work the same as python, so tests are the same as for the python program, making functions/
// structure match it too.

func matchPMCsWithDataset(data *csvData, matchPMCDatasetFileName string, matchByCoord bool) error {
	// Open the dataset
	datasetPB, err := datasetModel.ReadDatasetFile(matchPMCDatasetFileName)
	if err != nil {
		return err
	}

	fileNameMetaIndex := -1
	if !matchByCoord {
		// Look up the file name index
		for c, label := range datasetPB.GetMetaLabels() {
			if label == "SOURCEFILE" {
				// Make sure it's of type string
				if datasetPB.GetMetaTypes()[c] != protos.Experiment_MT_STRING {
					return fmt.Errorf("Filename column should be of type string, instead it is %v", datasetPB.GetMetaTypes()[c])
				}
				fileNameMetaIndex = c
				break
			}
		}
	}

	// Make a lookup table of XYZ or filename (as string) to PMC
	pmcLookup := map[string]int32{}

	for _, loc := range datasetPB.GetLocations() {
		locPMC64, err := strconv.ParseInt(loc.GetId(), 10, 32)
		if err != nil {
			return fmt.Errorf("Expected location ID to be integer PMC value, got: %v", loc.GetId())
		}
		locPMC := int32(locPMC64)

		if matchByCoord {
			// Lookup by XYZ coord
			if loc.Beam != nil {
				pmcLookup[xyzString(loc.Beam.X, loc.Beam.Y, loc.Beam.Z)] = locPMC
			}
		} else {
			// Lookup by file name
			found := false
			for _, det := range loc.GetDetectors() {
				for _, meta := range det.GetMeta() {
					if meta.GetLabelIdx() == int32(fileNameMetaIndex) {
						pmcLookup[meta.GetSvalue()] = locPMC
						found = true
						break
					}
				}
			}

			if !found {
				return fmt.Errorf("Failed to find file name in meta for PMC: %v", loc.GetId())
			}
		}
	}

	// Loop through matching values & find the PMC to store in the quant file
	pmcColIdx := -1
	xIdx := -1
	yIdx := -1
	zIdx := -1
	filenameIdx := -1

	for colIdx, col := range data.header {
		switch col {
		case "PMC":
			pmcColIdx = colIdx
		case "X":
			xIdx = colIdx
		case "Y":
			yIdx = colIdx
		case "Z":
			zIdx = colIdx
		case "filename":
			filenameIdx = colIdx
		}
	}

	// If these columns don't exist in our PMC, fail here
	if pmcColIdx == -1 {
		// Add this column to the CSV data we read in
		pmcColIdx = len(data.header)
		data.header = append(data.header, "PMC")
		log.Println("CSV does not contain PMC column, adding one in memory to store matched PMCs into")
	} else if matchByCoord && (xIdx == -1 || yIdx == -1 || zIdx == -1) {
		return fmt.Errorf("PMC matching failed: CSV does not contain X/Y/Z columns")
	}

	// Look up the PMC by whatever method selected (see matchByCoord), and store it in the PMC column of the CSV data
	for rowIdx, row := range data.data {
		lookupValue := ""

		if matchByCoord {
			fX, xerr := strconv.ParseFloat(row[xIdx], 32)
			fY, yerr := strconv.ParseFloat(row[yIdx], 32)
			fZ, zerr := strconv.ParseFloat(row[zIdx], 32)

			if xerr != nil || yerr != nil || zerr != nil {
				return fmt.Errorf("matchPMCsWithDataset Failed to read row %v XYZ (%v,%v,%v) coord for matching", rowIdx, row[xIdx], row[yIdx], row[zIdx])
			}

			lookupValue = xyzString(float32(fX), float32(fY), float32(fZ))
		} else {
			lookupValue = row[filenameIdx]
		}

		if pmc, ok := pmcLookup[lookupValue]; ok {
			strPMC := strconv.Itoa(int(pmc))
			// If the row doesn't contain a PMC, we add it
			if pmcColIdx == len(row) {
				data.data[rowIdx] = append(row, strPMC)
			} else {
				data.data[rowIdx][pmcColIdx] = strPMC
			}
		} else {
			return fmt.Errorf("matchPMCsWithDataset Failed to match %v to a PMC in dataset file", lookupValue)
		}
	}

	return nil
}

func filterListItems(stringList []string, indexToSkip map[int]bool) []string {
	result := make([]string, 0)

	for c, v := range stringList {
		if !indexToSkip[c] {
			result = append(result, v)
		}
	}

	return result
}

func decodeMapFileNameColumn(fileName string) (string, string, error) {
	ext := filepath.Ext(fileName)
	fileNameBits := strings.Split(fileName, "_")

	parsedREADTYPE := ""
	parsedDETECTOR_ID := ""

	if ext == "" {
		// Assume it's from PIQUANT having read a PIXLISE bin file, and it's just composed of READTYPE_DETECTORID
		// NOTE: we support Normal_A and Normal_A_roiID now that roiID can optionally be appended there
		if len(fileNameBits) == 2 || len(fileNameBits) == 3 {
			parsedREADTYPE = fileNameBits[0]
			parsedDETECTOR_ID = fileNameBits[1]
		}
	} else if strings.ToUpper(ext) == ".MSA" && len(fileNameBits) == 5 {
		// Here we try to parse the MSA file names of test datasets from EM, found in 5x5, 5x11 and EM cal target
		parsedREADTYPE = fileNameBits[0]
		parsedDETECTOR_ID = fileNameBits[1]
	}

	// See if what we found is valid
	// NOTE: Mixed is something we added, PIQUANT outputs this if there was a combination of READTYPE to form a PMC, eg Normals and Dwells
	if parsedREADTYPE != "Normal" && parsedREADTYPE != "Dwell" && parsedREADTYPE != "BulkSum" && parsedREADTYPE != "MaxValue" && parsedREADTYPE != "Mixed" {
		return "", "", fmt.Errorf("decodeMapFileNameColumn: Invalid READTYPE in filename: \"%v\"", fileName)
	}
	if parsedDETECTOR_ID != "A" && parsedDETECTOR_ID != "B" && parsedDETECTOR_ID != "Combined" {
		return "", "", fmt.Errorf("decodeMapFileNameColumn: Invalid DETECTOR_ID in filename: \"%v\"", fileName)
	}

	return parsedREADTYPE, parsedDETECTOR_ID, nil
}

func xyzString(x float32, y float32, z float32) string {
	return fmt.Sprintf("%.2f,%.2f,%.2f", x, y, z)
}

func getInterestingColIndexes(header []string, colNameList []string) (map[string]int, error) {
	// Find indexes for what we're interested in
	interestingColIdxs := map[string]int{}

	for _, col := range colNameList {
		interestingColIdxs[col] = -1
	}

	seenHeaderCols := map[string]bool{}
	found := 0
	for c, col := range header {
		// Check if it's one of the interesting columns, if so, save its index
		for name := range interestingColIdxs {
			if col == name {
				interestingColIdxs[name] = c
				found++
				break
			}
		}

		// Scan for duplicate column names while we're at it
		if seenHeaderCols[col] {
			return nil, fmt.Errorf("Duplicate CSV column: %v", col)
		}
		seenHeaderCols[col] = true
	}

	// Check we got all interesting columns in the CSV
	for name, idx := range interestingColIdxs {
		if idx == -1 {
			return nil, fmt.Errorf("CSV column missing: %v", name)
		}
	}

	return interestingColIdxs, nil
}

func getElements(columnLabels []string) []string {
	elements := make([]string, 0)

	for _, label := range columnLabels {
		if strings.HasSuffix(label, "_%") {
			elements = append(elements, label[0:len(label)-2])
		}
	}

	return elements
}

type quantLoc struct {
	pmc      int32
	rtt      int32
	sclk     int32
	filename string

	dataValues []string
}

type quantData struct {
	labels    []string
	types     []string
	locations []quantLoc
}

func makeColumnTypeList(csv csvData, colsToIgnore map[int]bool) ([]string, error) {
	result := make([]string, 0)

	// Using the first row...
	if len(csv.data) <= 0 {
		return result, errors.New("No data found in CSV")
	}

	colsRead := len(csv.data[0])

	// Iterate through data by column
	for colIdx := 0; colIdx < colsRead; colIdx++ {
		if !colsToIgnore[colIdx] {
			floatFound := false

			// Check if we have all floats or all ints in this column
			for rowIdx, row := range csv.data {
				value := row[colIdx]
				_, ierr := strconv.ParseInt(value, 10, 32)
				_, ferr := strconv.ParseFloat(value, 32)

				// If neither, something is wrong
				if ierr != nil && ferr != nil {
					return result, fmt.Errorf("Failed to parse \"%v\" as float or int at col %v/row %v", value, colIdx, rowIdx)
				}

				// If float, we found one, so whole col is float
				if ierr != nil && ferr == nil {
					floatFound = true
					break
				}
			}

			// If it's a float, remember this, else it's int
			t := "I"
			if floatFound {
				t = "F"
			}

			result = append(result, t)
		}
	}

	return result, nil
}

func convertQuantificationData(csv csvData, expectMetaColumns []string) (quantData, error) {
	var result quantData

	if len(csv.data) <= 0 {
		return result, errors.New("Expected at least 1 data row")
	}

	// Returns a dict with the column name, and the index of the columns specified
	interestingColIdxs, err := getInterestingColIndexes(csv.header, expectMetaColumns)
	if err != nil {
		return result, err
	}

	// If we have to skip any indexes, put them in the map
	indexToSkip := make(map[int]bool, 0)
	for _, v := range interestingColIdxs {
		if v > -1 {
			indexToSkip[v] = true
		}
	}

	//interestingColIdxsOnly = list(interestingColIdxs.values())

	// Get only the labels that are not in the "Interesting" list above
	result.labels = filterListItems(csv.header, indexToSkip)

	// Get data types for the non "Interesting" columns, ie for each element data column, like Fe_%, Fe_int, Fe_err and things like chisq, eVstart, etc.
	result.types, err = makeColumnTypeList(csv, indexToSkip)
	if err != nil {
		return result, err
	}

	// Read rows, separating columns into the "interesting" ones and the rest as "data"
	for _, row := range csv.data {
		loc, err := makeQuantedLocation(csv.header, row, indexToSkip)
		if err != nil {
			return result, err
		}
		result.locations = append(result.locations, loc)
	}

	return result, nil
}

func makeQuantedLocation(header []string, row []string, metaColumns map[int]bool) (quantLoc, error) {
	// Find the "metadata" values
	metaLookup := make(map[string]string, 0)

	for colIdx := range metaColumns {
		metaLookup[header[colIdx]] = row[colIdx]
	}

	// Set the meta values, if they were specified, otherwise we stick to their "zero value"
	var result quantLoc

	colNameExpected := "PMC"
	strValue, ok := metaLookup[colNameExpected]
	if ok {
		iValue, err := strconv.ParseInt(strValue, 10, 32)
		if err != nil {
			return result, fmt.Errorf("%v is not int: %v", colNameExpected, strValue)
		}
		result.pmc = int32(iValue)
	}

	colNameExpected = "SCLK"
	strValue, ok = metaLookup[colNameExpected]
	if ok {
		iValue, err := strconv.ParseInt(strValue, 10, 32)
		if err != nil {
			return result, fmt.Errorf("%v is not int: %v", colNameExpected, strValue)
		}
		result.sclk = int32(iValue)
	}

	colNameExpected = "RTT"
	strValue, ok = metaLookup[colNameExpected]
	if ok {
		iValue, err := strconv.ParseInt(strValue, 10, 32)
		if err != nil {
			return result, fmt.Errorf("%v is not int: %v", colNameExpected, strValue)
		}
		result.rtt = int32(iValue)
	}

	colNameExpected = "filename"
	strValue, ok = metaLookup[colNameExpected]
	if ok {
		result.filename = strValue
	}

	result.dataValues = make([]string, 0)
	for idx, value := range row {
		if !metaColumns[idx] {
			result.dataValues = append(result.dataValues, value)
		}
	}

	return result, nil
}

type csvData struct {
	header []string
	data   [][]string
}

func readCSV(data string, headerRowIdx int) (csvData, error) {
	var result csvData

	// If we have rows to ignore, do that before we get into CSV parsing
	for c := 0; c < headerRowIdx; c++ {
		idx := strings.Index(data, "\n")
		if idx == -1 {
			return result, fmt.Errorf("Failed to skip %v lines before header in CSV", headerRowIdx)
		}
		data = data[idx+1:]
	}

	r := csv.NewReader(strings.NewReader(data))
	r.TrimLeadingSpace = true

	result.data = make([][]string, 0)

	for c := 0; true; c++ {
		record, err := r.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return result, err
		}

		if c == 0 {
			// This is the header row!
			result.header = record
		} else {
			// And the rest
			result.data = append(result.data, record)
		}
	}

	return result, nil
}

// Verifying that parsing floats works as we need, because we have some floats come back from piquant in interesting ways
func parseFloatColumnValue(val string) (float32, error) {
	if val == "-nan" {
		val = "nan"
	}
	fVal, err := strconv.ParseFloat(val, 32)
	return float32(fVal), err
}

func saveLocation(loc quantLoc, types []string) (*protos.Quantification_QuantLocation, error) {
	result := &protos.Quantification_QuantLocation{Pmc: loc.pmc, Rtt: loc.rtt, Sclk: loc.sclk}

	// Fill in the data values, checking that type conversion works
	for c, v := range loc.dataValues {
		val := &protos.Quantification_QuantLocation_QuantDataItem{}

		if types[c] == "I" {
			iVal, err := strconv.ParseInt(v, 10, 32)
			if err != nil {
				return result, fmt.Errorf("saveLocation: Failed to convert %v to int: %v", v, err)
			}
			val.Ivalue = int32(iVal)
		} else if types[c] == "F" {
			fVal, err := parseFloatColumnValue(v)
			if err != nil {
				return result, fmt.Errorf("saveLocation: Failed to convert %v to float: %v", v, err)
			}
			val.Fvalue = fVal
		} else {
			return result, fmt.Errorf("saveLocation: Unexpected type %v defined for data column %v", v, c)
		}

		result.Values = append(result.Values, val)
	}

	return result, nil
}

func saveToProto(data quantData, detectorIDSpecified string, detectorDuplicateAB bool) (*protos.Quantification, error) {
	pb := &protos.Quantification{Labels: data.labels}

	// Save labels
	pb.Labels = data.labels

	// Save types
	for _, typ := range data.types {
		toSave := protos.Quantification_QT_INT
		if typ == "F" {
			toSave = protos.Quantification_QT_FLOAT
		}
		pb.Types = append(pb.Types, toSave)
	}

	// Save locations
	locByDetectorID := map[string]*protos.Quantification_QuantLocationSet{}

	// We can be saving for detector ID: A, B or Combined
	locByDetectorID["A"] = &protos.Quantification_QuantLocationSet{Detector: "A"}
	locByDetectorID["B"] = &protos.Quantification_QuantLocationSet{Detector: "B"}
	locByDetectorID["Combined"] = &protos.Quantification_QuantLocationSet{Detector: "Combined"}

	for _, loc := range data.locations {
		// Convert the location data
		locToSave, err := saveLocation(loc, data.types)
		if err != nil {
			return nil, err
		}

		// Figure out if it's to go into the A or B set
		detectorID := detectorIDSpecified
		if len(detectorID) <= 0 {
			_, id, err := decodeMapFileNameColumn(loc.filename)
			if err != nil {
				return nil, err
			}
			detectorID = id
		}

		locByDetectorID[detectorID].Location = append(locByDetectorID[detectorID].Location, locToSave)

		// If we're duplicating, do that
		if len(detectorIDSpecified) > 0 && detectorDuplicateAB {
			// If it was A, add it to B also, and vice-versa
			if detectorID == "A" {
				detectorID = "B"
			} else {
				detectorID = "A"
			}

			locByDetectorID[detectorID].Location = append(locByDetectorID[detectorID].Location, locToSave)
		}
	}

	// Add the per-detector location sets, using sorted map keys
	// TODO: make this work --> detIDs := utils.GetStringMapKeys(locByDetectorID)[]string{}
	detIDs := []string{}
	for k := range locByDetectorID {
		detIDs = append(detIDs, k)
	}
	sort.Strings(detIDs)

	for _, k := range detIDs {
		locs := locByDetectorID[k]
		if len(locs.Location) > 0 {
			pb.LocationSet = append(pb.LocationSet, locs)
		}
	}

	return pb, nil
}

// ConvertQuantificationCSV - converts from incoming string CSV data to serialised binary data
// Returns the serialised quantification bytes and the elements that were quantified
func ConvertQuantificationCSV(logName string, data string, expectMetaColumns []string, matchPMCDatasetFileName string, matchPMCByCoord bool, detectorIDOverride string, detectorDuplicateAB bool) ([]byte, []string, error) {
	mapData, err := readCSV(data, 1)
	if err != nil {
		return []byte{}, []string{}, err
	}

	// Match PMCS if required
	if len(matchPMCDatasetFileName) > 0 {
		if err = matchPMCsWithDataset(&mapData, matchPMCDatasetFileName, matchPMCByCoord); err != nil {
			return []byte{}, []string{}, err
		}

		// We've now created a PMC column, so ensure that it's in the list of expected columns
		hasPMC := false
		for _, col := range expectMetaColumns {
			if col == "PMC" {
				hasPMC = true
				break
			}
		}

		if !hasPMC {
			expectMetaColumns = append(expectMetaColumns, "PMC")
		}
	}

	// Parse/convert it to a form we can save it in
	quantToSave, err := convertQuantificationData(mapData, expectMetaColumns)
	if err != nil {
		return []byte{}, []string{}, err
	}

	log.Println("Data Types Saved:")
	for c, label := range quantToSave.labels {
		log.Printf("  %v as %v\n", label, quantToSave.types[c])
	}

	elements := getElements(mapData.header)
	log.Printf("Elements found: %v\n", elements)

	// Write to bytes
	quantProto, err := saveToProto(quantToSave, detectorIDOverride, detectorDuplicateAB)
	if err != nil {
		return []byte{}, []string{}, err
	}

	out, err := proto.Marshal(quantProto)
	if err != nil {
		return []byte{}, []string{}, fmt.Errorf("Failed to encode quantification protobuf: %v", err)
	}

	return out, elements, nil
}
