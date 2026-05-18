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

package utils

import "fmt"

func Example_makeSaveableFileName() {
	fmt.Println(MakeSaveableFileName("my roi"))
	fmt.Println(MakeSaveableFileName("Dust/Alteration"))
	fmt.Println(MakeSaveableFileName("I bet $100 this is cheese"))
	fmt.Println(MakeSaveableFileName("10% Ca/Fe & Coffee matrix?"))

	// Output:
	// my roi
	// Dust Alteration
	// I bet  100 this is cheese
	// 10% Ca Fe   Coffee matrix
}

func Example_utils_ApplyIndexToFileName() {
	fmt.Println(ApplyIndexToFileName("node.txt", 0, true))
	fmt.Println(ApplyIndexToFileName("node.txt", 1, true))
	fmt.Println(ApplyIndexToFileName("node.txt", 2, true))
	fmt.Println(ApplyIndexToFileName("node.txt", 3, false))
	fmt.Println(ApplyIndexToFileName("node.txt", 304023, true))
	fmt.Println(ApplyIndexToFileName("node.txt", 6304023, true))
	fmt.Println(ApplyIndexToFileName("file.name.img", 33, true))
	fmt.Println(ApplyIndexToFileName("extensionless", 3, true))

	// Output:
	// node000000.txt
	// node000001.txt
	// node000002.txt
	// node.txt
	// node304023.txt
	// node6304023.txt
	// file000033.name.img
	// extensionless000003
}
