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

package jplbreadboard

import (
	"fmt"
	"sort"

	"github.com/pixlise/core/v2/core/logger"
)

func Example_getContextImagesPerPMCFromListing() {
	listing := []string{"../datasets/FM_5x5/P13177_5x5_190602/hk.txt", "../datasets/FM_5x5/P13177_5x5_190602/0612747347_000001C5_005048.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612756313_000001C5_005651.jpg", "../datasets/FM_5x5/P13177_5x5_190602/.DS_Store", "../datasets/FM_5x5/P13177_5x5_190602/0612744373_000001C5_004847.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612735422_000001C5_004244.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612758681_000001C5_005807.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612741400_000001C5_004646.jpg", "../datasets/FM_5x5/P13177_5x5_190602/hk_data.csv", "../datasets/FM_5x5/P13177_5x5_190602/0612732390_000001C5_004042.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612750353_000001C5_005249.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612738397_000001C5_004445.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612753334_000001C5_005450.jpg"}
	results := getContextImagesPerPMCFromListing(listing, &logger.StdOutLogger{})

	fmt.Printf("length: %d\n", len(results))

	// Print in sorted order so we match output (go maps may not be storing these in ascending PMC order)
	lines := []string{}
	for k, v := range results {
		lines = append(lines, fmt.Sprintf("[%d]=%v", k, v))
	}

	sort.Strings(lines)
	for _, l := range lines {
		fmt.Println(l)
	}

	// Output:
	// length: 10
	// [4042]=0612732390_000001C5_004042.jpg
	// [4244]=0612735422_000001C5_004244.jpg
	// [4445]=0612738397_000001C5_004445.jpg
	// [4646]=0612741400_000001C5_004646.jpg
	// [4847]=0612744373_000001C5_004847.jpg
	// [5048]=0612747347_000001C5_005048.jpg
	// [5249]=0612750353_000001C5_005249.jpg
	// [5450]=0612753334_000001C5_005450.jpg
	// [5651]=0612756313_000001C5_005651.jpg
	// [5807]=0612758681_000001C5_005807.jpg
}
