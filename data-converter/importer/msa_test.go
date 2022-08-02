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

	"github.com/pixlise/core/data-converter/converterModels"
)

func Example_ReadMSAFileLines() {
	data := []string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: Y", "#NCOLUMNS: 2", "#NPOINTS : 3", "#SPECTRUM", "0", "23", "991231"}
	items, err := ReadMSAFileLines(data, false, true, false)
	fmt.Println(err)

	data = []string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: YY", "#NCOLUMNS: 1", "#NPOINTS : 3", "#SPECTRUM", "0", "23", "991231"}
	items, err = ReadMSAFileLines(data, false, true, false)
	fmt.Println(err)

	data = []string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: YY", "#NCOLUMNS: 2", "#DETECTOR_ID: A", "#NPOINTS : 3", "#SPECTRUM", "0, 0", "23, 0", "48, 991231"}
	items, err = ReadMSAFileLines(data, false, true, false)
	fmt.Println(err)

	data = []string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: YY", "#NCOLUMNS: 2", "#NPOINTS : 3", "#SPECTRUM", "0, 0", "23, 0", "48, 991231"}
	items, err = ReadMSAFileLines(data, false, true, false)
	fmt.Println(err)

	fmt.Println("A")
	fmt.Printf(" %v\n", items[0].ToString())

	fmt.Println("B")
	fmt.Printf(" %v\n", items[1].ToString())

	// Output:
	// Expected DATATYPE "YY" in MSA metadata
	// Expected NCOLUMNS "2" in MSA metadata
	// Unexpected DETECTOR_ID in multi-detector MSA
	// <nil>
	// A
	//  meta [DATATYPE:YY/s DETECTOR_ID:A/s NCOLUMNS:2/s NPOINTS:3/s PMC:3001/i SOMETHING:123/s] spectrum [0 23 48]
	// B
	//  meta [DATATYPE:YY/s DETECTOR_ID:B/s NCOLUMNS:2/s NPOINTS:3/s PMC:3001/i SOMETHING:123/s] spectrum [0 0 991231]
}

func Example_ReadMSAFileLines_Single() {
	data := []string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: A", "#NPOINTS : 3", "#SPECTRUM", "0", "23", "991231"}
	items, err := ReadMSAFileLines(data, true, true, false)
	fmt.Printf("A|%v|%v\n", items[0].ToString(), err)

	data = []string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#NPOINTS : 5", "#SPECTRUM", "0", "23", "991231", "0", "44", "#ENDOFDATA here"}
	items, err = ReadMSAFileLines(data, true, true, false)
	fmt.Printf("B|%v|%v\n", items[0].ToString(), err)

	data = []string{"#SOMETHING:123", "#PMC: 3001", "#COMMENT: one", "#COMMENT: two", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#NPOINTS : 5", "#SPECTRUM", "0", "23", "991231", "0", "44", "#ENDOFDATA here"}
	items, err = ReadMSAFileLines(data, true, true, false)
	fmt.Printf("C|%v|%v\n", items[0].ToString(), err)

	// Duplicate non-comment field
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: YY", "#NCOLUMNS: 1", "#DATATYPE: YY", "#DETECTOR_ID: A", "#NPOINTS : 3", "#SPECTRUM", "0", "23", "991231"}, true, true, false)
	fmt.Printf("Dup|%v\n", err)

	// Wrong DATATYPE
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: YY", "#NCOLUMNS: 1", "#DETECTOR_ID: A", "#NPOINTS : 3", "#SPECTRUM", "0", "23", "991231"}, true, true, false)
	fmt.Printf("WrongDT|%v\n", err)

	// Not expecting PMC
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#NPOINTS : 5", "#SPECTRUM", "0", "23", "991231", "0", "44", "#ENDOFDATA here"}, true, false, false)
	fmt.Printf("NoExpPMC|%v\n", err)

	// Wrong point count
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#NPOINTS : 4", "#SPECTRUM", "0", "23", "991231"}, true, true, false)
	fmt.Printf("Wrong#Pts|%v\n", err)

	// Missing SPECTRUM
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#NPOINTS : 3", "99", "23", "991231"}, true, true, false)
	fmt.Printf("MissingSPECTRUM|%v\n", err)

	// Missing PMC
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#NPOINTS : 5", "#SPECTRUM", "0", "23", "991231", "0", "44", "#ENDOFDATA here"}, true, true, false)
	fmt.Printf("MissingPMC|%v\n", err)

	// Missing DETECTOR_ID
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: Y", "#NCOLUMNS: 1", "#NPOINTS : 5", "#SPECTRUM", "0", "23", "991231", "0", "44", "#ENDOFDATA here"}, true, true, false)
	fmt.Printf("MissingDETECTOR_ID|%v\n", err)

	// Missing NPOINTS
	items, err = ReadMSAFileLines([]string{"#SOMETHING:123", "#PMC: 3001", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#SPECTRUM", "0", "23", "991231", "0", "44", "#ENDOFDATA here"}, true, true, false)
	fmt.Printf("MissingNPOINTS|%v\n", err)

	// No metadata
	items, err = ReadMSAFileLines([]string{"50", "23", "991231"}, true, true, false)
	fmt.Printf("NoMeta|%v\n", err)

	// Data after end of data is ignored
	data = []string{"#SOMETHING:123", "#PMC: 3001", "#COMMENT: one", "#COMMENT: two", "#DATATYPE: Y", "#NCOLUMNS: 1", "#DETECTOR_ID: B", "#NPOINTS : 5", "#SPECTRUM", "0", "23", "991231", "0", "44", "#ENDOFDATA here", "78", "#SOME COMMENT!"}
	items, err = ReadMSAFileLines(data, true, true, false)
	fmt.Printf("D|%v|%v\n", items[0].ToString(), err)

	// Blank line
	items, err = ReadMSAFileLines([]string{""}, true, true, false)
	fmt.Printf("Blank|%v\n", err)

	// Empty file
	items, err = ReadMSAFileLines([]string{}, true, true, false)
	fmt.Printf("Empty|%v\n", err)

	// Output:
	// A|meta [DATATYPE:Y/s DETECTOR_ID:A/s NCOLUMNS:1/s NPOINTS:3/s PMC:3001/i SOMETHING:123/s] spectrum [0 23 991231]|<nil>
	// B|meta [DATATYPE:Y/s DETECTOR_ID:B/s NCOLUMNS:1/s NPOINTS:5/s PMC:3001/i SOMETHING:123/s] spectrum [0 23 991231 0 44]|<nil>
	// C|meta [COMMENT:one two/s DATATYPE:Y/s DETECTOR_ID:B/s NCOLUMNS:1/s NPOINTS:5/s PMC:3001/i SOMETHING:123/s] spectrum [0 23 991231 0 44]|<nil>
	// Dup|Duplicate meta data lines found for: DATATYPE
	// WrongDT|Expected DATATYPE "Y" in MSA metadata
	// NoExpPMC|PMC NOT expected, but was found in MSA
	// Wrong#Pts|Expected 4 spectra, got 3
	// MissingSPECTRUM|Unexpected potential spectra found at 5: 99
	// MissingPMC|PMC expected, but not found in MSA
	// MissingDETECTOR_ID|Failed to find DETECTOR_ID in metadata
	// MissingNPOINTS|Failed to find NPOINTS in metadata
	// NoMeta|Unexpected potential spectra found at 0: 50
	// D|meta [COMMENT:one two/s DATATYPE:Y/s DETECTOR_ID:B/s NCOLUMNS:1/s NPOINTS:5/s PMC:3001/i SOMETHING:123/s] spectrum [0 23 991231 0 44]|<nil>
	// Blank|No spectra data found to be read
	// Empty|No spectra data found to be read
}

func Example_splitMSAMetaFor2Detectors() {
	meta := converterModels.MetaData{
		"COMMENT":    converterModels.StringMetaValue("My Comment"),
		"XPERCHAN":   converterModels.StringMetaValue("  10.30, 11.30 "),
		"OFFSET":     converterModels.StringMetaValue("  3.30,   5.30 "),
		"SIGNALTYPE": converterModels.StringMetaValue("  XRF"),
		"DATATYPE":   converterModels.StringMetaValue("YY"),
		"PMC":        converterModels.IntMetaValue(99),
		"SCLK":       converterModels.IntMetaValue(399),
		"XPOSITION":  converterModels.StringMetaValue("    1.0030"),
		"YPOSITION":  converterModels.FloatMetaValue(2.0040),
		"ZPOSITION":  converterModels.FloatMetaValue(2.4430),
		"LIVETIME":   converterModels.StringMetaValue("  25.090,  25.080"),
		"REALTIME":   converterModels.StringMetaValue("  25.110,  25.120"),
		"TRIGGERS":   converterModels.StringMetaValue(" 45993, 43902"),
		"EVENTS":     converterModels.StringMetaValue(" 44690, 42823"),
		"KETEK_ICR":  converterModels.StringMetaValue(" 1833.1, 1750.7"),
		"KETEK_OCR":  converterModels.StringMetaValue(" 1780.1, 1705.7"),
		"DATE":       converterModels.StringMetaValue("03-20-2018"),
		"TIME":       converterModels.StringMetaValue("13:10:30"),
		"NPOINTS":    converterModels.StringMetaValue("4096"),
		"NCOLUMNS":   converterModels.StringMetaValue("2"),
		"XUNITS":     converterModels.StringMetaValue("eV"),
		"YUNITS":     converterModels.StringMetaValue("COUNTS"),
	}

	a, b, e := splitMSAMetaFor2Detectors(meta, false)
	fmt.Printf("%v\n", e)

	fmt.Println("META A")
	fmt.Printf("%v\n", a.ToString())

	fmt.Println("META B")
	fmt.Printf("%v\n", b.ToString())

	meta = converterModels.MetaData{
		"COMMENT":  converterModels.StringMetaValue("My comment"),
		"LIVETIME": converterModels.StringMetaValue("  25.09,  25.08, 30"),
	}
	a, b, e = splitMSAMetaFor2Detectors(meta, false)
	fmt.Printf("%v\n", e)

	// Output:
	// <nil>
	// META A
	// [COMMENT:My Comment/s DATATYPE:YY/s DATE:03-20-2018/s DETECTOR_ID:A/s EVENTS:44690/s KETEK_ICR:1833.1/s KETEK_OCR:1780.1/s LIVETIME:25.09/f NCOLUMNS:2/s NPOINTS:4096/s OFFSET:3.3/f PMC:99/i REALTIME:25.11/f SCLK:399/i SIGNALTYPE:XRF/s TIME:13:10:30/s TRIGGERS:45993/s XPERCHAN:10.3/f XPOSITION:1.0030/s XUNITS:eV/s YPOSITION:2.004/f YUNITS:COUNTS/s ZPOSITION:2.443/f]
	// META B
	// [COMMENT:My Comment/s DATATYPE:YY/s DATE:03-20-2018/s DETECTOR_ID:B/s EVENTS:42823/s KETEK_ICR:1750.7/s KETEK_OCR:1705.7/s LIVETIME:25.08/f NCOLUMNS:2/s NPOINTS:4096/s OFFSET:5.3/f PMC:99/i REALTIME:25.12/f SCLK:399/i SIGNALTYPE:XRF/s TIME:13:10:30/s TRIGGERS:43902/s XPERCHAN:11.3/f XPOSITION:1.0030/s XUNITS:eV/s YPOSITION:2.004/f YUNITS:COUNTS/s ZPOSITION:2.443/f]
	// Metadata row cannot be split for 2 detectors due to commas
}

func Example_parseMSAMetadataLine() {
	lines := []string{
		"#LIVETIME    :  25.09,  25.08",
		"#OFFSET      :  0.3,   0.1    eV of first channel",
		"#XPERCHAN    :  10.0, 10.0    eV per channel",
		"#NCOLUMNS    : 2     Number of data columns",
		"123",
		"Some:Thing",
		"#SOME TEXT HERE",
		"#FIELD:1234",
		"##THE FIELD:12.34",
		"#ANOTHER FIELD  :  999",
		"#NCOLUMNS    : 2 ",
		"#DATE        :       Date in the format DD-MMM-YYYY, for example 07-JUL-2010",
		"#LIVETIME    :   9.87332058 ",
		"#XPERCHAN    : 7.9226, 7.9273   eV per channel",
	}

	for _, line := range lines {
		k, v, err := parseMSAMetadataLine(line)
		fmt.Printf("%v|%v|%v\n", k, v, err)
	}

	// Output:
	// LIVETIME|25.09, 25.08|<nil>
	// OFFSET|0.3, 0.1|<nil>
	// XPERCHAN|10.0, 10.0|<nil>
	// NCOLUMNS|2|<nil>
	// ||Expected # at start of metadata: 123
	// ||Expected # at start of metadata: Some:Thing
	// ||Failed to parse metadata line: #SOME TEXT HERE
	// FIELD|1234|<nil>
	// THE FIELD|12.34|<nil>
	// ANOTHER FIELD|999|<nil>
	// NCOLUMNS|2|<nil>
	// DATE||<nil>
	// LIVETIME|9.87332058|<nil>
	// XPERCHAN|7.9226, 7.9273|<nil>
}

type parseMSASpectraLineTestItem struct {
	line  string
	lc    int
	ncols int
}

func Example_parseMSASpectraLine() {

	testData := []parseMSASpectraLineTestItem{
		// 1 column
		parseMSASpectraLineTestItem{"1983", 7, 1},
		parseMSASpectraLineTestItem{"1", 8, 1},
		parseMSASpectraLineTestItem{"0", 9, 1},

		// 2 columns
		parseMSASpectraLineTestItem{"1983, 44", 7, 2},
		parseMSASpectraLineTestItem{"1, 0", 8, 2},
		parseMSASpectraLineTestItem{"2321,32342", 9, 2},

		// 3 columns (it doesn"t care)
		parseMSASpectraLineTestItem{"11, 22, 33", 9, 3},

		// 0 columns (sanity)
		parseMSASpectraLineTestItem{"11, 22, 33", 9, 0},

		// Wrong column counts
		parseMSASpectraLineTestItem{"1983, 44", 7, 1},
		parseMSASpectraLineTestItem{"1983", 7, 2},
		parseMSASpectraLineTestItem{"1983,", 7, 2},
		parseMSASpectraLineTestItem{"", 7, 1},
		parseMSASpectraLineTestItem{"", 7, 2},
		parseMSASpectraLineTestItem{",", 7, 1},
		parseMSASpectraLineTestItem{",", 7, 2},

		// Issues with parsing values
		parseMSASpectraLineTestItem{"#SOMETHING", 1, 1},
		parseMSASpectraLineTestItem{"#SOMETHING,#ELSE", 1, 2},
		parseMSASpectraLineTestItem{"1,#Number", 1, 2},
		parseMSASpectraLineTestItem{"Waffles", 2, 1},
		parseMSASpectraLineTestItem{"1.6", 4, 1},
		parseMSASpectraLineTestItem{"1.6, 3.1415926", 4, 2},
		parseMSASpectraLineTestItem{"16,3.1415926", 4, 2},
		parseMSASpectraLineTestItem{"-34, 10", 6, 2},
		parseMSASpectraLineTestItem{"34, -10", 6, 2},
		parseMSASpectraLineTestItem{"5, Waffles", 6, 2},
		parseMSASpectraLineTestItem{"Waffles, 5", 6, 2},
	}

	for _, t := range testData {
		v, e := parseMSASpectraLine(t.line, t.lc, t.ncols)
		fmt.Printf("%v|%v\n", v, e)
	}

	// Output:
	// [1983]|<nil>
	// [1]|<nil>
	// [0]|<nil>
	// [1983 44]|<nil>
	// [1 0]|<nil>
	// [2321 32342]|<nil>
	// [11 22 33]|<nil>
	// []|Expected 0 spectrum columns, got 3 on line [9]:11, 22, 33
	// []|Expected 1 spectrum columns, got 2 on line [7]:1983, 44
	// []|Expected 2 spectrum columns, got 1 on line [7]:1983
	// []|Failed to read spectra "" on line [7]:1983,
	// []|Failed to read spectra "" on line [7]:
	// []|Expected 2 spectrum columns, got 1 on line [7]:
	// []|Expected 1 spectrum columns, got 2 on line [7]:,
	// []|Failed to read spectra "" on line [7]:,
	// []|Failed to read spectra "#SOMETHING" on line [1]:#SOMETHING
	// []|Failed to read spectra "#SOMETHING" on line [1]:#SOMETHING,#ELSE
	// []|Failed to read spectra "#Number" on line [1]:1,#Number
	// []|Failed to read spectra "Waffles" on line [2]:Waffles
	// []|Failed to read spectra "1.6" on line [4]:1.6
	// []|Failed to read spectra "1.6" on line [4]:1.6, 3.1415926
	// []|Failed to read spectra "3.1415926" on line [4]:16,3.1415926
	// []|Spectra expected non-negative value "-34" on line [6]:-34, 10
	// []|Spectra expected non-negative value "-10" on line [6]:34, -10
	// []|Failed to read spectra "Waffles" on line [6]:5, Waffles
	// []|Failed to read spectra "Waffles" on line [6]:Waffles, 5
}
