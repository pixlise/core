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
)

func Example_parsePsuedoIntensityData() {
	pmcTableHeader := []string{"PMC", "x", "y", "z"}
	pmcTableData1 := []string{"77", "1", "2", "3"}
	pmcTableData2 := []string{"78", "4", "3", "2"}

	psHeader := []string{"pi1", "pi2", "pi3", "pi4", "pi5", "pi6"}
	psTableData1 := []string{"0.1", "0.2", "0.3", "0.4", "0.5", "0.6"}
	psTableData2 := []string{"10.1", "10.2", "10.3", "10.4", "10.5", "10.6"}

	csvData := [][]string{pmcTableHeader, pmcTableData1, pmcTableData2, psHeader, psTableData1, psTableData2}
	data, err := parsePsuedoIntensityData(csvData)
	fmt.Printf("%v|%v\n", err, len(data))
	fmt.Printf("%v\n", data[77])
	fmt.Printf("%v\n", data[78])

	csvData = [][]string{pmcTableData1, pmcTableData2, psHeader, psTableData1, psTableData2}
	data, err = parsePsuedoIntensityData(csvData)
	fmt.Printf("%v|%v\n", err, data)

	csvData = [][]string{pmcTableHeader, pmcTableData1, []string{"oops", "1", "2", "3"}, psHeader, psTableData1, psTableData2}
	data, err = parsePsuedoIntensityData(csvData)
	fmt.Printf("%v|%v\n", err, data)

	csvData = [][]string{pmcTableHeader, pmcTableData1, pmcTableData1, psHeader, psTableData1, []string{"10.1", "10.2", "wtf", "10.4", "10.5", "10.6"}}
	data, err = parsePsuedoIntensityData(csvData)
	fmt.Printf("%v|%v\n", err, data)

	// Output:
	// <nil>|2
	// [0.1 0.2 0.3 0.4 0.5 0.6]
	// [10.1 10.2 10.3 10.4 10.5 10.6]
	// expected first table to contain PMCs in first column, found: 77|map[]
	// row 2 - expected PMC, got: oops|map[]
	// row 5, col 3 - expected pseudointensity value, got: wtf|map[]
}
