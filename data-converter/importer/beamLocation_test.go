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
	cols, geom_corr, err := parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// With optional geom_corr
	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "geom_corr", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// With optional geom_corr mixed amongst ijs (SHOULD FAIL)
	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "geom_corr", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// With optional geom_corr and another mixed amongst ijs (SHOULD FAIL)
	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "geom_corr", "PMC_3027_MCC_i", "SCLK", "PMC_3027_MCC_j"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i", "PMC_3027_MCC_j"}, false, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "image_i", "image_j"}, false, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "image_i", "image_j", "SCLK"}, false, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "image_i", "SCLK", "image_j"}, false, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_3027_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_j"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "x", "y", "z", "SCLK", "PMC_777_MCC_i", "PMC_777_MCC_j", "PMC_3027_MCC_i"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	cols, geom_corr, err = parseBeamLocationHeaders([]string{"PMC", "SCLK", "y", "z", "PMC_777_MCC_i", "PMC_777_MCC_j"}, true, 444)
	fmt.Printf("%v|%v|%v\n", cols, geom_corr, err)

	// Output:
	// [{777 4 5} {3027 6 7}]|-1|<nil>
	// [{777 5 6} {3027 7 8}]|4|<nil>
	// []|-1|Unexpected count of i/j columns
	// []|-1|Unexpected column: geom_corr
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
	data, err := parseBeamLocations([][]string{[]string{"PMC", "x", "y", "z", "image_i", "image_j"}, []string{"33", "1.1", "1.2", "1.3", "55.1", "55.2"}}, false, 222)
	fmt.Printf("%v|%v\n", data, err)

	data, err = parseBeamLocations([][]string{
		[]string{"PMC", "x", "y", "z", "PMC_22_MCC_i", "PMC_22_MCC_j", "PMC_62_MCC_i", "PMC_62_MCC_j"},
		[]string{"33", "31.1", "31.2", "31.3", "355.1", "355.2", "3121.4", "3121.5"},
		[]string{"66", "91.1", "91.2", "91.3", "955.1", "955.2", "9121.4", "9121.5"},
	}, true, 333)
	fmt.Printf("%v|%v\n", data, err)

	data, err = parseBeamLocations([][]string{
		[]string{"PMC", "x", "y", "z", "geom_corr", "PMC_22_MCC_i", "PMC_22_MCC_j", "PMC_62_MCC_i", "PMC_62_MCC_j"},
		[]string{"33", "31.1", "31.2", "31.3", "1.03", "355.1", "355.2", "3121.4", "3121.5"},
		[]string{"66", "91.1", "91.2", "91.3", "0.99", "955.1", "955.2", "9121.4", "9121.5"},
	}, true, 333)
	fmt.Printf("%v|%v\n", data, err)

	// Output:
	// map[33:{1.1 1.2 1.3 0 map[222:{55.1 55.2}]}]|<nil>
	// map[33:{31.1 31.2 31.3 0 map[22:{355.1 355.2} 62:{3121.4 3121.5}]} 66:{91.1 91.2 91.3 0 map[22:{955.1 955.2} 62:{9121.4 9121.5}]}]|<nil>
	// map[33:{31.1 31.2 31.3 1.03 map[22:{355.1 355.2} 62:{3121.4 3121.5}]} 66:{91.1 91.2 91.3 0.99 map[22:{955.1 955.2} 62:{9121.4 9121.5}]}]|<nil>
}
