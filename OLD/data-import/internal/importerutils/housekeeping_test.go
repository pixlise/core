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
	"fmt"
	"sort"

	protos "github.com/pixlise/core/v3/generated-protos"
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
