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
	"sort"

	protos "github.com/pixlise/core/generated-protos"
)

func Example_scanHousekeepingData() {
	data := [][]string{
		{"ONE", "TWO", "PMC", "THREE", "FOUR"},
		{"13", "14", "34", "3.1415926", "44"},
		{"13", "13.33", "999", "55", "N/A"},
		{"Some other header"},
		{"TABLE", "TWO", "GOES", "HERE", "DUDE"},
		{"1", "2", "3", "4", "5"},
	}

	headers, pmcCol, dataTypes, rowCount := scanHousekeepingData(data)
	fmt.Printf("%v|%v|%v|%v\n", headers, pmcCol, dataTypes, rowCount)

	data = [][]string{
		{"ONE", "TWO", "PMC", "THREE"},
		{"13", "14", "34", "3.1415926"},
		{"13", "11", "999", "Fifty-Five"},
		{"1", "2", "3", "4"},
	}

	headers, pmcCol, dataTypes, rowCount = scanHousekeepingData(data)
	fmt.Printf("%v|%v|%v|%v\n", headers, pmcCol, dataTypes, rowCount)

	// Output:
	// [ONE TWO THREE FOUR]|2|[MT_INT MT_FLOAT MT_INT MT_FLOAT MT_STRING]|3
	// [ONE TWO THREE]|2|[MT_INT MT_INT MT_INT MT_STRING]|4
}

func Example_convertHousekeepingData() {
	data := [][]string{
		{"13", "14", "34", "3.1415926", "44"},
		{"13", "13.33", "999", "55", "N/A"},
	}

	result, err := convertHousekeepingData(
		[]string{"ONE", "TWO", "THREE", "FOUR"},
		2,
		data,
		[]protos.Experiment_MetaDataType{protos.Experiment_MT_INT, protos.Experiment_MT_FLOAT, protos.Experiment_MT_INT, protos.Experiment_MT_FLOAT, protos.Experiment_MT_STRING},
	)

	fmt.Printf("%v|%v|%v\n", err, result.Header, len(result.Data))

	// Print in increasing PMC order, map ordering is non-deterministic
	keys := []int{}
	for k := range result.Data {
		keys = append(keys, int(k))
	}
	sort.Ints(keys)

	for _, pmc := range keys {
		hks := result.Data[int32(pmc)]
		fmt.Printf("%v: %v\n", pmc, hks)
	}

	// Output:
	// <nil>|[ONE TWO THREE FOUR]|2
	// 34: [{ 13 0 MT_INT} { 0 14 MT_FLOAT} { 0 3.1415925 MT_FLOAT} {44 0 0 MT_STRING}]
	// 999: [{ 13 0 MT_INT} { 0 13.33 MT_FLOAT} { 0 55 MT_FLOAT} {N/A 0 0 MT_STRING}]
}
