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

package output

import (
	"fmt"
	"os"
)

func Example_convertTiffToPNG() {
	pngFile, err := os.CreateTemp("", "png")
	pngPath := pngFile.Name()
	err = convertTiffToPNG("./test-files/PCW_D141T0654328732_000RCM_N001000022000045301600LUJ01.TIF", pngPath)
	fileInfo, err2 := os.Stat(pngPath)

	//fmt.Println(pngPath)
	fmt.Printf("%v\n", err)
	fmt.Printf("%v\n", err2)
	fmt.Printf("%v\n", fileInfo.Size() > 234000 && fileInfo.Size() < 334000)

	// Output:
	// <nil>
	// <nil>
	// true
}
