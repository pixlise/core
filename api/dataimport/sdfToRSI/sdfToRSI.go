package sdfToRSI

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"path"
	"strconv"
	"strings"
)

// Given an SDF path and an output path, this generates RSI files for each scan mentioned in the SDF.
// Returns the file names generated and an error if any
func ConvertSDFtoRSIs(sdfPath string, outPath string) ([]string, []int64, error) {
	files := []string{}
	rtts := []int64{}

	refs, err := scanSDF(sdfPath)
	if err != nil {
		return files, rtts, err
	}

	// Loop through RTTs, output one RSI file per RTT
	rtt := int64(0)
	startLine := 0
	for _, ref := range refs {
		if ref.What == "new-rtt" {
			// Parse the RTT as hex
			rtt, err = strconv.ParseInt(ref.Value, 16, 32)
		} else if ref.What == "science" {
			if ref.Value == "begin" {
				startLine = ref.Line
			} else if ref.Value != "end" {
				return files, rtts, fmt.Errorf("End not found for science RTT: %v", rtt)
			} else {
				name := fmt.Sprintf("RSI-%v.csv", rtt)
				err = sdfToRSI(sdfPath, rtt, startLine, ref.Line, path.Join(outPath, name))
				if err != nil {
					return files, rtts, fmt.Errorf("Failed to generate %v: %v", name, err)
				}
				files = append(files, name)
				rtts = append(rtts, rtt)
			}
		}
	}

	return files, rtts, nil
}

func sdfToRSI(sdfPath string, rtt int64, startLine int, endLine int, outPath string) error {
	file, err := os.Open(sdfPath)
	if err != nil {
		return fmt.Errorf("Failed to open SDF %v: %v", sdfPath, err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	fout, err := os.Create(outPath)
	if err != nil {
		return fmt.Errorf("Failed to create output CSV %v: %v", outPath, err)
	}

	_, err = fout.WriteString(fmt.Sprintf("Spatial information from PIXL SDF or dat files %v for RTT: %v\n", sdfPath, rtt) +
		"SCLK, RTT, PMC, PDP category, PDP name, PDP information (content varies)\n" +
		"comment,,,, Housekeeping columns, Mtr1, Mtr2, Mtr3, Mtr4, Mtr5, Mtr6, SDD1_V, SDD2_V, Arm_R, SDD1_T, SDD2_T, SDD1_TEC_T, SDD2_TEC_T, Yellow_T, AFE_T, LVCM_T, HVMM_T, Fil_V, Fil_I, HV, Em_I\n")

	if err != nil {
		return fmt.Errorf("Failed to write output CSV headers for %v: %v", outPath, err)
	}

	state := ""
	currentDetector := ""

	outLinesByType := map[string][]string{}
	lastHKTime := int64(0)

	lineNo := 0
	lastPMC := 0

	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		if lineNo < startLine {
			continue
		}

		if lineNo > endLine {
			break
		}

		// Try decode the start of the line, SCLK is before :
		sep := strings.Index(line, " : ")
		if sep < 0 {
			return fmt.Errorf("Failed to read timestamp on line: %v, \"%v\"", lineNo, line)
		}

		sclk := line[0:sep]
		lineData := line[sep+3:]
		lineData = strings.Trim(lineData, " ")

		// Ignore lines starting with: "fpga", "LVL", "...", "0 "
		if strings.HasPrefix(lineData, "fpga ") || strings.HasPrefix(lineData, "inv ") || strings.HasPrefix(lineData, "sen ") || strings.HasPrefix(lineData, "LVL ") || strings.HasPrefix(lineData, "... ") || strings.HasPrefix(lineData, "0 ") {
			continue
		}

		// First thing should be the PMC
		var tok string
		var ok bool
		tok, lineData, ok = takeToken(lineData, " ")
		if !ok {
			return fmt.Errorf("Failed to read PMC on line: %v, \"%v\"", lineNo, line)
		}

		pmc, err := strconv.Atoi(tok)
		if err != nil {
			return fmt.Errorf("Invalid PMC on line: %v, \"%v\"", lineNo, line)
		}

		// If we just arrived at a new PMC, flush the outgoing stuff for the previous PMC
		if pmc != lastPMC {
			writeOutput(outLinesByType, fout)

			// Clear what we've written!
			outLinesByType = map[string][]string{}
			lastPMC = pmc
			lastHKTime = 0
		}

		// Find the line type
		tok, lineData, ok = takeToken(lineData, " ")
		if !ok {
			return fmt.Errorf("Failed to read line type on line: %v, \"%v\"", lineNo, line)
		}

		if len(state) > 0 && state != tok {
			state = "" // no longer reading whatever that was...
		}

		out := strings.Builder{}
		err = nil
		if tok == "gv" {
			if state != "gv" {
				// NOTE: we ignore gv until we find startTok on the line - we then expect/read gv lines until they stop coming
				startTok := "Filename token: \"_MCC_SLI_SpotList_BF\""

				if strings.HasSuffix(lineData, startTok) {
					state = "gv" // expect gv from now
					continue
				} else {
					// We're not interested in this gv
					continue
				}
			}

			err = processGV(lineNo, line, lineData, sclk, rtt, pmc, &out)
		} else if tok == "hk" {
			// Read all hk lines
			hkLines, err := readAheadLines(scanner, lineNo, 23)
			if err != nil {
				return fmt.Errorf("hk: %v", err)
			}

			hktime, hkline, err := processHousekeeping(lineNo, lineData, hkLines, sclk, rtt, pmc)
			if hktime == lastHKTime {
				// We overwrite in this case!
				hklinesSaved := len(outLinesByType[tok])
				outLinesByType[tok] = outLinesByType[tok][0 : hklinesSaved-1]
			}

			// just let it get written like anything else
			out.WriteString(hkline)

			lastHKTime = hktime
			lineNo += 23
		} else if tok == "scanlog" {
			err = processScanLog(lineNo, lineData, sclk, rtt, pmc, &out)
		} else if tok == "mcc_ram" {
			// If we're at entry 00384 we check which detector is being dumped for future reference as we read the centroids
			detector, ok, err := checkMCCDetector(lineData)

			// Only care if it read the right line
			if ok {
				if err != nil {
					// Report error and stop
					return fmt.Errorf("%v on line: %v", err, lineNo)
				}

				currentDetector = detector
				//fmt.Printf("Found mcc_ram detector: %v\n", currentDetector)
			}
		} else if tok == "mcc_trn" {
			if state != "mcc_trn" {
				state = "mcc_trn" // expect mcc_trn from now until we don't see it any more
			}

			// Example read:
			// 2022-301T22:12:23 : 1058 mcc_trn Reference: 0x0003 -- Sclk: 15:42:17   ---> Flags: 0x300E
			// 2022-301T22:12:23 : 1058 mcc_trn  Features Count: Reference: 266  -- Current: 283  -- Matches: 149 --  Residual:   3
			// 2022-301T22:12:23 : 1058 mcc_trn Reference Plane:     1.9962833     0.3108490     0.9504520  Dist:   55.7330017
			// 2022-301T22:12:23 : 1058 mcc_trn   Current Plane:     1.9778011     0.3285987     0.9442087  Dist:   55.7389984
			// 2022-301T22:12:23 : 1058 mcc_trn    TRN Solution:  -189.2319946  -270.0509949   -38.2099991
			// 2022-301T22:12:23 : 1058 mcc_trn 00000 : 00000003  0000345A  00002EDE  00000000  0000010A  0000011B  00000095  00000003
			// 2022-301T22:12:23 : 1058 mcc_trn 00032 : FF863633  27C9E67F  79A8697F  0000D9B5  FD28959D  2A0F85BF  78DBD4FF  0000D9BB
			// 2022-301T22:12:23 : 1058 mcc_trn 00064 : 0000300E  FFFD1CD0  FFFBE11D  FFFF6ABE

			// Check that we're at the first row
			if !strings.Contains(lineData, "---> Flags: ") {
				return fmt.Errorf("mcc_trn unexpected structure start on line: %v, \"%v\"", lineNo, line)
			}

			// We have this and 7 more lines to read in and parse together
			lines, err := readAheadLines(scanner, lineNo, 7)
			if err != nil {
				return fmt.Errorf("mcc_trn: %v", err)
			}

			err = processMCCTRN(lineNo, line, lineData, lines, sclk, rtt, pmc, &out)
			lineNo += 7
		} else if tok == "CenSLI_struct" {
			err = processCentroid(lineNo, line, lineData, scanner, currentDetector, sclk, rtt, pmc, &out)
		}

		if err != nil {
			// Stop here!
			return fmt.Errorf("ERROR line [%v], data type \"%v\": %v", lineNo, tok, err)
		}

		// If we have something to write, store it in the map, so we write in the correct order later
		s := out.String()
		if len(s) > 0 {
			existing, ok := outLinesByType[tok]
			if !ok {
				outLinesByType[tok] = []string{s}
			} else {
				existing = append(existing, s)
				outLinesByType[tok] = existing
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	writeOutput(outLinesByType, fout)

	return fout.Close()
}

func writeOutput(outLinesByType map[string][]string, fout *os.File) {
	// We save in this order:
	// gv aka _MCC_SLI_SpotList_BF
	// scanlog aka _Grand_Scan_Log
	// hk aka HK Frame
	// CenSLI_struct aka MCC SLI Estimates A/B
	// mcc_trn aka MCC OLM TRN Estimates
	writeOrder := []string{"gv", "scanlog", "hk", "CenSLI_struct", "mcc_trn"}
	for _, key := range writeOrder {
		lines, ok := outLinesByType[key]
		if ok {
			for _, line := range lines {
				_, err := fout.WriteString(line)
				if err != nil {
					log.Fatalf("Failed to write to output file: %v", err)
				}
			}

			delete(outLinesByType, key)
		}
	}

	// If any left at the end, we've got a bug
	if len(outLinesByType) > 0 {
		for k := range outLinesByType {
			log.Fatalf("Failed to write '%v' output file!", k)
		}
	}
}
