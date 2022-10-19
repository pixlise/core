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
)

func Example_parseRanges() {
	rangeHeader := []string{"Name", "StartChannel", "EndChannel"}
	range1 := []string{"ps1", "100", "120"}
	range2 := []string{"ps2", "144", "173"}

	csvData := [][]string{rangeHeader, range1, range2}
	data, err := parseRanges(csvData)
	fmt.Printf("%v|%v\n", err, len(data))
	fmt.Printf("%+v\n", data[0])
	fmt.Printf("%+v\n", data[1])

	csvData = [][]string{[]string{"Date", "StartChannel", "EndChannel"}, range1, range2}
	data, err = parseRanges(csvData)
	fmt.Printf("%v|%v\n", err, data)
	// Output:
	// <nil>|2
	// {Name:ps1 Start:100 End:120}
	// {Name:ps2 Start:144 End:173}
	// Pseudo-intensity ranges has unexpected headers|[]
}
