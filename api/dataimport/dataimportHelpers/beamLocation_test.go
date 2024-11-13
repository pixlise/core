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

package dataImportHelpers

import (
	"fmt"

	"github.com/pixlise/core/v4/core/logger"
)

func Example_splitColumnHeader() {
	pmc, data, ij, err := splitColumnHeader("PMC_333_corr_i")
	fmt.Printf("%v,%v,%v|%v\n", pmc, data, ij, err)

	pmc, data, ij, err = splitColumnHeader("PMC_777_MCC_j")
	fmt.Printf("%v,%v,%v|%v\n", pmc, data, ij, err)

	pmc, data, ij, err = splitColumnHeader("PMC_777_MC_j")
	fmt.Printf("%v,%v,%v|%v\n", pmc, data, ij, err)

	pmc, data, ij, err = splitColumnHeader("PMC_bob_MCC_j")
	fmt.Printf("%v,%v,%v|%v\n", pmc, data, ij, err)

	pmc, data, ij, err = splitColumnHeader("PMC_777_MCC_k")
	fmt.Printf("%v,%v,%v|%v\n", pmc, data, ij, err)

	pmc, data, ij, err = splitColumnHeader("PMc_777_MCC_j")
	fmt.Printf("%v,%v,%v|%v\n", pmc, data, ij, err)

	pmc, data, ij, err = splitColumnHeader("nan")
	fmt.Printf("%v,%v,%v|%v\n", pmc, data, ij, err)

	// Output:
	// 333,corr,i|<nil>
	// 777,MCC,j|<nil>
	// 0,,|Unexpected column: PMC_777_MC_j
	// 0,,|Unexpected column: PMC_bob_MCC_j
	// 0,,|Unexpected column: PMC_777_MCC_k
	// 0,,|Unexpected column: PMc_777_MCC_j
	// 0,,|Unexpected column: nan
}

func Example_parseBeamLocationHeaders() {
	cols, geom_corr, err := parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// With optional geom_corr
	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "geom_corr", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// With optional geom_corr mixed amongst ijs (We used to expect this to fail, but parser now accepts geom_corr anywhere)
	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "geom_corr", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// With optional geom_corr and another mixed amongst ijs (SHOULD FAIL)
	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "geom_corr", "PMC_3027_MCC_i", "SCLK", "PMC_3027_MCC_j"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, false, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "image_i", "image_j"}, false, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "image_i", "image_j", "SCLK"}, false, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "image_i", "SCLK", "image_j"}, false, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_3027_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_j"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "SCLK", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "SCLK", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j"}, true, 444, []string{})
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// Output:
	// [{777 4 5} {3027 6 7}]|-1|<nil>
	// [{777 5 6} {3027 7 8}]|4|<nil>
	// [{777 4 5} {3027 7 8}]|6|<nil>
	// []|6|Unexpected count of i/j columns
	// []|-1|Expected column image_i, got: PMC_777_MCC_i
	// [{444 4 5}]|-1|<nil>
	// [{444 4 5}]|-1|<nil>
	// []|-1|Expected column image_j, got: SCLK
	// []|-1|Unexpected column header PMC_3027_MCC_i after PMC_777_MCC_i
	// []|-1|Unexpected count of i/j columns
	// []|-1|Unexpected column: SCLK
	// []|-1|Expected column x, got: SCLK
}

func Example_parseBeamLocationRow() {
	pmc, beam, err := parseBeamLocationRow([]string{"33", "1.1", "1.2", "1.3", "55.1", "55.2"}, []pmcColIdxs{{20, 4, 5}}, -1)
	fmt.Printf("%v,%v|%v\n", pmc, beam, err)

	pmc, beam, err = parseBeamLocationRow([]string{"33", "1.1", "1.2", "1.3", "55.1", "55.2", "444", "8.1", "lala", "8.2"}, []pmcColIdxs{{777, 4, 5}, {320, 7, 9}}, -1)
	fmt.Printf("%v,%v|%v\n", pmc, beam, err)

	pmc, beam, err = parseBeamLocationRow([]string{"33", "1.1", "1.2", "1.3", "0.983", "55.1", "55.2", "444", "8.1", "lala", "8.2"}, []pmcColIdxs{{777, 5, 6}, {320, 8, 10}}, 4)
	fmt.Printf("%v,%v|%v\n", pmc, beam, err)

	// Output:
	// 33,{1.1 1.2 1.3 0 map[20:{55.1 55.2}]}|<nil>
	// 33,{1.1 1.2 1.3 0 map[320:{8.1 8.2} 777:{55.1 55.2}]}|<nil>
	// 33,{1.1 1.2 1.3 0.983 map[320:{8.1 8.2} 777:{55.1 55.2}]}|<nil>
}

func Example_parseBeamLocations() {
	data, pmcs, err := parseBeamLocations([][]string{{"PMC", "x", "y", "z", "image_i", "image_j"}, {"33", "1.1", "1.2", "1.3", "55.1", "55.2"}}, false, 222, []string{})
	fmt.Printf("%v|%v|%v\n", data, pmcs, err)

	data, pmcs, err = parseBeamLocations([][]string{
		{"PMC", "x", "y", "z", "PMC_22_MCC_i", "PMC_22_MCC_j", "PMC_62_MCC_i", "PMC_62_MCC_j"},
		{"33", "31.1", "31.2", "31.3", "355.1", "355.2", "3121.4", "3121.5"},
		{"66", "91.1", "91.2", "91.3", "955.1", "955.2", "9121.4", "9121.5"},
	}, true, 333, []string{})
	fmt.Printf("%v|%v|%v\n", data, pmcs, err)

	data, pmcs, err = parseBeamLocations([][]string{
		{"PMC", "x", "y", "z", "geom_corr", "PMC_22_MCC_i", "PMC_22_MCC_j", "PMC_62_MCC_i", "PMC_62_MCC_j"},
		{"33", "31.1", "31.2", "31.3", "1.03", "355.1", "355.2", "3121.4", "3121.5"},
		{"66", "91.1", "91.2", "91.3", "0.99", "955.1", "955.2", "9121.4", "9121.5"},
	}, true, 333, []string{})
	fmt.Printf("%v|%v|%v\n", data, pmcs, err)

	data, pmcs, err = parseBeamLocations([][]string{
		{"PMC", "x", "y", "z", "PMC_22_MCC_i", "PMC_22_MCC_j", "geom_corr", "PMC_62_MCC_i", "PMC_62_MCC_j"},
		{"33", "31.1", "31.2", "31.3", "355.1", "355.2", "1.03", "3121.4", "3121.5"},
		{"66", "91.1", "91.2", "91.3", "955.1", "955.2", "0.99", "9121.4", "9121.5"},
	}, true, 333, []string{})
	fmt.Printf("%v|%v|%v\n", data, pmcs, err)

	// Output:
	// map[33:{1.1 1.2 1.3 0 map[222:{55.1 55.2}]}]|[222]|<nil>
	// map[33:{31.1 31.2 31.3 0 map[22:{355.1 355.2} 62:{3121.4 3121.5}]} 66:{91.1 91.2 91.3 0 map[22:{955.1 955.2} 62:{9121.4 9121.5}]}]|[22 62]|<nil>
	// map[33:{31.1 31.2 31.3 1.03 map[22:{355.1 355.2} 62:{3121.4 3121.5}]} 66:{91.1 91.2 91.3 0.99 map[22:{955.1 955.2} 62:{9121.4 9121.5}]}]|[22 62]|<nil>
	// map[33:{31.1 31.2 31.3 1.03 map[22:{355.1 355.2} 62:{3121.4 3121.5}]} 66:{91.1 91.2 91.3 0.99 map[22:{955.1 955.2} 62:{9121.4 9121.5}]}]|[22 62]|<nil>
}

// This is kind of redunant, was already tested elsewhere, but this is an easy point to add a file and run the test to make sure it imports!
func Example_ReadBeamLocationsFile() {
	rxl, pmcs, err := ReadBeamLocationsFile("./test-data/GeomCorrMoved.CSV", true, 4, []string{}, &logger.StdOutLoggerForTest{})
	fmt.Printf("%v\npmcs: %v\nrxl: %v\n", err, pmcs, len(rxl))

	rxl, pmcs, err = ReadBeamLocationsFile("./test-data/GeomCorrAtExpected.CSV", true, 4, []string{}, &logger.StdOutLoggerForTest{})
	fmt.Printf("%v\npmcs: %v\nrxl: %v\n", err, pmcs, len(rxl))

	rxl, pmcs, err = ReadBeamLocationsFile("./test-data/MissingJ.CSV", true, 4, []string{}, &logger.StdOutLoggerForTest{})
	fmt.Printf("%v\npmcs: %v\nrxl: %v\n", err, pmcs, len(rxl))

	// Output:
	// <nil>
	// pmcs: [4 457 5 458]
	// rxl: 3
	// <nil>
	// pmcs: [4 457 5 458]
	// rxl: 3
	// Unexpected count of i/j columns
	// pmcs: []
	// rxl: 0
}
