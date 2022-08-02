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

package main

import (
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"strings"

	"github.com/pixlise/core/core/quantModel"
)

func makeExpectedMetaList(allMetaColumns []string, userExpectedMissingColumns []string) ([]string, error) {
	// User can specify columns which are known to be missing, so we have to build a list of meta columns we DO expect to be here...
	expectMissingColumnLookup := map[string]bool{}
	for _, expCol := range userExpectedMissingColumns {
		if len(expCol) > 0 {
			expectMissingColumnLookup[expCol] = true
		}
	}

	// Run through all meta columns, if one is expected to NOT be there, don't add it to the list we expect...
	// Also make sure there is nothing unknown left int he expected missing col list
	remainingMetaColumns := []string{}
	for _, col := range allMetaColumns {
		if expectMissingColumnLookup[col] {
			// Also remove it from map, so we're left with only ones that didn't get checked
			delete(expectMissingColumnLookup, col)
		} else {
			remainingMetaColumns = append(remainingMetaColumns, col)
		}
	}

	// If anything is left in the lookup, stop here
	if len(expectMissingColumnLookup) > 0 {
		// What's the user doing specifying this one?
		return nil, fmt.Errorf("Unknown columns set as expected to be missing: %v", expectMissingColumnLookup)
	}

	return remainingMetaColumns, nil
}

func main() {
	// Parse command line args
	var matchPMCDatasetFileName string
	var matchPMCMode string
	var expectMissingColumnStr string
	var detectorIDOverride string
	var detectorDuplicateAB bool

	// Python used to allow a short and long name, so we're defining them both here so old shell scripts still work
	flag.StringVar(&matchPMCDatasetFileName, "p", "", "Specify dataset file to match PMCs with, will fail if PMC column exists")
	flag.StringVar(&matchPMCDatasetFileName, "match_pmcs", "", "Specify dataset file to match PMCs with, will fail if PMC column exists")

	flag.StringVar(&matchPMCMode, "match_pmc_mode", "coord", "Can be coord or filename, to match based on MSA's xyz coord or filename")

	flag.StringVar(&expectMissingColumnStr, "sc", "", "List of columns which are OK to be missing")
	flag.StringVar(&expectMissingColumnStr, "sub_empty_columns", "", "List of columns which are OK to be missing")

	flag.StringVar(&detectorIDOverride, "d", "", "Specify a detector (A, B or Combined), if not specified works it out from filename column")
	flag.StringVar(&detectorIDOverride, "detector", "", "Specify a detector (A, B or Combined), if not specified works it out from filename column")

	flag.BoolVar(&detectorDuplicateAB, "d-dup", false, "Make A and B the same")
	flag.BoolVar(&detectorDuplicateAB, "detector_duplicate", false, "Make A and B the same")

	flag.Parse()

	if flag.NArg() != 2 {
		log.Fatalln("Must specify CSV_FILE_NAME and OUTPUT_FILE_NAME arguments")
	}

	mapCSVFileName := flag.Arg(0)
	outFileName := flag.Arg(1)

	fmt.Println("Quantification CSV --> PIXLISE binary format converter")
	fmt.Println("======================================================")
	fmt.Printf(" Input CSV file: \"%v\", Output file: \"%v\"\n", mapCSVFileName, outFileName)

	if len(matchPMCDatasetFileName) > 0 {
		// Make sure the mode is valid
		if matchPMCMode != "coord" && matchPMCMode != "filename" {
			log.Fatalln("match_pmc_mode must be coord or filename")
		}
		fmt.Printf(" Match PMC with dataset: \"%v\", mode: \"%v\"\n", matchPMCDatasetFileName, matchPMCMode)
	}
	if len(detectorIDOverride) > 0 {
		// Make sure it's A or B
		if detectorIDOverride != "A" && detectorIDOverride != "B" && detectorIDOverride != "Combined" {
			log.Fatalln("detector must be A, B or Combined")
		}
		fmt.Printf(" Detector override: \"%v\"\n", detectorIDOverride)
	}
	txt := "ON"
	if !detectorDuplicateAB {
		txt = "OFF"
	}
	fmt.Println(" Detector AB duplication: " + txt)

	// ALL meta columns...
	metaColumns, err := makeExpectedMetaList([]string{"PMC", "SCLK", "RTT", "filename"}, strings.Split(expectMissingColumnStr, ","))
	if err != nil {
		log.Fatalln(err)
	}
	fmt.Printf(" Expecting meta columns to exist: %v\n", metaColumns)

	// If output file path doesn't exist, make it
	outPath := path.Dir(outFileName)
	os.MkdirAll(outPath, os.ModePerm)

	// Run it!
	log.Println("Reading Quantification CSV: " + mapCSVFileName)

	// Read the CSV
	data, err := ioutil.ReadFile(mapCSVFileName)
	if err != nil {
		log.Fatalf("Failed to read CSV %v. Error: %v", mapCSVFileName, err)
	}

	serialisedBytes, _, err := quantModel.ConvertQuantificationCSV("local", string(data), metaColumns, matchPMCDatasetFileName, matchPMCMode == "coord", detectorIDOverride, detectorDuplicateAB)
	if err != nil {
		log.Fatalf("Conversion error: %v", err)
	}

	if err := ioutil.WriteFile(outFileName, serialisedBytes, 0644); err != nil {
		log.Fatalf("Failed to write quantification protobuf: %v", err)
	}

	log.Printf("Quantification converted successfully, saved to: %v\n", outFileName)
}
