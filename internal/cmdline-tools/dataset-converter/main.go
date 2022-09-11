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

package main

import (
	"flag"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/pixlise/core/v2/core/logger"

	"github.com/pixlise/core/v2/data-converter/importer"
	"github.com/pixlise/core/v2/data-converter/importer/msatestdata"
	"github.com/pixlise/core/v2/data-converter/importer/pixlfm"
	"github.com/pixlise/core/v2/data-converter/output"
)

func main() {
	fmt.Println("==============================")
	fmt.Println("=  PIXLISE dataset importer  =")
	fmt.Println("==============================")

	var jobLog logger.ILogger = &logger.StdOutLogger{}

	importers := map[string]importer.Importer{"test-msa": msatestdata.MSATestData{}, "pixl-fm": pixlfm.PIXLFM{}}
	importerNames := []string{} // TODO: REFACTOR: Make this work instead importerNames := utils.GetStringMapKeys(importers)
	for k := range importers {
		importerNames = append(importerNames, k)
	}

	var argFormat = flag.String("format", "", "Input format, one of: "+strings.Join(importerNames, ","))
	var argInPath = flag.String("inpath", "", "Path to directory containing input dataset")
	var argRangesPath = flag.String("rangespath", "", "Path to pseudo-intensity range CSV, only required if dataset contains pseudo-intensity data")
	var argOutPath = flag.String("outpath", "", "Output path")
	var argOutDirPrefix = flag.String("outdirprefix", "", "Output directory prefix")
	var argOutTitleOverride = flag.String("outtitle", "", "Output title override")
	var argDetectorOverride = flag.String("outdetector", "", "Output detector config override")
	var argGroupOverride = flag.String("outgroup", "", "Output dataset group override")

	flag.Parse()

	importer, ok := importers[*argFormat]
	if !ok {
		jobLog.Infof("Importer for format \"%v\" not found\n", *argFormat)
		printFail()
		return
	}

	jobLog.Infof("----- Importing %v dataset: %v -----\n", *argFormat, *argInPath)

	data, contextImageSrcPath, err := importer.Import(*argInPath, *argRangesPath, jobLog)
	if err != nil {
		jobLog.Infof("IMPORT ERROR: %v\n", err)
		printFail()
		return
	}

	// Override dataset ID for output if required
	if argOutTitleOverride != nil && len(*argOutTitleOverride) > 0 {
		data.Meta.Title = *argOutTitleOverride
	}

	// Override detector config if required
	if argDetectorOverride != nil && len(*argDetectorOverride) > 0 {
		data.DetectorConfig = *argDetectorOverride
	}

	// Override dataset group if required
	if argGroupOverride != nil && len(*argGroupOverride) > 0 {
		data.Group = *argGroupOverride
	}

	// Form the output path
	outPath := path.Join(*argOutPath, *argOutDirPrefix+data.DatasetID)

	jobLog.Infof("----- Writing Dataset to: %v -----\n", outPath)
	saver := output.PIXLISEDataSaver{}
	err = saver.Save(*data, contextImageSrcPath, outPath, time.Now().Unix(), jobLog)
	if err != nil {
		jobLog.Infof("WRITE ERROR: %v\n", err)
		printFail()
		return
	}

	jobLog.Infof("\n--------  SUCCESS  --------\n\n\n")
}

func printFail() {
	fmt.Printf("\n****************************\n")
	fmt.Printf("**  FAIL    FAIL    FAIL  **\n")
	fmt.Printf("****************************\n\n\n")
}
