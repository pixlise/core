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

package quantification

import (
	"fmt"

	"github.com/pixlise/core/v3/core/logger"
)

func Example_filterListItems() {
	// Should just filter indexes that are valid
	idxToIgnoreMap := map[int]bool{
		-9: true,
		1:  true,
		2:  true,
		5:  true,
		6:  true,
	}

	fmt.Println(filterListItems([]string{"snowboarding", "is", "awesome", "says", "Peter", "Nemere"}, idxToIgnoreMap))
	// Output: [snowboarding says Peter]
}

func Example_getInterestingColIndexes() {
	header := []string{"PMC", "K_%", "Ca_%", "Fe_%", "K_int", "Ca_int", "Fe_int", "K_err", "Ca_err", "Fe_err", "total_counts", "livetime", "chisq", "eVstart", "eV/ch", "res", "iter", "filename", "Events", "Triggers", "SCLK", "RTT"}
	interesting, err := getInterestingColIndexes(header, []string{"PMC", "filename", "SCLK", "RTT"})
	fmt.Printf("\"%v\" \"%v\"\n", interesting, err)
	interesting, err = getInterestingColIndexes(header, []string{"K_%", "total_counts"})
	fmt.Printf("\"%v\" \"%v\"\n", interesting, err)

	// Bad cases
	interesting, err = getInterestingColIndexes(header, []string{"PMC", "TheFileName", "SCLK", "RTT"})
	fmt.Printf("\"%v\" \"%v\"\n", interesting, err)
	header[5] = "SCLK"
	interesting, err = getInterestingColIndexes(header, []string{"PMC", "TheFileName", "SCLK", "RTT"})
	fmt.Printf("\"%v\" \"%v\"\n", interesting, err)

	// 22 header items...

	// Output:
	// "map[PMC:0 RTT:21 SCLK:20 filename:17]" "<nil>"
	// "map[K_%:1 total_counts:10]" "<nil>"
	// "map[]" "CSV column missing: TheFileName"
	// "map[]" "Duplicate CSV column: SCLK"
}

func Example_getElements() {
	fmt.Printf("%v", getElements([]string{"PMC", "SCLK", "Ca_%", "Ti_%", "Ca_int", "Ti_int", "livetime", "Mg_%", "chisq"}))
	// Output: [Ca Ti Mg]
}

func Example_makeColumnTypeList() {
	data := csvData{[]string{"a", "b", "c", "d", "e"}, [][]string{[]string{"1.11111", "2", "3.1415962", "5", "6"}}}
	result, err := makeColumnTypeList(data, map[int]bool{2: true, 3: true})
	fmt.Printf("%v|%v\n", result, err)
	result, err = makeColumnTypeList(data, map[int]bool{})
	fmt.Printf("%v|%v\n", result, err)

	// Bad type
	data = csvData{[]string{"a", "b", "c", "d", "e"}, [][]string{[]string{"1.11111", "Wanaka", "3.1415962", "5"}}}
	result, err = makeColumnTypeList(data, map[int]bool{2: true, 3: true})
	fmt.Printf("%v|%v\n", result, err)

	// Skipping the string 1 should make it work...
	result, err = makeColumnTypeList(data, map[int]bool{1: true, 3: true})
	fmt.Printf("%v|%v\n", result, err)
	// Output:
	// [F I I]|<nil>
	// [F I F I I]|<nil>
	// [F]|Failed to parse "Wanaka" as float or int at col 1/row 0
	// [F F]|<nil>
}

func Example_makeQuantedLocation() {
	// Should just filter indexes that are valid
	fmt.Println(makeQuantedLocation([]string{"Ca_%", "PMC", "Ti_%", "RTT", "filename", "Ca_int"}, []string{"1.11111", "2", "3.1415962", "5", "FileA.msa", "6"}, map[int]bool{1: true, 3: true, 4: true}))
	// Output:
	// {2 5 0 FileA.msa [1.11111 3.1415962 6]} <nil>
}

func Example_convertQuantificationData() {
	data := csvData{
		[]string{"PMC", "Ca_%", "Ca_int", "SCLK", "Ti_%", "filename", "RTT"},
		[][]string{
			[]string{"23", "1.5", "5", "11111", "4", "fileA.msa", "44"},
			[]string{"70", "3.4", "32", "12345", "4.21", "fileB.msa", "45"},
		},
	}

	result, err := convertQuantificationData(data, []string{"PMC", "RTT", "SCLK", "filename"})
	fmt.Printf("%v|%v\n", result, err)

	// Output:
	// {[Ca_% Ca_int Ti_%] [F I F] [{23 44 11111 fileA.msa [1.5 5 4]} {70 45 12345 fileB.msa [3.4 32 4.21]}]}|<nil>
}

func Example_readCSV() {
	csv := `something header
more header
col 1,"col, 2",  col_3
"value one",123, 456
value two,444,555
`
	d, err := readCSV(csv, 2)
	fmt.Printf("%v|%v", d, err)
	// Output: {[col 1 col, 2 col_3] [[value one 123 456] [value two 444 555]]}|<nil>
}

func Example_matchPMCsWithDataset() {
	l := &logger.StdOutLogger{}
	data := csvData{[]string{"X", "Y", "Z", "filename", "Ca_%"}, [][]string{[]string{"1", "0.40", "0", "Roastt_Laguna_Salinas_28kV_230uA_03_03_2020_111.msa", "4.5"}}}

	err := matchPMCsWithDataset(&data, "non-existant-file.bin", true, l)
	if err.Error() == "open non-existant-file.bin: no such file or directory" || err.Error() == "open non-existant-file.bin: The system cannot find the file specified." {
		fmt.Println("open non-existant-file.bin: Failed as expected")
	}

	fmt.Printf("%v, header[%v]=%v, data[%v]=%v\n", matchPMCsWithDataset(&data, "./testdata/LagunaSalinasdataset.bin", true, l), len(data.header)-1, data.header[5], len(data.data[0])-1, data.data[0][5])

	data = csvData{[]string{"X", "Y", "Z", "filename", "Ca_%"}, [][]string{[]string{"1", "930.40", "0", "Roastt_Laguna_Salinas_28kV_230uA_03_03_2020_111.msa", "4.5"}}}
	fmt.Println(matchPMCsWithDataset(&data, "./testdata/LagunaSalinasdataset.bin", true, l))

	data = csvData{[]string{"X", "Y", "Z", "filename", "Ca_%"}, [][]string{[]string{"1", "0.40", "0", "Roastt_Laguna_Salinas_28kV_230uA_03_03_2020_116.msa", "4.5"}}}
	fmt.Printf("%v, header[%v]=%v, data[%v]=%v\n", matchPMCsWithDataset(&data, "./testdata/LagunaSalinasdataset.bin", false, l), len(data.header)-1, data.header[5], len(data.data[0])-1, data.data[0][5])

	// Output:
	// open non-existant-file.bin: Failed as expected
	// <nil>, header[5]=PMC, data[5]=111
	// matchPMCsWithDataset Failed to match 1.00,930.40,0.00 to a PMC in dataset file
	// <nil>, header[5]=PMC, data[5]=116
}

func Example_decodeMapFileNameColumn() {
	rt, det, err := decodeMapFileNameColumn("file.txt")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("Normal_A")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("Normal_A_MyRoiID")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("Dwell_B")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("Normal_C")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("LongRead_B")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("Scotland_something_00012.msa")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("Scotland_something_00012_10keV_33.msa")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	rt, det, err = decodeMapFileNameColumn("Normal_A_0123456789_873495_455.msa")
	fmt.Printf("%v|%v|%v\n", rt, det, err)

	// Output:
	// ||decodeMapFileNameColumn: Invalid READTYPE in filename: "file.txt"
	// Normal|A|<nil>
	// Normal|A|<nil>
	// Dwell|B|<nil>
	// ||decodeMapFileNameColumn: Invalid DETECTOR_ID in filename: "Normal_C"
	// ||decodeMapFileNameColumn: Invalid READTYPE in filename: "LongRead_B"
	// ||decodeMapFileNameColumn: Invalid READTYPE in filename: "Scotland_something_00012.msa"
	// ||decodeMapFileNameColumn: Invalid READTYPE in filename: "Scotland_something_00012_10keV_33.msa"
	// Normal|A|<nil>
}

func Example_parseFloatColumnValue() {
	fVal, err := parseFloatColumnValue("3.1415926")
	fmt.Printf("%v|%v\n", fVal, err)

	fVal, err = parseFloatColumnValue("-3.15")
	fmt.Printf("%v|%v\n", fVal, err)

	fVal, err = parseFloatColumnValue("1.234e02")
	fmt.Printf("%v|%v\n", fVal, err)

	fVal, err = parseFloatColumnValue("")
	fmt.Printf("%v|%v\n", fVal, err)

	fVal, err = parseFloatColumnValue("nan")
	fmt.Printf("%v|%v\n", fVal, err)

	fVal, err = parseFloatColumnValue("-nan")
	fmt.Printf("%v|%v\n", fVal, err)

	// Output:
	// 3.1415925|<nil>
	// -3.15|<nil>
	// 123.4|<nil>
	// 0|strconv.ParseFloat: parsing "": invalid syntax
	// NaN|<nil>
	// NaN|<nil>
}
