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
	"path"
	"strings"
	"time"

	"github.com/pixlise/core/core/awsutil"
	"github.com/pixlise/core/core/logger"

	"github.com/pixlise/core/data-converter/importer"
	"github.com/pixlise/core/data-converter/importer/msatestdata"
	"github.com/pixlise/core/data-converter/importer/pixlfm"
	"github.com/pixlise/core/data-converter/output"
)

func main() {
	fmt.Println("==============================")
	fmt.Println("=  PIXLISE dataset importer  =")
	fmt.Println("==============================")

	var jobLog logger.ILogger
	makeLog := true
	if !makeLog {
		// Creator doesn't want it logged - used for unit tests so we don't have to set up AWS credentials
		jobLog = logger.NullLogger{}
	} else {
		var err error
		var loglevel = logger.LogDebug
		sess, _ := awsutil.GetSession()
		jobLog, err = logger.Init("dataimport-manual", loglevel, "prod", sess)
		if err != nil {
			fmt.Printf("WARNING: Failed to create log group stream. Logging to stdout. Error was: \"%v\"\n", err)
			jobLog = logger.StdOutLogger{}
		}
	}

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
