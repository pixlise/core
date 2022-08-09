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

package msatestdata

import (
	"fmt"

	"github.com/pixlise/core/data-converter/converterModels"
)

func Example_getMSASeqNo() {
	names := []string{"file0001.msa", "file_0001.msa", "file_1.txt", "Normal_A_0612673072_000001C5_000013.msa", "Normal_A_0612673072_000001C5_0033213.msa", "Some_Thing_With_Spectra_16.msa", "../Path_With/Underscores/Some_Thing_With_Spectra_43.msa"}

	for _, name := range names {
		s, e := getMSASeqNo(name)
		fmt.Printf("%v|%v\n", s, e)
	}

	// Output:
	// 0|Invalid MSA file name: file0001.msa
	// 1|<nil>
	// 0|Unexpected file extension when reading MSA: .txt
	// 13|<nil>
	// 33213|<nil>
	// 16|<nil>
	// 43|<nil>
}

func Example_getSpectraFiles() {
	files := []string{"../something/file.txt", "../something/Normal_B_1_2_3.msa", "../something/BulkSum_B_1_2_3.msa", "../something/Another_B_1_2_3.msa", "../something/Normal_B_1_2.jpg", "Normal_P_1_2_3.msa"}
	f, l := getSpectraFiles(files, true)

	for _, v := range f {
		fmt.Printf(v + "\n")
	}

	for _, v := range l {
		fmt.Printf(v + "\n")
	}

	// Output:
	// ../something/Normal_B_1_2_3.msa
	//../something/BulkSum_B_1_2_3.msa
	// Normal_P_1_2_3.msa
	// Ignoring extension '.txt' in: ../something/file.txt
	// unexpected MSA read type
	// Ignoring extension '.jpg' in: ../something/Normal_B_1_2.jpg
}

func detectorsToString(ds []converterModels.DetectorSample) string {
	result := ""
	for c, d := range ds {
		if c > 0 {
			result += " "
		}
		result += fmt.Sprintf("[%v]", c)
		result += d.ToString()
	}
	return result
}

func Example_addToSpectraLookup() {
	lookup := converterModels.DetectorSampleByPMC{}
	lookup = addToSpectraLookup(lookup,
		converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(123),
			"DETECTOR_ID": converterModels.StringMetaValue("A"),
		},
		[]int64{1, 2, 3},
	)

	fmt.Printf("%+v\n", detectorsToString(lookup[123]))

	lookup = addToSpectraLookup(lookup,
		converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(123),
			"DETECTOR_ID": converterModels.StringMetaValue("B"),
		},
		[]int64{4, 5, 6},
	)

	fmt.Printf("%+v\n", detectorsToString(lookup[123]))

	lookup = addToSpectraLookup(lookup,
		converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(555),
			"DETECTOR_ID": converterModels.StringMetaValue("B"),
		},
		[]int64{9, 8, 7},
	)

	fmt.Printf("%+v\n", detectorsToString(lookup[555]))
	fmt.Printf("%+v\n", detectorsToString(lookup[123]))

	lookup = addToSpectraLookup(lookup,
		converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(123),
			"DETECTOR_ID": converterModels.StringMetaValue("A"),
		},
		[]int64{7, 8, 9},
	)

	fmt.Printf("%+v\n", detectorsToString(lookup[555]))
	fmt.Printf("%+v\n", detectorsToString(lookup[123]))

	// Output:
	// [0]meta [DETECTOR_ID:A/s PMC:123/i] spectrum [1 2 3]
	// [0]meta [DETECTOR_ID:A/s PMC:123/i] spectrum [1 2 3] [1]meta [DETECTOR_ID:B/s PMC:123/i] spectrum [4 5 6]
	// [0]meta [DETECTOR_ID:B/s PMC:555/i] spectrum [9 8 7]
	// [0]meta [DETECTOR_ID:A/s PMC:123/i] spectrum [1 2 3] [1]meta [DETECTOR_ID:B/s PMC:123/i] spectrum [4 5 6]
	// [0]meta [DETECTOR_ID:B/s PMC:555/i] spectrum [9 8 7]
	// [0]meta [DETECTOR_ID:A/s PMC:123/i] spectrum [1 2 3] [1]meta [DETECTOR_ID:B/s PMC:123/i] spectrum [4 5 6] [2]meta [DETECTOR_ID:A/s PMC:123/i] spectrum [7 8 9]
}

func Example_makeBulkMaxSpectra() {
	spectrumLookup := converterModels.DetectorSampleByPMC{
		1: []converterModels.DetectorSample{
			{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(1), "DETECTOR_ID": converterModels.StringMetaValue("A"), "XPERCHAN": converterModels.FloatMetaValue(10.4), "OFFSET": converterModels.FloatMetaValue(4)},
				Spectrum: []int64{1, 10, 100},
			},
			{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(1), "DETECTOR_ID": converterModels.StringMetaValue("B"), "LIVETIME": converterModels.FloatMetaValue(9.5)},
				Spectrum: []int64{3, 4, 5},
			},
		},
		2: []converterModels.DetectorSample{
			{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(2), "DETECTOR_ID": converterModels.StringMetaValue("A"), "XPERCHAN": converterModels.FloatMetaValue(6.4), "OFFSET": converterModels.FloatMetaValue(-6), "LIVETIME": converterModels.FloatMetaValue(8.5)},
				Spectrum: []int64{20, 30, 40},
			},
			{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(2), "DETECTOR_ID": converterModels.StringMetaValue("B"), "LIVETIME": converterModels.FloatMetaValue(10)},
				Spectrum: []int64{21, 22, 23},
			},
		},
	}

	pmc, data := makeBulkMaxSpectra(spectrumLookup, 0, 0, 40, 50)

	fmt.Printf("pmc=%d, len=%d\n", pmc, len(data))
	fmt.Printf("[0]=%+v\n", data[0].ToString())
	fmt.Printf("[1]=%+v\n", data[1].ToString())
	fmt.Printf("[2]=%+v\n", data[2].ToString())
	fmt.Printf("[3]=%+v\n", data[3].ToString())

	// Output:
	// pmc=3, len=4
	// [0]=meta [DETECTOR_ID:A/s LIVETIME:8.5/f OFFSET:-1/f PMC:3/i READTYPE:BulkSum/s SOURCEFILE:GeneratedByPIXLISEConverter/s XPERCHAN:8.4/f] spectrum [21 40 140]
	// [1]=meta [DETECTOR_ID:B/s LIVETIME:19.5/f OFFSET:50/f PMC:3/i READTYPE:BulkSum/s SOURCEFILE:GeneratedByPIXLISEConverter/s XPERCHAN:40/f] spectrum [24 26 28]
	// [2]=meta [DETECTOR_ID:A/s LIVETIME:8.5/f OFFSET:-1/f PMC:3/i READTYPE:MaxValue/s SOURCEFILE:GeneratedByPIXLISEConverter/s XPERCHAN:8.4/f] spectrum [20 30 100]
	// [3]=meta [DETECTOR_ID:B/s LIVETIME:9.75/f OFFSET:50/f PMC:3/i READTYPE:MaxValue/s SOURCEFILE:GeneratedByPIXLISEConverter/s XPERCHAN:40/f] spectrum [21 22 23]
}

func Example_eVCalibrationOverride() {
	spectrumLookup := converterModels.DetectorSampleByPMC{
		1: []converterModels.DetectorSample{
			converterModels.DetectorSample{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(1), "DETECTOR_ID": converterModels.StringMetaValue("A"), "XPERCHAN": converterModels.FloatMetaValue(10.4), "OFFSET": converterModels.FloatMetaValue(4)},
				Spectrum: []int64{1, 10, 100},
			},
			converterModels.DetectorSample{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(1), "DETECTOR_ID": converterModels.StringMetaValue("B")},
				Spectrum: []int64{3, 4, 5},
			},
		},
		2: []converterModels.DetectorSample{
			converterModels.DetectorSample{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(2), "DETECTOR_ID": converterModels.StringMetaValue("A"), "XPERCHAN": converterModels.FloatMetaValue(6.4), "OFFSET": converterModels.FloatMetaValue(-6)},
				Spectrum: []int64{20, 30, 40},
			},
			converterModels.DetectorSample{
				Meta:     converterModels.MetaData{"PMC": converterModels.IntMetaValue(2), "DETECTOR_ID": converterModels.StringMetaValue("B")},
				Spectrum: []int64{21, 22, 23},
			},
		},
	}

	err := eVCalibrationOverride(&spectrumLookup, 0, 0, 40, 50)

	fmt.Printf("err=%v, pmcs=%v, detector counts=%v,%v\n", err, len(spectrumLookup), len(spectrumLookup[1]), len(spectrumLookup[2]))

	for pmc := int32(1); pmc <= 2; pmc++ {
		for detIdx := 0; detIdx <= 1; detIdx++ {
			spec := spectrumLookup[pmc]
			//for pmc, spec := range spectrumLookup {
			//	for detIdx := range spec {
			fmt.Printf("pmc[%v][%v].Meta=%v, Spectrum=%v\n", pmc, detIdx, spec[detIdx].Meta.ToString(), spec[detIdx].Spectrum)
		}
	}

	// Output:
	// err=<nil>, pmcs=2, detector counts=2,2
	// pmc[1][0].Meta=[DETECTOR_ID:A/s OFFSET:4/f PMC:1/i XPERCHAN:10.4/f], Spectrum=[1 10 100]
	// pmc[1][1].Meta=[DETECTOR_ID:B/s OFFSET:50/f PMC:1/i XPERCHAN:40/f], Spectrum=[3 4 5]
	// pmc[2][0].Meta=[DETECTOR_ID:A/s OFFSET:-6/f PMC:2/i XPERCHAN:6.4/f], Spectrum=[20 30 40]
	// pmc[2][1].Meta=[DETECTOR_ID:B/s OFFSET:50/f PMC:2/i XPERCHAN:40/f], Spectrum=[21 22 23]
}

func Example_getSpectraReadType() {
	names := []string{
		"Normal_A_0612673072_000001C5_000013.msa",
		"Dwell_A_0612673072_000001C5_000013.msa",
		"MaxValue_A_0612729542_000001C5_004040.msa",
		"BulkSum_B_0612729540_000001C5_004040.msa",
		"File1.msa",
		"File.txt",
		"Average_B_0612729540_000001C5_004040.msa",
	}

	for _, name := range names {
		s, e := getSpectraReadType(name)
		fmt.Printf("%v|%v\n", s, e)
	}

	// Output:
	// Normal|<nil>
	// Dwell|<nil>
	// MaxValue|<nil>
	// BulkSum|<nil>
	// |unexpected MSA filename when detecting read type
	// |unexpected MSA filename when detecting read type
	// |unexpected MSA read type
}
