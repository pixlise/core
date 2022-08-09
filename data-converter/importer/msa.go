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

package importer

import (
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/pixlise/core/core/utils"
	"github.com/pixlise/core/data-converter/converterModels"
	protos "github.com/pixlise/core/generated-protos"
)

func ReadMSAFileLines(lines []string, singleDetectorMSA bool, expectPMC bool, detectorADuplicate bool) ([]converterModels.DetectorSample, error) {
	var err error
	// If single detector, we're reading:
	meta := converterModels.MetaData{}
	var spectra []int64

	// If multi-detector, we also read:
	metaB := converterModels.MetaData{}
	var spectraB []int64

	msaNumColumns := 1
	expColCount := 2

	if detectorADuplicate {
		expColCount = 1
	}

	startMarker := "#SPECTRUM"
	endMarker := "#ENDOFDATA"

	readingSpectra := false
	lc := 0

	for _, l := range lines {
		l = strings.TrimSpace(l)
		if len(l) > 0 {
			if strings.HasPrefix(l, startMarker) {
				if readingSpectra == true {
					return nil, fmt.Errorf("Unexpected spectrum start marker at %v", lc)
				}
				readingSpectra = true

				if singleDetectorMSA {
					err = verifyDetectorMSAMeta(meta, []string{"DETECTOR_ID", "NPOINTS", "DATATYPE", "NCOLUMNS"}, "Y", 1)
					if err != nil {
						return nil, err
					}
				} else {
					err = verifyDetectorMSAMeta(meta, []string{"NPOINTS", "DATATYPE", "NCOLUMNS"}, "YY", expColCount)
					if err != nil {
						return nil, err
					}

					msaNumColumns = expColCount

					if _, ok := meta["DETECTOR_ID"]; ok {
						return nil, errors.New("Unexpected DETECTOR_ID in multi-detector MSA")
					}
				}

			} else if len(l) >= len(endMarker) && l[0:len(endMarker)] == endMarker {
				if readingSpectra == false {
					return nil, fmt.Errorf("Unexpected end of data marker at %v", lc)
				}
				readingSpectra = false
				break
			} else if l[0] == '#' {
				if readingSpectra == true {
					return nil, errors.New("Unexpected # after started spectra read")
				}

				f, v, err := parseMSAMetadataLine(l)
				if err != nil {
					return nil, err
				}

				if _, ok := meta[f]; ok {
					if f == "COMMENT" {
						meta[f] = converterModels.MetaValue{
							SValue:   meta[f].SValue + " " + v,
							IValue:   meta[f].IValue,
							FValue:   meta[f].FValue,
							DataType: meta[f].DataType,
						}
					} else {
						return nil, fmt.Errorf("Duplicate meta data lines found for: %v", f)
					}
				} else {
					// Don't store it if there's no data!
					if len(v) > 0 {
						// At this point, if we're going to split this, save these as strings, because we know we will split
						// and convert them to the right data type in the next step
						if !singleDetectorMSA && (f == "XPERCHAN" || f == "OFFSET" || f == "LIVETIME" || f == "REALTIME") {
							meta[f] = converterModels.StringMetaValue(v)
						} else {
							meta[f], err = makeMetaValue(f, v)
							if err != nil {
								return nil, err
							}
						}
					}
				}
			} else {
				if readingSpectra != true {
					return nil, fmt.Errorf("Unexpected potential spectra found at %v: %v", lc, l)
				}

				spectrumRowData, err := parseMSASpectraLine(l, lc, msaNumColumns)
				if err != nil {
					return nil, err
				}

				spectra = append(spectra, spectrumRowData[0])

				if !singleDetectorMSA {
					readFromIdx := 1
					if detectorADuplicate {
						readFromIdx = 0
					}
					spectraB = append(spectraB, spectrumRowData[readFromIdx])
				}
			}
		}
		lc = lc + 1
	}

	if len(spectra) <= 0 {
		return nil, errors.New("No spectra data found to be read")
	}

	// Read point count
	npoints, err := strconv.Atoi(meta["NPOINTS"].SValue)
	if err != nil {
		return nil, errors.New("Failed to read NPOINTS, got: " + meta["NPOINTS"].SValue)
	}

	if int64(len(spectra)) != int64(npoints) {
		return nil, fmt.Errorf("Expected %v spectra, got %v", meta["NPOINTS"].SValue, len(spectra))
	}

	if !singleDetectorMSA && int64(len(spectraB)) != int64(npoints) {
		return nil, fmt.Errorf("Expected %v B spectra, got %v", meta["NPOINTS"].SValue, len(spectraB))
	}

	if _, ok := meta["PMC"]; ok {
		if !expectPMC {
			return nil, errors.New("PMC NOT expected, but was found in MSA")
		}
	} else {
		if expectPMC {
			return nil, errors.New("PMC expected, but not found in MSA")
		}
	}

	if !singleDetectorMSA {
		meta, metaB, err = splitMSAMetaFor2Detectors(meta, detectorADuplicate)
		if err != nil {
			return nil, err
		}
		verifyDetectorMSAMeta(metaB, []string{"NPOINTS", "DATATYPE", "NCOLUMNS"}, "YY", expColCount)
		if err != nil {
			return nil, err
		}
	}

	result := []converterModels.DetectorSample{
		{
			Meta:     meta,
			Spectrum: spectra,
		},
	}

	if !singleDetectorMSA {
		second := converterModels.DetectorSample{
			Meta:     metaB,
			Spectrum: spectraB,
		}
		result = append(result, second)
	}
	return result, nil
}

func splitMSAMetaFor2Detectors(meta converterModels.MetaData, detectorADuplicate bool) (converterModels.MetaData, converterModels.MetaData, error) {
	/*
	   An example of what we're splitting...

	   #XPERCHAN    :  10.0, 10.0    eV per channel
	   #OFFSET      :  0.0,   0.0    eV of first channel
	   #SIGNALTYPE  :  XRF
	   #COMMENT     :  2500 point scan of Scotland RR4 - Mar. 13, 2018.
	   #COMMENT     :  28 kV at 155 uA, 20 s spot scans, 200 micron steps, 10 mm (y) x 10 mm (x) area
	   #XPOSITION   :    0.000
	   #YPOSITION   :    0.000
	   #ZPOSITION   :    2.443
	   #LIVETIME    :  25.09,  25.08
	   #REALTIME    :  25.11,  25.11
	   ##TRIGGERS   : 45993, 43902
	   ##EVENTS     : 44690, 42823
	   ##KETEK_ICR  : 1833.1, 1750.7
	   ##KETEK_OCR  : 1780.1, 1705.7
	*/
	metaA := converterModels.MetaData{"DETECTOR_ID": converterModels.StringMetaValue("A")}
	metaB := converterModels.MetaData{"DETECTOR_ID": converterModels.StringMetaValue("B")}

	needsSplit := []string{"XPERCHAN", "OFFSET", "LIVETIME", "REALTIME", "TRIGGERS", "EVENTS", "KETEK_ICR", "KETEK_OCR", "OVERFLOWS", "UNDERFLOWS", "BASE_EVENTS", "RESETS", "OVER_ADCMAX"}

	var err error

	for k, val := range meta {
		if utils.StringInSlice(k, needsSplit) && val.DataType == protos.Experiment_MT_STRING {
			v := val.SValue
			v = strings.TrimSpace(v)

			parts := strings.Split(v, ", ")

			if len(parts) != 2 && !detectorADuplicate {
				return nil, nil, errors.New("Metadata row cannot be split for 2 detectors due to commas")
			}

			readBIdx := 1

			if detectorADuplicate {
				readBIdx = 0
			}

			metaA[k], err = makeMetaValue(k, strings.TrimSpace(parts[0]))
			if err != nil {
				return nil, nil, err
			}
			metaB[k], err = makeMetaValue(k, strings.TrimSpace(parts[readBIdx]))
			if err != nil {
				return nil, nil, err
			}
		} else {
			if val.DataType == protos.Experiment_MT_STRING {
				val.SValue = strings.TrimSpace(val.SValue)
			}

			metaA[k] = val
			metaB[k] = val
		}
	}

	return metaA, metaB, nil
}

func makeMetaValue(label string, value string) (converterModels.MetaValue, error) {
	asInt := []string{"PMC", "SCLK", "RTT"}
	asFloat := []string{"XPERCHAN", "OFFSET", "LIVETIME", "REALTIME", "XPOSITION", "YPOSITION", "ZPOSITION"}

	// Some fields need to be saved as different data types...

	if utils.StringInSlice(label, asInt) {
		// Must parse it as an int
		iV, err := strconv.ParseInt(value, 0, 32)
		if err != nil {
			return converterModels.StringMetaValue(""), fmt.Errorf("Failed to read integer for: %v, got: %v", label, value)
		}
		return converterModels.IntMetaValue(int32(iV)), nil
	} else if utils.StringInSlice(label, asFloat) {
		// Must parse it as float
		fV, err := strconv.ParseFloat(value, 32)
		if err != nil {
			return converterModels.StringMetaValue(""), fmt.Errorf("Failed to read float for: %v, got: %v", label, value)
		}
		return converterModels.FloatMetaValue(float32(fV)), nil
	}

	// Default is string...
	return converterModels.StringMetaValue(value), nil
}

func verifyDetectorMSAMeta(meta converterModels.MetaData, expFields []string, datatype string, ncolumns int) error {
	for _, exp := range expFields {
		if _, ok := meta[exp]; !ok {
			return fmt.Errorf("Failed to find %v in metadata", exp)
		}
	}

	if meta["DATATYPE"].SValue != datatype {
		return fmt.Errorf("Expected DATATYPE \"%v\" in MSA metadata", datatype)
	}

	nColsRead, err := strconv.Atoi(meta["NCOLUMNS"].SValue)
	if err != nil {
		return errors.New("Failed to read NCOLUMNS, got: " + meta["NCOLUMNS"].SValue)
	}

	if int64(ncolumns) != int64(nColsRead) {
		return fmt.Errorf("Expected NCOLUMNS \"%v\" in MSA metadata", ncolumns)
	}

	return nil
}

func parseMSAMetadataLine(line string) (string, string, error) {
	if line[0] != '#' {
		return "", "", errors.New("Expected # at start of metadata: " + line)
	}

	colIdx := strings.Index(line, ":")

	if colIdx < 0 {
		return "", "", errors.New("Failed to parse metadata line: " + line)
	}

	field := strings.TrimLeft(strings.TrimSpace(line[:colIdx]), "#")
	value := line[colIdx+1:]

	// NOTE: we get some weird situations where there are comments on the line after real data
	// for now, it looks like there are always several spaces before the comment, so for example:
	// #NCOLUMNS    : 2     Number of data columns
	// we can find the first value text, and if there is anything after it with multiple spaces, we
	// can trim it there
	// Another situation is it seems if the entire thing is a comment it has more than 4 spaces
	// at the start. Some valid values start at up to 4 spaces, so if we see 5+ spaces at the start
	// we consider the entire thing to be a comment
	// Another weirder consideration... if we have this:
	// "0.0,   0.0    eV of first channel"
	// We really want to determine that there is float,float and cut the rest off. So first test for that
	bits := strings.Split(value, ",")
	done := false
	if len(bits) >= 2 {
		// if the first 2 values are floats, assume we've just read float,float and stop
		str1 := strings.Trim(bits[0], " ")
		_, e1 := strconv.ParseFloat(str1, 32)
		secondbits := strings.Split(strings.Trim(bits[1], " "), " ")
		if len(secondbits) >= 1 {
			str2 := strings.Trim(secondbits[0], " ")
			_, e2 := strconv.ParseFloat(str2, 32)
			if e1 == nil && e2 == nil {
				value = fmt.Sprintf("%v, %v", str1, str2)
				done = true
			}
		}
	}

	if !done {
		if len(value) > 5 && value[0:5] == "     " {
			value = ""
		} else {
			// Trim left hand spaces
			value = strings.TrimLeft(value, " ")

			// Now find anything we can chop off inside
			multiSpacePos := strings.Index(value, "  ")
			if multiSpacePos > 0 {
				value = value[0:multiSpacePos]
			}
		}
	}

	// Final trim, in case we had a space before/after, eg // #NCOLUMNS    : 2
	value = strings.TrimSpace(value)

	return field, value, nil
}

func parseMSASpectraLine(line string, lc int, ncolumns int) ([]int64, error) {
	items := strings.Split(line, ",")

	if len(items) != ncolumns {
		return nil, fmt.Errorf("Expected %d spectrum columns, got %d on line [%d]:%s", ncolumns, len(items), lc, line)
	}

	var specvals []int64

	for _, v := range items {
		val := strings.TrimSpace(v)

		specval, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("Failed to read spectra \"%v\" on line [%v]:%v", val, lc, line)
		}

		if specval < 0 {
			return nil, fmt.Errorf("Spectra expected non-negative value \"%v\" on line [%v]:%v", val, lc, line)
		}

		specvals = append(specvals, specval)
	}

	return specvals, nil
}
