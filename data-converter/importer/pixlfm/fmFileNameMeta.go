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

package pixlfm

import (
	"errors"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode"

	"github.com/pixlise/core/core/logger"
)

// FileNameMeta See docs/PIXL_filename.docx
type FileNameMeta struct {
	instrument         string // PC=PIXL MCC, PE=PIXL engineering, PS=PIXL spectrometer
	colourFilter       string // R=red, G=green, B=blue, W=multiple, U=UV, D=SLI-A(dense), S=SLI-B(sparse), _=N/A, M=greyscale (PIXL MCC)
	special            string // Only images, otherwise _. If image, this is user defined special processing copy of an image, eg remove shadows. Ad-hoc, can look up in a DB
	primaryTimestamp   string // ____=out of range, cruise=Year-DOY(A-Z=2017-2042, 01-365 DOY), surface=SOL 4 integers, ground test either SOL, Year-DOY or DOY-Year
	venue              string // _=surface/cruise, A=AVSTB, F=FSWTB, M=MSTB, R=ROASTT, S=Scarecrow, V=VSTB
	secondaryTimestamp string // SCLK or ERT for ground tests: MMDDHHmmss
	// _ always separates
	ternaryTimestamp string // Milliseconds of SCLK or UTC
	prodType         string // Product identifier
	geometry         string // _=N/A, L=linearised with normal stereo partner, A=linearised with actual stereo partner
	thumbnail        string // _=N/A, T=thumbnail, N= nominal product (full frame, sub-frame or downsample)
	siteStr          string // Site ID 000-999=0-999, A00-A99=1000-1099 ... ZZ0-ZZ9=10350-10359
	driveStr         string // Drive count 0000-9999=0-9999, A000,A001-A999=10000-10999, etc
	seqRTT           string // SeqID (cmd seq img acquired from) OR RTT (PIXL files ONLY)
	camSpecific      string // PIXL MCC format PPPP = PMC
	downsample       string // 0=1x1, 1=2x2, 2=4x4, 3=8x8
	compression      string // 00=thumbnail, 01-99,A0=JPG quality, I1-I9=ICER, LI,LL,LM,LU=lossless
	producer         string // J=JPL, P=Principal investigator
	versionStr       string // 01-99=1-9, A0-A9=100-109, AA-AZ=110-135, B0-B9=136-145, __=out of range
	// . always before...
	// EXT - file extension, which we get through conventional Go filepath.Ext()
}

func (m *FileNameMeta) SetColourFilter(colourFilter string) {
	m.colourFilter = colourFilter
}

func (m *FileNameMeta) SetProdType(prodType string) {
	m.prodType = prodType
}

func (m *FileNameMeta) SetVersionStr(versionStr string) {
	m.versionStr = versionStr
}

func (m FileNameMeta) PMC() (int32, error) {
	// PMC is only stored by PIXL
	if m.instrument != "PC" && m.instrument != "PE" && m.instrument != "PS" {
		return 0, errors.New("PMC only stored for PIXL files")
	}
	i, err := strconv.Atoi(m.camSpecific)
	if err != nil {
		return 0, errors.New("Failed to get PMC from: " + m.camSpecific)
	}
	return int32(i), nil
}

func (m FileNameMeta) RTT() (int32, error) {
	// Seems RTT is usually stored, but this can be a seq ID
	// TODO: do we need to make this fail for some conditions?
	i, err := strconv.Atoi(m.seqRTT)
	if err != nil {
		return 0, errors.New("Failed to get RTT from: " + m.seqRTT)
	}
	return int32(i), nil
}

func (m FileNameMeta) SOL() (string, error) {
	return m.primaryTimestamp, nil
}

func (m FileNameMeta) SCLK() (int32, error) {
	/*if m.venue != "_" {
		return 0, errors.New("SCLK not stored for ground test: " + m.secondaryTimestamp)
	}*/
	i, err := strconv.Atoi(m.secondaryTimestamp)
	if err != nil {
		return 0, errors.New("Failed to get SCLK from: " + m.secondaryTimestamp)
	}
	return int32(i), nil
}

func (m FileNameMeta) site() (int32, error) {
	return stringToSiteID(m.siteStr)
}

func (m FileNameMeta) drive() (int32, error) {
	return stringToDriveID(m.driveStr)
}

func (m FileNameMeta) version() (int32, error) {
	return stringToVersion(m.versionStr)
}

func (m FileNameMeta) ToString() string {
	var s strings.Builder

	s.WriteString(m.instrument)
	s.WriteString(m.colourFilter)
	s.WriteString(m.special)
	s.WriteString(m.primaryTimestamp)
	s.WriteString(m.venue)
	s.WriteString(m.secondaryTimestamp)
	s.WriteString("_")
	s.WriteString(m.ternaryTimestamp)
	s.WriteString(m.prodType)
	s.WriteString(m.geometry)
	s.WriteString(m.thumbnail)
	s.WriteString(m.siteStr)
	s.WriteString(m.driveStr)
	s.WriteString(m.seqRTT)
	s.WriteString(m.camSpecific)
	s.WriteString(m.downsample)
	s.WriteString(m.compression)
	s.WriteString(m.producer)
	s.WriteString(m.versionStr)

	return s.String()
}

func (m *FileNameMeta) SetInstrumentType(instrumentType string) {
	m.instrument = instrumentType
}

// ParseFileName
/*
func (m FileNameMeta) Timestamp() (int32, error) {
	// Built from multiple bits of the structure...

	i, err := strconv.Atoi(m.camSpecific)
	return int32(i), err
}
*/
func ParseFileName(fileName string) (FileNameMeta, error) {
	fileNames := strings.Split(fileName, "/")
	if len(fileNames) > 1 {
		fileName = fileNames[1]
	}
	result := FileNameMeta{}

	if len(fileName) != 58 {
		return result, errors.New("Failed to parse meta from file name")
	}

	// Read anything we can get out of the file name
	// See docs/PIXL_filename.docx
	result.instrument = fileName[0:2]
	result.colourFilter = fileName[2:3]
	result.special = fileName[3:4]
	result.primaryTimestamp = fileName[4:8]
	result.venue = fileName[8:9]
	result.secondaryTimestamp = fileName[9:19]
	// _
	result.ternaryTimestamp = fileName[20:23]
	result.prodType = fileName[23:26]
	result.geometry = fileName[26:27]
	result.thumbnail = fileName[27:28]
	result.siteStr = fileName[28:31]
	result.driveStr = fileName[31:35]
	result.seqRTT = fileName[35:44]
	result.camSpecific = fileName[44:48]
	result.downsample = fileName[48:49]
	result.compression = fileName[49:51]
	result.producer = fileName[51:52]
	result.versionStr = fileName[52:54]
	//         "." = fileName[53:54]
	//         EXT = fileName[54:57]
	// Length 58 chars

	return result, nil
}

// Run through all file names, return a map of file name->parsed meta data for the latest
// files in the list. This is determined by looking at the versionStr and SCLK fields.
// The "latest" file has the highest version, AND lowest SCLK value.
// NOTE: If there are different kinds of files (different extensions), it returns the
// latest of each one, not just the latest of ALL files blindly.
func getLatestFileVersions(fileNames []string, jobLog logger.ILogger) map[string]FileNameMeta {
	byNonVerFields := map[string]map[string]FileNameMeta{}

	for _, file := range fileNames {
		meta, err := ParseFileName(file)
		if err != nil {
			jobLog.Infof("Failed to parse \"%v\": %v\n", file, err)
		} else {
			// Check we've got one for this file
			ext := strings.ToUpper(filepath.Ext(file))

			// Store the key as all the fields we're NOT interested in comparing:
			// this way if we have 2 TIF files with different PMCs, we won't think we need to ignore some due to versioning
			nonVerFields := ext + meta.instrument + meta.colourFilter + meta.prodType + meta.siteStr + meta.driveStr + meta.seqRTT + meta.camSpecific + meta.downsample + meta.compression + meta.producer

			if _, ok := byNonVerFields[nonVerFields]; !ok {
				// Add an empty map for this
				byNonVerFields[nonVerFields] = map[string]FileNameMeta{}
			}
			byNonVerFields[nonVerFields][file] = meta
		}
	}

	// Now pick out the highest version from each
	result := map[string]FileNameMeta{}

	for _, lookup := range byNonVerFields {
		selectedName := ""
		selectedVersion := int32(0)
		selectedSCLK := int32(0)
		var selectedMeta FileNameMeta

		for name, meta := range lookup {
			metaSCLK, err := meta.SCLK()
			if err != nil {
				jobLog.Infof("Failed to parse SCLK for \"%v\": %v\n", name, err)
			}
			metaVersion, err := meta.version()
			if err != nil {
				jobLog.Infof("Failed to parse version for \"%v\": %v\n", name, err)
			}

			if len(selectedName) <= 0 || (metaVersion > selectedVersion) || (metaVersion == selectedVersion && metaSCLK < selectedSCLK) {
				selectedName = name
				selectedMeta = meta
				selectedVersion = metaVersion
				selectedSCLK = metaSCLK
			}
		}

		if len(selectedName) > 0 {
			result[selectedName] = selectedMeta
		}
	}

	return result
}

func stringToIDSimpleCase(str string) (int32, bool) {
	if !isAllDigits(str) {
		return 0, false
	}

	iVal, err := strconv.Atoi(str)
	if err != nil {
		return 0, false
	}

	return int32(iVal), true
}

func isAllDigits(str string) bool {
	for _, c := range str {
		if !unicode.IsDigit(c) {
			return false
		}
	}
	return true
}

func isAlpha(c byte) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func letterValue(c byte) int32 {
	return int32(c) - 'A'
}

func stringToVersion(version string) (int32, error) {
	if len(version) == 2 {
		id, ok := stringToIDSimpleCase(version)
		if ok {
			return id, nil
		}

		if isAlpha(version[0]) && isAllDigits(version[1:]) {
			remainder, ok := stringToIDSimpleCase(version[1:])
			if ok {
				return 100 + letterValue(version[0])*36 + remainder, nil
			}
		}

		if isAlpha(version[0]) && isAlpha(version[1]) {
			return 110 + letterValue(version[0])*36 + letterValue(version[1]), nil
		}
	}
	return 0, fmt.Errorf("Failed to convert: %v to version", version)
}

func stringToSiteID(site string) (int32, error) {
	if len(site) == 3 {
		id, ok := stringToIDSimpleCase(site)
		if ok {
			return id, nil
		}

		if isAlpha(site[0]) && isAllDigits(site[1:]) {
			remainder, ok := stringToIDSimpleCase(site[1:])
			if ok {
				return 1000 + letterValue(site[0])*100 + remainder, nil
			}
		}

		if isAlpha(site[0]) && isAlpha(site[1]) && isAllDigits(site[2:]) {
			remainder, ok := stringToIDSimpleCase(site[2:])
			if ok {
				return 3600 + letterValue(site[0])*260 + letterValue(site[1])*10 + remainder, nil
			}
		}

		if isAlpha(site[0]) && isAlpha(site[1]) && isAlpha(site[2]) {
			return 10360 + letterValue(site[0])*26*26 + letterValue(site[1])*26 + letterValue(site[2]), nil
		}

		if isAllDigits(site[0:1]) && isAlpha(site[1]) && isAlpha(site[2]) {
			firstDigit, ok := stringToIDSimpleCase(site[0:1])
			if ok {
				val := 27936 + firstDigit*26*26 + letterValue(site[1])*26 + letterValue(site[2])
				if val < 32768 {
					return val, nil
				}
			}
		}
	}

	return 0, fmt.Errorf("Failed to convert: %v to site ID", site)
}

func stringToDriveID(drive string) (int32, error) {
	if len(drive) == 4 {
		id, ok := stringToIDSimpleCase(drive)
		if ok {
			return id, nil
		}

		if isAlpha(drive[0]) && isAllDigits(drive[1:]) {
			remainder, ok := stringToIDSimpleCase(drive[1:])
			if ok {
				return 10000 + letterValue(drive[0])*1000 + remainder, nil
			}
		}

		if isAlpha(drive[0]) && isAlpha(drive[1]) && isAllDigits(drive[2:]) {
			remainder, ok := stringToIDSimpleCase(drive[2:])
			if ok {
				val := 36000 + letterValue(drive[0])*2600 + letterValue(drive[1])*100 + remainder
				if val < 65536 {
					return val, nil
				}
			}
		}
	}
	return 0, fmt.Errorf("Failed to convert: %v to drive ID", drive)
}
