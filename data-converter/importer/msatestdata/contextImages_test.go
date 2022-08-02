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

package msatestdata

import (
	"fmt"
	"sort"
)

func Example_getContextImagesPerPMCFromListing() {
	listing := []string{"../datasets/FM_5x5/P13177_5x5_190602/hk.txt", "../datasets/FM_5x5/P13177_5x5_190602/0612747347_000001C5_005048.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612756313_000001C5_005651.jpg", "../datasets/FM_5x5/P13177_5x5_190602/.DS_Store", "../datasets/FM_5x5/P13177_5x5_190602/0612744373_000001C5_004847.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612735422_000001C5_004244.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612758681_000001C5_005807.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612741400_000001C5_004646.jpg", "../datasets/FM_5x5/P13177_5x5_190602/hk_data.csv", "../datasets/FM_5x5/P13177_5x5_190602/0612732390_000001C5_004042.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612750353_000001C5_005249.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612738397_000001C5_004445.jpg", "../datasets/FM_5x5/P13177_5x5_190602/0612753334_000001C5_005450.jpg"}
	results := getContextImagesPerPMCFromListing(listing)

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
