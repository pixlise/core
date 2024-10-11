package sdfToRSI

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"
	"time"
)

func ConvertSDFtoRSI(sdfPath string, outPath string) error {
	rtts, _ /*minSCLK*/, err := scanForBasicStats(sdfPath)
	if err != nil {
		return err
	}

	// Find the max RTT, that's what we'll be writing
	rtt := int64(0)
	for _, r := range rtts {
		if r > rtt {
			rtt = r
		}
	}

	// Now read it again

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

	_, err = fout.WriteString("Spatial information from PIXL SDF or dat files " + sdfPath + "\n" +
		"SCLK, RTT, PMC, PDP category, PDP name, PDP information (content varies)\n" +
		"comment,,,, Housekeeping columns, Mtr1, Mtr2, Mtr3, Mtr4, Mtr5, Mtr6, SDD1_V, SDD2_V, Arm_R, SDD1_T, SDD2_T, SDD1_TEC_T, SDD2_TEC_T, Yellow_T, AFE_T, LVCM_T, HVMM_T, Fil_V, Fil_I, HV, Em_I\n")

	if err != nil {
		return fmt.Errorf("Failed to write output CSV headers for %v: %v", outPath, err)
	}

	state := ""

	lineNo := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		// Skip blank/comment lines at the start starting with *** or : or ::
		if len(line) <= 0 || strings.HasPrefix(line, "*** ") || strings.HasPrefix(line, ":") {
			continue
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

		// Find the line type
		tok, lineData, ok = takeToken(lineData, " ")
		if !ok {
			return fmt.Errorf("Failed to read line type on line: %v, \"%v\"", lineNo, line)
		}

		if len(state) > 0 && state != tok {
			state = "" // no longer reading whatever that was...
		}

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

			err = processGV(lineNo, line, lineData, sclk, rtt, pmc, fout)
			if err != nil {
				return err
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
			// 2022-301T22:12:23 : 1058 Wrote 0 bytes to raw file 'trn_0720267208_0C6F0202_001058.unc'.

			// Check that we're at the first row
			if !strings.Contains(lineData, "---> Flags: ") {
				return fmt.Errorf("mcc_trn unexpected structure start on line: %v, \"%v\"", lineNo, line)
			}

			// We have this and 7 more lines to read in and parse together
			lines := []string{}
			for c := 0; c < 7; c++ {
				if !scanner.Scan() {
					return fmt.Errorf("mcc_trn missing line %v line: %v, \"%v\"", c, lineNo, line)
				}

				lines = append(lines, scanner.Text())
				lineNo++
			}

			err = processMCCTRN(lineNo, line, lineData, lines, sclk, rtt, pmc, fout)
			if err != nil {
				return err
			}
		} else if tok == "CenSLI_struct" {
			// Example - a block of these lines:
			// 2022-301T14:53:53 :   44 CenSLI_struct  0 -- pixel x,y,intensity: [0x7ab6fdcf] [0x37d1f047] [0x03f1] |     411.7626     187.3011
			// 2022-301T14:53:53 :   44 CenSLI_struct  0 -- position x,y,z: [0x00000b28] [0xffffe82d] [0x0000e031]  |     0.002856    -0.006099     0.057393
			// 2022-301T14:53:53 :   44 CenSLI_struct  0 -- ID: 0x0a, Residual: 0x08

			sliNum, pixX, pixY, intensity, err := processCentroidLine1(lineNo, line, lineData)
			if err != nil {
				return err
			}

			// Read next line to get x,y,z
			if !scanner.Scan() {
				return fmt.Errorf("CenSLI_struct missing second line on line: %v, \"%v\"", lineNo, line)
			}

			line = scanner.Text()
			lineNo++

			x, y, z, err := processCentroidLine2(lineNo, line, sliNum)
			if err != nil {
				return err
			}

			// Read last line to get ID/Residual
			if !scanner.Scan() {
				return fmt.Errorf("CenSLI_struct missing second line on line: %v, \"%v\"", lineNo, line)
			}

			line = scanner.Text()
			lineNo++

			id, res, err := processCentroidLine3(lineNo, line, sliNum)
			if err != nil {
				return err
			}

			// Example output:
			// TODO: Not sure how we tell A from B!!!
			// 2AEE2547, C6F0202, 2, 57, MCC SLI Estimates B, 183.552124, 39.678978, 961.000000, -0.009290, -0.013930, 0.059620, 74.000000, 0.300000
			_, err = fout.WriteString(fmt.Sprintf("%v, %X, %v, 57, MCC SLI Estimates ???, %.6f, %.6f, %.6f, %.6f, %.6f, %.6f, %.6f, %.6f\n",
				makeWriteSCLK(sclk), rtt, pmc, pixX, pixY, intensity, x, y, z, float32(id), float32(res)/10))
			if err != nil {
				return fmt.Errorf("Failed to write to output CSV: %v", err)
			}
		}

		//fmt.Printf("%v,%v,%v\n", minSCLK, sclk, pmc)
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return fout.Close()
}

// Expecting inputs to be:
// lineData: Reference: 0x0003 -- Sclk: 15:42:17   ---> Flags: 0x300E
// lines: [
//
//	2022-301T22:12:23 : 1058 mcc_trn  Features Count: Reference: 266  -- Current: 283  -- Matches: 149 --  Residual:   3
//	2022-301T22:12:23 : 1058 mcc_trn Reference Plane:     1.9962833     0.3108490     0.9504520  Dist:   55.7330017
//	2022-301T22:12:23 : 1058 mcc_trn   Current Plane:     1.9778011     0.3285987     0.9442087  Dist:   55.7389984
//	2022-301T22:12:23 : 1058 mcc_trn    TRN Solution:  -189.2319946  -270.0509949   -38.2099991
//	2022-301T22:12:23 : 1058 mcc_trn 00000 : 00000003  0000345A  00002EDE  00000000  0000010A  0000011B  00000095  00000003
//	2022-301T22:12:23 : 1058 mcc_trn 00032 : FF863633  27C9E67F  79A8697F  0000D9B5  FD28959D  2A0F85BF  78DBD4FF  0000D9BB
//	2022-301T22:12:23 : 1058 mcc_trn 00064 : 0000300E  FFFD1CD0  FFFBE11D  FFFF6ABE
//
// ]
func processMCCTRN(lineNo int, line string, lineData string, lines []string, sclk string, rtt int64, pmc int, fout *os.File) error {
	if len(lines) != 7 {
		return fmt.Errorf("mcc_trn line count invalid on line %v", lineNo)
	}

	// Snip all lines so they start after mcc_trn
	tok := fmt.Sprintf("%v mcc_trn", pmc)
	for c := 0; c < 7; c++ {
		pos := strings.Index(lines[c], tok)
		if pos < 0 {
			return fmt.Errorf("%v not found on line %v", tok, lineNo)
		}

		lines[c] = strings.Trim(lines[c][pos+len(tok):], " ")
	}

	// Read fields from each line as expected, in order expected...
	ref, lastPos, err := readIntBetween(lineData, "Reference: 0x", " ", true)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	lineData = lineData[lastPos:]

	flags, _, err := readIntBetween(lineData, "---> Flags: 0x", " ", true)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}

	// Line 2
	lineData = lines[0]
	ref2, lastPos, err := readIntBetween(lineData, "Reference: ", " ", false)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	lineData = lineData[lastPos:]

	curr, lastPos, err := readIntBetween(lineData, "Current: ", " ", false)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	lineData = lineData[lastPos:]

	match, _ /*lastPos*/, err := readIntBetween(lineData, "Matches: ", " ", false)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	/*lineData = lineData[lastPos:]

	residual, lastPos, err := readIntBetween(lineData, "Residual: ", " ", false)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	*/

	// Expected output:
	// 2AEE67C8, C6F0202, 1058, 56, MCC OLM TRN Estimates, 3, 300E, 266, 283, 149, -0.0037167, 0.3108490, 0.9504520, 55.7330000, -0.0221990, 0.3285987, 0.9442087, 55.7390000, -189.2320000, -270.0510000, -38.2100000

	writeLine := fmt.Sprintf("%v, %X, %v, 56, MCC OLM TRN Estimates, %X, %X, %v, %v, %v", makeWriteSCLK(sclk), rtt, pmc, ref, flags, ref2, curr, match)

	writeVals := []float32{}

	for _, val := range writeVals {
		writeLine += fmt.Sprintf(", %v", val)
	}
	writeLine += "\n"

	_, err = fout.WriteString(writeLine)
	if err != nil {
		return fmt.Errorf("Failed to write to output CSV: %v", err)
	}

	return nil
}

// Returns the centroid number, x, y, intensity, error if any
func processCentroidLine1(lineNo int, line string, lineData string) (int64, float32, float32, float32, error) {
	// Expecting:
	// 0 -- pixel x,y,intensity: [0x7ab6fdcf] [0x37d1f047] [0x03f1] |     411.7626     187.3011

	// Read the number (0 in above example)
	var sliNum int64
	var err error
	var ok bool
	var tok string
	sliNum, lineData, err = readInt(lineData)

	// If we encountered, we expect to be on the first line, if so, swallow all 3
	startTok := "-- pixel x,y,intensity: "

	pos := strings.Index(lineData, startTok)
	if pos < 0 {
		return 0, 0, 0, 0, fmt.Errorf("Encountered CenSLI_struct but not start of structure on line: %v, \"%v\"", lineNo, line)
	}

	// Read the rest. Note we want what's in the last [] and the following 2 floats
	lineData = lineData[pos+len(startTok):]

	// Skip 2 []
	_, lineData, ok = takeToken(lineData, " ")
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("Invalid CenSLI_struct [1] on line: %v, \"%v\"", lineNo, line)
	}
	_, lineData, ok = takeToken(lineData, " ")
	if !ok {
		return 0, 0, 0, 0, fmt.Errorf("Invalid CenSLI_struct [2] on line: %v, \"%v\"", lineNo, line)
	}

	// Read the hex value encased in []
	var strVal string
	strVal, lineData, ok = takeToken(lineData, " ")
	if !ok || !strings.HasPrefix(strVal, "[0x") || !strings.HasSuffix(strVal, "]") {
		return 0, 0, 0, 0, fmt.Errorf("Invalid CenSLI_struct intensity on line: %v, \"%v\"", lineNo, line)
	}

	strVal = strVal[3 : len(strVal)-1]

	// Now parse it
	intensity, err := strconv.ParseInt(strVal, 16, 32)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("Failed to parse CenSLI_struct intensity (%v) on line: %v, \"%v\"", strVal, lineNo, line)
	}

	// Expect a pipe
	tok, lineData, ok = takeToken(lineData, " ")
	if !ok || tok != "|" {
		return 0, 0, 0, 0, fmt.Errorf("CenSLI_struct expected | on line: %v, \"%v\"", lineNo, line)
	}

	// Read x, y
	var x float32
	x, lineData, err = readFloat(lineData)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("CenSLI_struct failed to read x on line: %v, \"%v\"", lineNo, line)
	}

	var y float32
	y, lineData, err = readFloat(lineData)
	if err != nil {
		return 0, 0, 0, 0, fmt.Errorf("CenSLI_struct failed to read x on line: %v, \"%v\"", lineNo, line)
	}

	return sliNum, x, y, float32(intensity), nil
}

func processCentroidSubsequentLine(lineNo int, line string, sliNum int64) (string, error) {
	startTok := "CenSLI_struct"
	pos := strings.Index(line, startTok)
	if pos < 0 {
		return "", fmt.Errorf("Invalid centroid line 2 on line: %v, \"%v\"", lineNo, line)
	}

	lineData := strings.TrimLeft(line[pos+len(startTok):], " ")
	var readSliNum int64
	var err error
	readSliNum, lineData, err = readInt(lineData)
	if err != nil {
		return "", fmt.Errorf("Failed to read centroid num on line: %v, \"%v\". Error: %v", lineNo, line, err)
	}

	if readSliNum != sliNum {
		return "", fmt.Errorf("Centroid number mismatch %v, expected %v on line: %v, \"%v\"", readSliNum, sliNum, lineNo, line)
	}

	return lineData, nil
}

// Returns x, y, z, and verifies sliNum matches error if any
func processCentroidLine2(lineNo int, line string, sliNum int64) (float32, float32, float32, error) {
	// Example input:
	// 2022-301T17:28:38 :    2 CenSLI_struct  0 -- position x,y,z: [0xffffdbb9] [0xffffc999] [0x0000e8e3]  |    -0.009287    -0.013927     0.059619
	lineData, err := processCentroidSubsequentLine(lineNo, line, sliNum)
	if err != nil {
		return 0, 0, 0, err
	}

	// Read the rest!
	pos := strings.Index(lineData, "|")
	if pos < 0 {
		return 0, 0, 0, fmt.Errorf("Failed to find x,y,z on line: %v, \"%v\"", lineNo, line)
	}

	lineData = strings.TrimLeft(lineData[pos+1:], " ")

	var x, y, z float32
	x, lineData, err = readFloat(lineData)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Failed to read x on line: %v, \"%v\"", lineNo, line)
	}

	y, lineData, err = readFloat(lineData)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Failed to read x on line: %v, \"%v\"", lineNo, line)
	}

	z, lineData, err = readFloat(lineData)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("Failed to read x on line: %v, \"%v\"", lineNo, line)
	}

	return x, y, z, nil
}

// Returns x, y, z, and verifies sliNum matches error if any
func processCentroidLine3(lineNo int, line string, sliNum int64) (int64, int64, error) {
	// Example input:
	// 2022-301T17:28:38 :    2 CenSLI_struct  0 -- ID: 0x4a, Residual: 0x03
	lineData, err := processCentroidSubsequentLine(lineNo, line, sliNum)
	if err != nil {
		return 0, 0, err
	}

	// Read after ID:, then after Residual:
	tokens := []string{"ID: 0x", "Residual: 0x"}
	vals := []int64{}

	for c := 0; c < 2; c++ {
		idToken := tokens[c]
		pos := strings.Index(lineData, idToken)
		if pos < 0 {
			return 0, 0, fmt.Errorf("Failed to find id/res on line: %v, \"%v\"", lineNo, line)
		}

		lineData = strings.TrimLeft(lineData[pos+len(idToken):], " ")

		var ok bool
		var tok string
		tok, lineData, ok = takeToken(lineData, ",")

		if !ok {
			return 0, 0, fmt.Errorf("Failed to read %v on line: %v, \"%v\"", idToken, lineNo, line)
		}

		val, err := strconv.ParseInt(tok, 16, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("Failed to read id on line: %v, \"%v\"", lineNo, line)
		}

		vals = append(vals, val)
	}

	return vals[0], vals[1], nil
}

func processGV(lineNo int, line string, lineData string, sclk string, rtt int64, pmc int, fout *os.File) error {
	var ok bool
	var err error

	// Example read:
	// 2022-301T14:31:18 :    1 gv - 0x018800 : BDFF66C6 3E1D61C3 3E7C78D2 00000000 ::   -0.12470774    0.15369324    0.24655464             0
	// 2022-301T13:54:41 :   24 gv - 0x00b7d4 : 0000188E 000018A2 000018F3 000020D3 ::          6286          6306          6387          8403
	// At this point, we have lineData set to:
	// - 0x018800 : BDFF66C6 3E1D61C3 3E7C78D2 00000000 ::   -0.12470774    0.15369324    0.24655464             0

	_, lineData, ok = takeToken(lineData, " :: ")
	if !ok {
		// There are other kinds of "gv" lines, eg:
		// 2022-301T13:53:02 :    2 gv - Start Indx: 220 [0x000000DC] Length: 8 bytes [00000008] Filename token: "_HES_and_HESSaved"
		return fmt.Errorf("Failed to read line gv data on line: %v, \"%v\"", lineNo, line)
		//continue
	}

	// Read the 3 values
	vals := []float32{}
	var f float32
	for c := 0; c < 3; c++ {
		f, lineData, err = readFloat(lineData)
		if err != nil {
			// There are other kinds of "gv" lines, eg:
			// 2022-301T13:53:02 :    2 gv - 0x0000dc : 00000200 00000200                   ::           512           512
			return fmt.Errorf("Failed to read line gv coord %v on line: %v, \"%v\". Error: %v", c, lineNo, line, err)
			//break
		}

		vals = append(vals, f)
	}

	if len(vals) != 3 {
		// Didn't read enough
		fmt.Printf("Expected 3 floats for gv, got %v\n", len(vals))
		return nil
	}

	// We may also read other lines that aren't what we're after, eg:
	// 2022-301T13:54:41 :   24 gv - 0x00b7d4 : 0000188E 000018A2 000018F3 000020D3 ::          6286          6306          6387          8403

	// Output example:
	// 2AEE3F04, C6F0202, 404, 5, _MCC_SLI_SpotList_BF, -0.136560, 0.137250, 0.246220

	_, err = fout.WriteString(fmt.Sprintf("%v, %X, %v, 5, _MCC_SLI_SpotList_BF, %.6f, %.6f, %.6f\n",
		makeWriteSCLK(sclk), rtt, pmc, vals[0], vals[1], vals[2]))
	if err != nil {
		return fmt.Errorf("Failed to write to output CSV: %v", err)
	}

	return nil
}

// Returns RTT, lowest SCLK, error
func scanForBasicStats(sdfPath string) ([]int64, int64, error) {
	file, err := os.Open(sdfPath)
	if err != nil {
		return []int64{}, 0, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	rttMap := map[int64]bool{}
	sclk := int64(0)

	lineNo := 0
	for scanner.Scan() {
		line := scanner.Text()
		lineNo++

		tok, ok := findToken(line, "- HK Time: 0x", " ")
		if ok && len(tok) > 0 {
			// Read the SCLK as hex number
			thisSclk, err := strconv.ParseInt(tok, 16, 32)
			if err != nil {
				return []int64{}, 0, fmt.Errorf("Failed to read sclk from line %v: \"%v\". Error: %v", lineNo, line, err)
			}

			if sclk == 0 {
				sclk = thisSclk
			} else if thisSclk < sclk {
				sclk = thisSclk
			}
		}

		tok, ok = findToken(line, " RTT: ", " ")
		if ok && len(tok) > 0 && !strings.HasPrefix(tok, "0x") {
			// Read the RTT as hex number
			thisRTT, err := strconv.ParseInt(tok, 16, 32)
			if err != nil {
				return []int64{}, 0, fmt.Errorf("Failed to read RTT from line %v: \"%v\". Error: %v", lineNo, line, err)
			}

			if thisRTT > 0 {
				rttMap[thisRTT] = true
			}
		}
	}

	rtts := []int64{}
	for rtt := range rttMap {
		rtts = append(rtts, rtt)
	}
	return rtts, sclk, nil
}

func findToken(line string, prefix string, endDelim string) (string, bool) {
	pos := strings.Index(line, prefix)
	if pos > -1 {
		// Find the end
		start := pos + len(prefix)
		end := strings.Index(line[start:], endDelim)
		if end > -1 {
			end = start + end
			return line[start:end], true
		} else {
			// didn't find another space, we might be at the end of the line, see if token length is > 0 here
			if start < len(line) {
				return line[start:], true
			}
		}
	}

	return "", false
}

// Split off the first token (space separated), returns token, rest-of-line, error if any
func takeToken(line string, sep string) (string, string, bool) {
	pos := strings.Index(line, sep)
	if pos > 0 {
		return line[0:pos], strings.TrimLeft(line[pos+len(sep):], " "), true
	}

	// If we're at the end of the line, just return that as the token
	if len(line) > 0 {
		return line, "", true
	}

	return "", line, false
}

func readFloat(line string) (float32, string, error) {
	fStr, remainder, ok := takeToken(line, " ")
	if !ok {
		return 0, line, fmt.Errorf("Failed to read token")
	}

	f, err := strconv.ParseFloat(fStr, 32)
	if err != nil {
		return 0, remainder, fmt.Errorf("Failed to parse float: %v. Error: %v", fStr, err)
	}

	return float32(f), remainder, nil
}

func readInt(line string) (int64, string, error) {
	iStr, remainder, ok := takeToken(line, " ")
	if !ok {
		return 0, line, fmt.Errorf("Failed to read token")
	}

	i, err := strconv.ParseInt(iStr, 10, 32)
	if err != nil {
		return 0, remainder, fmt.Errorf("Failed to parse int: %v. Error: %v", iStr, err)
	}

	return i, remainder, nil
}

// For example, reading a value marked with VVV "Reference: 0xVVV,"
func readIntBetween(line string, prefix string, suffix string, isHex bool) (int64, int, error) {
	pos := strings.Index(line, prefix)
	if pos < 0 {
		return 0, 0, fmt.Errorf("failed to find value after %v", prefix)
	}
	lastPos := pos + len(prefix)
	line = strings.TrimLeft(line[lastPos:], " ")

	// If there is no suffix, just read till the next space OR end of string
	if len(suffix) <= 0 {
		suffix = " "
	}

	var ok bool
	var tok string
	tok, _, ok = takeToken(line, suffix)
	if !ok {
		return 0, 0, fmt.Errorf("failed to read suffix '%v' after '%v'", suffix, prefix)
	}
	lastPos += len(tok)

	radix := 10
	if isHex {
		radix = 16
	}

	val, err := strconv.ParseInt(tok, radix, 32)
	if err != nil {
		return 0, 0, fmt.Errorf("failed to read value after '%v'", prefix)
	}

	return val, lastPos, nil
}

func makeWriteSCLK(readSCLK string) string {
	// Read the SCLK value as a string time, expecting eg 2022-301T14:31:18
	t, err := time.Parse("2006-002T15:04:05", readSCLK)
	if err != nil {
		log.Fatalf("Failed to parse read SCLK: %v. Error: %v", readSCLK, err)
	}

	// Bit of a hack/approximation. We assume that SCLK is in seconds, and our reference is:
	// 2022-301T14:31:19 ==  Friday, October 28, 2022 14:31:19 == 1666967479 == 0x2AEDFBB8 == 720239544
	unixSec := t.Unix()

	// Work out where we are relative to the epoch above, and add to it to work out SCLK to use
	secSinceEpoch := unixSec - 1666967479

	sclk := 720239544 + secSinceEpoch

	// Return as hex
	return fmt.Sprintf("%X", sclk)
}
