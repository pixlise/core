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
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"gitlab.com/pixlise/pixlise-go-api/core/logger"
	"gitlab.com/pixlise/pixlise-go-api/core/utils"
	"gitlab.com/pixlise/pixlise-go-api/data-converter/converterModels"

	"gitlab.com/pixlise/pixlise-go-api/data-converter/importer"
)

func listMSAFilesToProcess(path string, ignoreMSAFiles string, jobLog logger.ILogger) ([]string, error) {
	allMSAFiles, err := importer.GetDirListing(path, "", jobLog)

	if err != nil {
		return nil, err
	}

	if ignoreMSAFiles != "" {
		var splitmsas = strings.Split(ignoreMSAFiles, ",")

		for _, ignoreMSA := range splitmsas {
			for i, f := range allMSAFiles {
				if strings.HasSuffix(f, ignoreMSA) {
					copy(allMSAFiles[i:], allMSAFiles[i+1:])
					allMSAFiles[len(allMSAFiles)-1] = ""
					allMSAFiles = allMSAFiles[:len(allMSAFiles)-1]
				}
			}
		}
	}

	return allMSAFiles, nil
}

func getSpectraFiles(allFiles []string, verifyReadType bool) ([]string, []string) {
	var toRead []string
	var logs []string

	for _, f := range allFiles {
		ext := filepath.Ext(f)
		_, file := filepath.Split(f)

		if ext != ".msa" {
			logs = append(logs, "Ignoring extension '"+ext+"' in: "+f)
		} else {
			if verifyReadType {
				_, err := getSpectraReadType(file)
				if err != nil {
					logs = append(logs, err.Error())
				} else {
					toRead = append(toRead, f)
				}
			} else {
				toRead = append(toRead, f)
			}
		}
	}

	count := len(toRead)
	sort.SliceStable(toRead, func(i, j int) bool {
		iPMC, err1 := getMSASeqNo(toRead[i])
		jPMC, err2 := getMSASeqNo(toRead[j])
		if err1 != nil {
			fmt.Printf("ERROR when sorting in getSpectraFiles, filename: %v\n", toRead[i])
			iPMC = 0
		}
		if err2 != nil {
			fmt.Printf("ERROR when sorting in getSpectraFiles, filename: %v\n", toRead[j])
			jPMC = 0
		}
		return iPMC < jPMC
	})

	if len(toRead) != count {
		panic("COUNT MISMATCH when sorting spectra file names")
	}

	return toRead, logs
}

func getMSASeqNo(path string) (int64, error) {
	ext := filepath.Ext(path)

	if strings.ToUpper(ext) != ".MSA" {
		return 0, errors.New("Unexpected file extension when reading MSA: " + ext)
	}

	filenamebits := strings.Split(path, "_")
	if len(filenamebits) < 2 {
		return 0, errors.New("Invalid MSA file name: " + path)
	}

	seqNoStr := filenamebits[len(filenamebits)-1]
	seqNoStr = seqNoStr[0 : len(seqNoStr)-len(ext)]
	seqNo, err := strconv.Atoi(seqNoStr)
	if err != nil {
		return 0, err
	}

	return int64(seqNo), nil
}

func makeSpectraLookup(inputpath string, spectraFiles []string, singleDetectorMSAs bool, genPMCs bool, readTypeOverride string, detectorADuplicate bool, jobLog logger.ILogger) (converterModels.DetectorSampleByPMC, error) {
	spectraLookup := make(converterModels.DetectorSampleByPMC)

	c := 1
	for _, f := range spectraFiles {
		path := path.Join(inputpath, f)
		lines, err := importer.ReadFileLines(path, jobLog)
		if err != nil {
			return spectraLookup, fmt.Errorf("Error in %v: %v", path, err)
		}

		// Make this overwrite last line, don't want to print 1000s of lines
		ending := "\n"
		if c < len(spectraFiles) {
			ending = "\r"
		}
		fmt.Printf("  Reading spectrum [%v/%v] from: %v       %v", c, len(spectraFiles), f, ending)

		spectrumList, err := importer.ReadMSAFileLines(lines, singleDetectorMSAs, !genPMCs, detectorADuplicate)
		if err != nil {
			return spectraLookup, fmt.Errorf("Error in %v: %v", path, err)
		}

		for _, s := range spectrumList {
			if genPMCs {
				s.Meta["PMC"] = converterModels.IntMetaValue(int32(c))
			}

			if _, ok := s.Meta["SOURCEFILE"]; ok {
				return spectraLookup, fmt.Errorf("Unexpected SOURCEFILE metadata already defined in %v", path)
			}

			s.Meta["SOURCEFILE"] = converterModels.StringMetaValue(f)

			// Use the override if it's provided
			rt := readTypeOverride
			if rt == "" {
				rt, err = getSpectraReadType(f)
				if err != nil {
					return spectraLookup, fmt.Errorf("Error in %v: %v", path, err)
				}
			}

			s.Meta["READTYPE"] = converterModels.StringMetaValue(rt)

			spectraLookup = addToSpectraLookup(spectraLookup, s.Meta, s.Spectrum)
		}
		c = c + 1
	}

	return spectraLookup, nil
}

func eVCalibrationOverride(spectraLookup *converterModels.DetectorSampleByPMC, xperchanA float32, offsetA float32, xperchanB float32, offsetB float32) error {
	// Overrides eV calibration metadata in each spectrum (XPERCHAN, OFFSET)
	for pmc, detSamples := range *spectraLookup {
		for detIdx := range detSamples {
			det, ok := (*spectraLookup)[pmc][detIdx].Meta["DETECTOR_ID"]
			if !ok {
				return fmt.Errorf("Failed to determine detector ID for PMC: %v", pmc)
			}

			if det.SValue == "A" {
				if xperchanA != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["XPERCHAN"] = converterModels.FloatMetaValue(xperchanA)
				}
				if offsetA != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["OFFSET"] = converterModels.FloatMetaValue(offsetA)
				}
			} else if det.SValue == "B" {
				if xperchanB != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["XPERCHAN"] = converterModels.FloatMetaValue(xperchanB)
				}
				if offsetB != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["OFFSET"] = converterModels.FloatMetaValue(offsetB)
				}
			} else {
				return fmt.Errorf("Invalid detector ID \"%v\" for PMC: %v", det.SValue, pmc)
			}
		}
	}
	return nil
}

func getSpectraReadType(filename string) (string, error) {
	_, f := filepath.Split(filename)
	bits := strings.Split(f, "_")

	if len(bits) != 5 {
		return "", errors.New("unexpected MSA filename when detecting read type")
	}

	readType := bits[0]

	expTypes := []string{"BulkSum", "MaxValue", "Normal", "Dwell"}

	if !utils.StringInSlice(readType, expTypes) {
		return "", errors.New("unexpected MSA read type")
	}

	return readType, nil
}

func addToSpectraLookup(spectralookup converterModels.DetectorSampleByPMC, metaFromFile converterModels.MetaData, spectraFromFile []int64) converterModels.DetectorSampleByPMC {
	pmc := metaFromFile["PMC"].IValue

	if _, ok := spectralookup[pmc]; ok {
		s := converterModels.DetectorSample{
			Meta:     metaFromFile,
			Spectrum: spectraFromFile,
		}
		spectralookup[pmc] = append(spectralookup[pmc], s)
	} else {
		spectralookup[pmc] = nil
		s := converterModels.DetectorSample{
			Meta:     metaFromFile,
			Spectrum: spectraFromFile,
		}
		spectralookup[pmc] = append(spectralookup[pmc], s)
	}

	return spectralookup
}

func makeBulkMaxSpectra(spectraLookup converterModels.DetectorSampleByPMC, xperchanA float32, offsetA float32, xperchanB float32, offsetB float32) (int32, []converterModels.DetectorSample) {
	specialPMC := int32(len(spectraLookup) + 1)

	generated := map[string]converterModels.DetectorSample{}

	AXPERCHAN := float32(0)
	AXCount := 0
	AOFFSET := float32(0)
	AOffsetCount := 0
	BXPERCHAN := float32(0)
	BXCount := 0
	BOFFSET := float32(0)
	BOffsetCount := 0

	generated["bulkA"] = converterModels.DetectorSample{
		Meta: converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": converterModels.StringMetaValue("A"),
			"READTYPE":    converterModels.StringMetaValue("BulkSum"),
			"SOURCEFILE":  converterModels.StringMetaValue("GeneratedByPIXLISEConverter"),
		},
		Spectrum: []int64{},
	}

	generated["bulkB"] = converterModels.DetectorSample{
		Meta: converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": converterModels.StringMetaValue("B"),
			"READTYPE":    converterModels.StringMetaValue("BulkSum"),
			"SOURCEFILE":  converterModels.StringMetaValue("GeneratedByPIXLISEConverter"),
		},
		Spectrum: []int64{},
	}

	generated["maxA"] = converterModels.DetectorSample{
		Meta: converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": converterModels.StringMetaValue("A"),
			"READTYPE":    converterModels.StringMetaValue("MaxValue"),
			"SOURCEFILE":  converterModels.StringMetaValue("GeneratedByPIXLISEConverter"),
		},
		Spectrum: []int64{},
	}

	generated["maxB"] = converterModels.DetectorSample{
		Meta: converterModels.MetaData{
			"PMC":         converterModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": converterModels.StringMetaValue("B"),
			"READTYPE":    converterModels.StringMetaValue("MaxValue"),
			"SOURCEFILE":  converterModels.StringMetaValue("GeneratedByPIXLISEConverter"),
		},
		Spectrum: []int64{},
	}

	liveTimeA := float32(0)
	liveTimeB := float32(0)
	liveTimeACount := 0
	liveTimeBCount := 0

	for _, data := range spectraLookup {
		for _, detectorData := range data {
			detectorID := detectorData.Meta["DETECTOR_ID"].SValue

			if detectorID == "A" {
				xperchan, ok := detectorData.Meta["XPERCHAN"]
				if ok {
					AXPERCHAN += xperchan.FValue
					AXCount++
				}
				offset, ok := detectorData.Meta["OFFSET"]
				if ok {
					AOFFSET += offset.FValue
					AOffsetCount++
				}
				liveTime, ok := detectorData.Meta["LIVETIME"]
				if ok {
					liveTimeA += liveTime.FValue
					liveTimeACount++
				}
			}
			if detectorID == "B" {
				xperchan, ok := detectorData.Meta["XPERCHAN"]
				if ok {
					BXPERCHAN += xperchan.FValue
					BXCount++
				}
				offset, ok := detectorData.Meta["OFFSET"]
				if ok {
					BOFFSET += offset.FValue
					BOffsetCount++
				}
				liveTime, ok := detectorData.Meta["LIVETIME"]
				if ok {
					liveTimeB += liveTime.FValue
					liveTimeBCount++
				}
			}

			bulkID := "bulk" + detectorID
			maxID := "max" + detectorID

			if len(generated[bulkID].Spectrum) <= 0 {
				l := len(detectorData.Spectrum)
				generated[bulkID] = converterModels.DetectorSample{
					Meta:     generated[bulkID].Meta,
					Spectrum: make([]int64, l),
				}
			}

			if len(generated[maxID].Spectrum) <= 0 {
				l := len(detectorData.Spectrum)
				generated[maxID] = converterModels.DetectorSample{
					Meta:     generated[maxID].Meta,
					Spectrum: make([]int64, l),
				}
			}

			for i := range detectorData.Spectrum {
				generated[bulkID].Spectrum[i] = generated[bulkID].Spectrum[i] + detectorData.Spectrum[i]
				if detectorData.Spectrum[i] > generated[maxID].Spectrum[i] {
					generated[maxID].Spectrum[i] = detectorData.Spectrum[i]
				}
			}
		}
	}

	// If we found any calibration values, save them in each
	if AXCount > 0 || xperchanA != 0 {
		x := converterModels.FloatMetaValue(xperchanA)
		if AXCount > 0 {
			x = converterModels.FloatMetaValue(AXPERCHAN / float32(AXCount))
		}
		generated["bulkA"].Meta["XPERCHAN"] = x
		generated["maxA"].Meta["XPERCHAN"] = x
	}

	if AOffsetCount > 0 || offsetA != 0 {
		x := converterModels.FloatMetaValue(offsetA)
		if AOffsetCount > 0 {
			x = converterModels.FloatMetaValue(AOFFSET / float32(AOffsetCount))
		}
		generated["bulkA"].Meta["OFFSET"] = x
		generated["maxA"].Meta["OFFSET"] = x
	}

	if BXCount > 0 || xperchanB != 0 {
		x := converterModels.FloatMetaValue(xperchanB)
		if BXCount > 0 {
			x = converterModels.FloatMetaValue(BXPERCHAN / float32(BXCount))
		}
		generated["bulkB"].Meta["XPERCHAN"] = x
		generated["maxB"].Meta["XPERCHAN"] = x
	}

	if BOffsetCount > 0 || offsetB != 0 {
		x := converterModels.FloatMetaValue(offsetB)
		if BOffsetCount > 0 {
			x = converterModels.FloatMetaValue(BOFFSET / float32(BOffsetCount))
		}
		generated["bulkB"].Meta["OFFSET"] = x
		generated["maxB"].Meta["OFFSET"] = x
	}

	// Set the live times for bulk spectra
	generated["bulkA"].Meta["LIVETIME"] = converterModels.FloatMetaValue(liveTimeA)
	generated["bulkB"].Meta["LIVETIME"] = converterModels.FloatMetaValue(liveTimeB)

	// Ensure no div by 0
	if liveTimeACount == 0 {
		liveTimeACount = 1
	}
	if liveTimeBCount == 0 {
		liveTimeBCount = 1
	}

	// Set average live time for max spectra
	generated["maxA"].Meta["LIVETIME"] = converterModels.FloatMetaValue(liveTimeA / float32(liveTimeACount))
	generated["maxB"].Meta["LIVETIME"] = converterModels.FloatMetaValue(liveTimeB / float32(liveTimeBCount))

	arr := []converterModels.DetectorSample{generated["bulkA"], generated["bulkB"], generated["maxA"], generated["maxB"]}
	return specialPMC, arr
}
