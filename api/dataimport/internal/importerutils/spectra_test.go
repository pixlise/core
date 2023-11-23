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

	"github.com/pixlise/core/v3/api/dataimport/internal/dataConvertModels"
	"github.com/pixlise/core/v3/core/logger"
)

func Example_splitSpectraCSVTables_OneTable() {
	lines := [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
		[]string{"PMC", "x", "y", "z"},
		[]string{"33", "10", "20", "30"},
		[]string{"A_1", "A_2", "A_3", "A_4", "A_5", "A_6"},
		[]string{"21", "22", "23", "24", "25", "26"},
		[]string{"B_1", "B_2", "B_3", "B_4", "B_5", "B_6"},
		[]string{"41", "42", "43", "44", "45", "46"},
	}

	data1, data2 := splitSpectraCSVTables(lines)
	fmt.Printf("table1=%v, table2=%v\n", len(data1), len(data2))
	fmt.Printf("%v\n", data1[0])

	// Output:
	// table1=8, table2=0
	// [SCLK_A SCLK_B PMC real_time_A real_time_B live_time_A live_time_B XPERCHAN_A XPERCHAN_B OFFSET_A OFFSET_B]
}

func Example_splitSpectraCSVTables_TwoTable() {
	lines := [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
		[]string{"PMC", "x", "y", "z"},
		[]string{"33", "10", "20", "30"},
		[]string{"A_1", "A_2", "A_3", "A_4", "A_5", "A_6"},
		[]string{"21", "22", "23", "24", "25", "26"},
		[]string{"B_1", "B_2", "B_3", "B_4", "B_5", "B_6"},
		[]string{"41", "42", "43", "44", "45", "46"},
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"21", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
		[]string{"13", "14", "34", "18.7", "18.8", "18.1", "18.2", "18.5", "18.6", "18.3", "18.4"},
		[]string{"PMC", "x", "y", "z"},
		[]string{"23", "10", "20", "30"},
		[]string{"34", "11", "21", "31"},
		[]string{"A_1", "A_2", "A_3", "A_4", "A_5", "A_6"},
		[]string{"31", "22", "23", "24", "25", "26"},
		[]string{"121", "122", "123", "124", "125", "126"},
		[]string{"B_1", "B_2", "B_3", "B_4", "B_5", "B_6"},
		[]string{"31", "42", "43", "44", "45", "46"},
		[]string{"141", "142", "143", "144", "145", "146"},
	}

	data1, data2 := splitSpectraCSVTables(lines)
	fmt.Printf("table1=%v, table2=%v\n", len(data1), len(data2))
	fmt.Printf("%v\n", data1[0])
	fmt.Printf("%v\n", data2[0])
	fmt.Printf("%v\n", data2[1])

	// Output:
	// table1=8, table2=12
	// [SCLK_A SCLK_B PMC real_time_A real_time_B live_time_A live_time_B XPERCHAN_A XPERCHAN_B OFFSET_A OFFSET_B]
	// [SCLK_A SCLK_B PMC real_time_A real_time_B live_time_A live_time_B XPERCHAN_A XPERCHAN_B OFFSET_A OFFSET_B]
	// [21 12 33 17.7 17.8 17.1 17.2 17.5 17.6 17.3 17.4]
}

func Example_parseSpectraCSVData_OK() {
	lines := [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
		[]string{"13", "14", "34", "18.7", "18.8", "18.1", "18.2", "18.5", "18.6", "18.3", "18.4"},
		[]string{"PMC", "x", "y", "z"},
		[]string{"33", "10", "20", "30"},
		[]string{"34", "11", "21", "31"},
		[]string{"A_1", "A_2", "A_3", "A_4", "A_5", "A_6"},
		[]string{"21", "22", "23", "24", "25", "26"},
		[]string{"121", "122", "123", "124", "125", "126"},
		[]string{"B_1", "B_2", "B_3", "B_4", "B_5", "B_6"},
		[]string{"41", "42", "43", "44", "45", "46"},
		[]string{"141", "142", "143", "144", "145", "146"},
	}
	data, err := parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})

	fmt.Printf("%v\n", err)

	combPMCs := []int{}
	for k := range data {
		combPMCs = append(combPMCs, int(k))
	}
	sort.Ints(combPMCs)

	for _, pmc := range combPMCs {
		s := data[int32(pmc)]
		fmt.Printf("pmc[%v]\n", pmc)
		for detIdx := range s {
			fmt.Printf(" det[%v]\n  %v\n", detIdx, s[detIdx].ToString())
		}
	}

	// Output:
	// <nil>
	// pmc[33]
	//  det[0]
	//   meta [DETECTOR_ID:A/s LIVETIME:17.1/f OFFSET:17.3/f PMC:33/i READTYPE:Normal/s REALTIME:17.7/f SCLK:11/i XPERCHAN:17.5/f] spectrum [21 22 23 24 25 26]
	//  det[1]
	//   meta [DETECTOR_ID:B/s LIVETIME:17.2/f OFFSET:17.4/f PMC:33/i READTYPE:Normal/s REALTIME:17.8/f SCLK:12/i XPERCHAN:17.6/f] spectrum [41 42 43 44 45 46]
	// pmc[34]
	//  det[0]
	//   meta [DETECTOR_ID:A/s LIVETIME:18.1/f OFFSET:18.3/f PMC:34/i READTYPE:Normal/s REALTIME:18.7/f SCLK:13/i XPERCHAN:18.5/f] spectrum [121 122 123 124 125 126]
	//  det[1]
	//   meta [DETECTOR_ID:B/s LIVETIME:18.2/f OFFSET:18.4/f PMC:34/i READTYPE:Normal/s REALTIME:18.8/f SCLK:14/i XPERCHAN:18.6/f] spectrum [141 142 143 144 145 146]
}

func Example_combineNormalDwellSpectra_Mismatch() {
	s1 := dataConvertModels.DetectorSampleByPMC{
		3:  []dataConvertModels.DetectorSample{},
		44: []dataConvertModels.DetectorSample{},
	}
	s2 := dataConvertModels.DetectorSampleByPMC{
		82: []dataConvertModels.DetectorSample{},
	}

	_, err := combineNormalDwellSpectra(s1, s2)
	fmt.Printf("%v\n", err)

	// Output:
	// Found dwell spectrum PMC: 82 which has no corresponding normal spectrum
}

func Example_combineNormalDwellSpectra_OK() {
	s1 := dataConvertModels.DetectorSampleByPMC{
		3:  []dataConvertModels.DetectorSample{},
		44: []dataConvertModels.DetectorSample{},
	}
	s2 := dataConvertModels.DetectorSampleByPMC{
		44: []dataConvertModels.DetectorSample{},
	}

	comb, err := combineNormalDwellSpectra(s1, s2)
	fmt.Printf("%v\n", err)

	combPMCs := []int{}
	for k := range comb {
		combPMCs = append(combPMCs, int(k))
	}
	sort.Ints(combPMCs)

	for _, pmc := range combPMCs {
		fmt.Printf("%v\n", pmc)
	}

	// Output:
	// <nil>
	// 3
	// 44
}

func Example_parseSpectraCSVData_TopTableErrors() {
	lines := [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "XPERCHAN_A", "XPERCHAN_B"},
	}
	data, err := parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	lines = [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3"},
	}
	data, err = parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	lines = [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4", "666"},
	}
	data, err = parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	lines = [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "something", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
	}
	data, err = parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	lines = [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "something", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
	}
	data, err = parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	lines = [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "something", "17.5", "17.6", "17.3", "17.4"},
	}
	data, err = parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	// Output:
	// map[]|Unexpected columns in metadata table: [SCLK_A SCLK_B PMC XPERCHAN_A XPERCHAN_B]
	// map[]|row 1 - expected 11 metadata items in row, got: 10
	// map[]|row 1 - expected 11 metadata items in row, got: 12
	// map[]|row 1 - expected SCLK, got: something
	// map[]|row 1 - expected PMC, got: something
	// map[]|row 1 - live_time_B expected float, got: something
}

func Example_parseSpectraCSVData_SpectrumTableDiffColCounts() {
	lines := [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
		[]string{"13", "14", "34", "18.7", "18.8", "18.1", "18.2", "18.5", "18.6", "18.3", "18.4"},
		[]string{"PMC", "x", "y", "z"},
		[]string{"33", "10", "20", "30"},
		[]string{"34", "11", "21", "31"},
		[]string{"A_1", "A_2", "A_3", "A_4", "A_5", "A_6"},
		[]string{"21", "22", "23", "24", "25", "26"},
		[]string{"121", "122", "123", "124", "125", "126"},
		[]string{"B_1", "B_2", "B_3", "B_4", "B_5", "B_6", "B_7"},
		[]string{"41", "42", "43", "44", "45", "46", "47"},
		[]string{"141", "142", "143", "144", "145", "146", "147"},
	}

	data, err := parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	// Output:
	// map[]|row 9 - differing channel count found, A was 6, B is 7
}

func Example_parseSpectraCSVData_SpectrumTableBadData() {
	lines := [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
		[]string{"13", "14", "34", "18.7", "18.8", "18.1", "18.2", "18.5", "18.6", "18.3", "18.4"},
		[]string{"PMC", "x", "y", "z"},
		[]string{"33", "10", "20", "30"},
		[]string{"34", "11", "21", "31"},
		[]string{"A_1", "A_2", "A_3", "A_4", "A_5", "A_6"},
		[]string{"21", "22", "something", "24", "25", "26"},
		[]string{"121", "122", "123", "124", "125", "126"},
		[]string{"B_1", "B_2", "B_3", "B_4", "B_5", "B_6"},
		[]string{"41", "42", "43", "44", "45", "46"},
		[]string{"141", "142", "143", "144", "145", "146"},
	}

	data, err := parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	// Output:
	// map[]|row 7, col 3 - failed to read value, got: something
}

func Example_parseSpectraCSVData_SpectrumTablesDifferingRows() {
	lines := [][]string{
		[]string{"SCLK_A", "SCLK_B", "PMC", "real_time_A", "real_time_B", "live_time_A", "live_time_B", "XPERCHAN_A", "XPERCHAN_B", "OFFSET_A", "OFFSET_B"},
		[]string{"11", "12", "33", "17.7", "17.8", "17.1", "17.2", "17.5", "17.6", "17.3", "17.4"},
		[]string{"13", "14", "34", "18.7", "18.8", "18.1", "18.2", "18.5", "18.6", "18.3", "18.4"},
		[]string{"PMC", "x", "y", "z"},
		[]string{"33", "10", "20", "30"},
		[]string{"34", "11", "21", "31"},
		[]string{"A_1", "A_2", "A_3", "A_4", "A_5", "A_6"},
		[]string{"21", "22", "23", "24", "25", "26"},
		[]string{"121", "122", "123", "124", "125", "126"},
		[]string{"B_1", "B_2", "B_3", "B_4", "B_5", "B_6"},
		[]string{"41", "42", "43", "44", "45", "46"},
	}

	data, err := parseSpectraCSVData(lines, "Normal", &logger.NullLogger{})
	fmt.Printf("%v|%v\n", data, err)

	// Output:
	// map[]|A table had 2 rows, B had 1
}
