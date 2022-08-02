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
