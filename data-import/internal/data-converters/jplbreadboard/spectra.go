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
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/core/utils"
	"github.com/pixlise/core/v2/data-import/internal/dataConvertModels"
	"github.com/pixlise/core/v2/data-import/internal/importerutils"
)

func getSpectraFiles(allFiles []string, verifyReadType bool, jobLog logger.ILogger) ([]string, []string) {
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

	// Get the sequence number from each file name and sort them
	// We call SliceStable here because we may have duplicates, where 2 sequence numbers match
	// for example if we have just 1 detector in an msa, and we have an A and B file
	// First, we scan through all the file names to make sure we filter out any where we can't find
	// a sequence number. This is because the breadboard exports a "notes" file (may not exist) and
	// we can't rely on its file name containing "notes" or something... but it ends in date usually.
	// Example:
	// Notes file: YL_DAKP_rock_28V_230uA_08_15_2022_notes_08-16-22-18-04-58.msa
	// MSA file:   YL_DAKP_rock_28V_230uA_08_15_2022_5625.msa

	toReadFiltered := make([]string, 0, len(toRead))
	for _, name := range toRead {
		iPMC, err := getMSASeqNo(name)
		if err == nil && iPMC > 0 {
			toReadFiltered = append(toReadFiltered, name)
		} else {
			jobLog.Infof("Warning: ignoring spectrum file, due to not finding sequence number: %v", name)
		}
	}

	count := len(toReadFiltered)
	sort.SliceStable(toReadFiltered, func(i, j int) bool {
		iPMC, err1 := getMSASeqNo(toReadFiltered[i])
		jPMC, err2 := getMSASeqNo(toReadFiltered[j])
		if err1 != nil {
			jobLog.Errorf("Failed to sort1 in getSpectraFiles, filename: %v", toReadFiltered[i])
			iPMC = 0
		}
		if err2 != nil {
			jobLog.Errorf("Failed to sort2 in getSpectraFiles, filename: %v", toReadFiltered[j])
			jPMC = 0
		}
		return iPMC < jPMC
	})

	if len(toReadFiltered) != count {
		jobLog.Errorf("COUNT MISMATCH when sorting spectra file names")
		toReadFiltered = []string{}
	}

	return toReadFiltered, logs
}

// Assumes file name ends in _00123.msa
// Returns 123
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

func makeSpectraLookup(inputpath string, spectraFiles []string, singleDetectorMSAs bool, genPMCs bool, readTypeOverride string, detectorADuplicate bool, jobLog logger.ILogger) (dataConvertModels.DetectorSampleByPMC, error) {
	spectraLookup := make(dataConvertModels.DetectorSampleByPMC)

	reportInterval := len(spectraFiles) / 10

	c := 1
	for idx, f := range spectraFiles {
		path := path.Join(inputpath, f)
		lines, err := utils.ReadFileLines(path)
		if err != nil {
			return spectraLookup, fmt.Errorf("Error in %v: %v", path, err)
		}

		/*
			// This was useful for showing on command line tool, but we don't want to spam cloudwatch logs with this stuff
			// Make this overwrite last line, don't want to print 1000s of lines
			ending := "\n"
			if c < len(spectraFiles) {
				ending = "\r"
			}

			//fmt.Printf("  Reading spectrum [%v/%v] from: %v       %v", c, len(spectraFiles), f, ending)
		*/

		// Rate limit progress reporting to logs
		if idx%reportInterval == 0 {
			jobLog.Infof("  Reading spectrum [%v/%v] %v%%", c, len(spectraFiles), 100*c/len(spectraFiles))
		}

		spectrumList, err := importerutils.ReadMSAFileLines(lines, singleDetectorMSAs, !genPMCs, detectorADuplicate)
		if err != nil {
			return spectraLookup, fmt.Errorf("Error in %v: %v", path, err)
		}

		for _, s := range spectrumList {
			if genPMCs {
				s.Meta["PMC"] = dataConvertModels.IntMetaValue(int32(c))
			}

			if _, ok := s.Meta["SOURCEFILE"]; ok {
				return spectraLookup, fmt.Errorf("Unexpected SOURCEFILE metadata already defined in %v", path)
			}

			s.Meta["SOURCEFILE"] = dataConvertModels.StringMetaValue(f)

			// Use the override if it's provided
			rt := readTypeOverride
			if rt == "" {
				rt, err = getSpectraReadType(f)
				if err != nil {
					return spectraLookup, fmt.Errorf("Error in %v: %v", path, err)
				}
			}

			s.Meta["READTYPE"] = dataConvertModels.StringMetaValue(rt)

			spectraLookup = addToSpectraLookup(spectraLookup, s.Meta, s.Spectrum)
		}
		c = c + 1
	}

	return spectraLookup, nil
}

func eVCalibrationOverride(spectraLookup *dataConvertModels.DetectorSampleByPMC, xperchanA float32, offsetA float32, xperchanB float32, offsetB float32) error {
	// Overrides eV calibration metadata in each spectrum (XPERCHAN, OFFSET)
	for pmc, detSamples := range *spectraLookup {
		for detIdx := range detSamples {
			det, ok := (*spectraLookup)[pmc][detIdx].Meta["DETECTOR_ID"]
			if !ok {
				return fmt.Errorf("Failed to determine detector ID for PMC: %v", pmc)
			}

			if det.SValue == "A" {
				if xperchanA != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["XPERCHAN"] = dataConvertModels.FloatMetaValue(xperchanA)
				}
				if offsetA != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["OFFSET"] = dataConvertModels.FloatMetaValue(offsetA)
				}
			} else if det.SValue == "B" {
				if xperchanB != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["XPERCHAN"] = dataConvertModels.FloatMetaValue(xperchanB)
				}
				if offsetB != 0 {
					(*spectraLookup)[pmc][detIdx].Meta["OFFSET"] = dataConvertModels.FloatMetaValue(offsetB)
				}
			} else {
				return fmt.Errorf("Invalid detector ID \"%v\" for PMC: %v", det.SValue, pmc)
			}
		}
	}
	return nil
}

// Assumes a file name like: Normal_A_0612673072_000001C5_000013.msa
// Returns Normal from the above example
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

func addToSpectraLookup(spectralookup dataConvertModels.DetectorSampleByPMC, metaFromFile dataConvertModels.MetaData, spectraFromFile []int64) dataConvertModels.DetectorSampleByPMC {
	pmc := metaFromFile["PMC"].IValue

	if _, ok := spectralookup[pmc]; ok {
		s := dataConvertModels.DetectorSample{
			Meta:     metaFromFile,
			Spectrum: spectraFromFile,
		}
		spectralookup[pmc] = append(spectralookup[pmc], s)
	} else {
		spectralookup[pmc] = nil
		s := dataConvertModels.DetectorSample{
			Meta:     metaFromFile,
			Spectrum: spectraFromFile,
		}
		spectralookup[pmc] = append(spectralookup[pmc], s)
	}

	return spectralookup
}

func makeBulkMaxSpectra(spectraLookup dataConvertModels.DetectorSampleByPMC, xperchanA float32, offsetA float32, xperchanB float32, offsetB float32) (int32, []dataConvertModels.DetectorSample) {
	specialPMC := int32(len(spectraLookup) + 1)

	generated := map[string]dataConvertModels.DetectorSample{}

	AXPERCHAN := float32(0)
	AXCount := 0
	AOFFSET := float32(0)
	AOffsetCount := 0
	BXPERCHAN := float32(0)
	BXCount := 0
	BOFFSET := float32(0)
	BOffsetCount := 0

	generated["bulkA"] = dataConvertModels.DetectorSample{
		Meta: dataConvertModels.MetaData{
			"PMC":         dataConvertModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": dataConvertModels.StringMetaValue("A"),
			"READTYPE":    dataConvertModels.StringMetaValue("BulkSum"),
			"SOURCEFILE":  dataConvertModels.StringMetaValue("GeneratedByPIXLISEConverter"),
		},
		Spectrum: []int64{},
	}

	generated["bulkB"] = dataConvertModels.DetectorSample{
		Meta: dataConvertModels.MetaData{
			"PMC":         dataConvertModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": dataConvertModels.StringMetaValue("B"),
			"READTYPE":    dataConvertModels.StringMetaValue("BulkSum"),
			"SOURCEFILE":  dataConvertModels.StringMetaValue("GeneratedByPIXLISEConverter"),
		},
		Spectrum: []int64{},
	}

	generated["maxA"] = dataConvertModels.DetectorSample{
		Meta: dataConvertModels.MetaData{
			"PMC":         dataConvertModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": dataConvertModels.StringMetaValue("A"),
			"READTYPE":    dataConvertModels.StringMetaValue("MaxValue"),
			"SOURCEFILE":  dataConvertModels.StringMetaValue("GeneratedByPIXLISEConverter"),
		},
		Spectrum: []int64{},
	}

	generated["maxB"] = dataConvertModels.DetectorSample{
		Meta: dataConvertModels.MetaData{
			"PMC":         dataConvertModels.IntMetaValue(specialPMC),
			"DETECTOR_ID": dataConvertModels.StringMetaValue("B"),
			"READTYPE":    dataConvertModels.StringMetaValue("MaxValue"),
			"SOURCEFILE":  dataConvertModels.StringMetaValue("GeneratedByPIXLISEConverter"),
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
				generated[bulkID] = dataConvertModels.DetectorSample{
					Meta:     generated[bulkID].Meta,
					Spectrum: make([]int64, l),
				}
			}

			if len(generated[maxID].Spectrum) <= 0 {
				l := len(detectorData.Spectrum)
				generated[maxID] = dataConvertModels.DetectorSample{
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
		x := dataConvertModels.FloatMetaValue(xperchanA)
		if AXCount > 0 {
			x = dataConvertModels.FloatMetaValue(AXPERCHAN / float32(AXCount))
		}
		generated["bulkA"].Meta["XPERCHAN"] = x
		generated["maxA"].Meta["XPERCHAN"] = x
	}

	if AOffsetCount > 0 || offsetA != 0 {
		x := dataConvertModels.FloatMetaValue(offsetA)
		if AOffsetCount > 0 {
			x = dataConvertModels.FloatMetaValue(AOFFSET / float32(AOffsetCount))
		}
		generated["bulkA"].Meta["OFFSET"] = x
		generated["maxA"].Meta["OFFSET"] = x
	}

	if BXCount > 0 || xperchanB != 0 {
		x := dataConvertModels.FloatMetaValue(xperchanB)
		if BXCount > 0 {
			x = dataConvertModels.FloatMetaValue(BXPERCHAN / float32(BXCount))
		}
		generated["bulkB"].Meta["XPERCHAN"] = x
		generated["maxB"].Meta["XPERCHAN"] = x
	}

	if BOffsetCount > 0 || offsetB != 0 {
		x := dataConvertModels.FloatMetaValue(offsetB)
		if BOffsetCount > 0 {
			x = dataConvertModels.FloatMetaValue(BOFFSET / float32(BOffsetCount))
		}
		generated["bulkB"].Meta["OFFSET"] = x
		generated["maxB"].Meta["OFFSET"] = x
	}

	// Set the live times for bulk spectra
	generated["bulkA"].Meta["LIVETIME"] = dataConvertModels.FloatMetaValue(liveTimeA)
	generated["bulkB"].Meta["LIVETIME"] = dataConvertModels.FloatMetaValue(liveTimeB)

	// Ensure no div by 0
	if liveTimeACount == 0 {
		liveTimeACount = 1
	}
	if liveTimeBCount == 0 {
		liveTimeBCount = 1
	}

	// Set average live time for max spectra
	generated["maxA"].Meta["LIVETIME"] = dataConvertModels.FloatMetaValue(liveTimeA / float32(liveTimeACount))
	generated["maxB"].Meta["LIVETIME"] = dataConvertModels.FloatMetaValue(liveTimeB / float32(liveTimeBCount))

	arr := []dataConvertModels.DetectorSample{generated["bulkA"], generated["bulkB"], generated["maxA"], generated["maxB"]}
	return specialPMC, arr
}
