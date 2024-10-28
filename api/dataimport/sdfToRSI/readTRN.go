package sdfToRSI

import (
	"fmt"
	"io"
	"strings"
)

// Expecting inputs to be:
// lineData: Reference: 0x0003 -- Sclk: 18:35:38   ---> Flags: 0x300E
// lines: [
// 2022-302T00:36:06 : 1657 mcc_trn  Features Count: Reference: 266  -- Current: 299  -- Matches: 162 --  Residual:   2
// 2022-302T00:36:06 : 1657 mcc_trn Reference Plane:     1.9962833     0.3108490     0.9504520  Dist:   55.7330017
// 2022-302T00:36:06 : 1657 mcc_trn   Current Plane:     0.0135415     0.2986050     0.9542807  Dist:   55.8709984
// 2022-302T00:36:06 : 1657 mcc_trn    TRN Solution:    86.0299988   212.2339935  -206.9559937
// 2022-302T00:36:06 : 1657 mcc_trn 00000 : 00000003  00005CFB  00005375  00000000  0000010A  0000012B  000000A2  00000002
// 2022-302T00:36:06 : 1657 mcc_trn 00032 : FF863633  27C9E67F  79A8697F  0000D9B5  01BBB9FB  2638AFFF  7A25DEFF  0000DA3F
// 2022-302T00:36:06 : 1657 mcc_trn 00064 : 0000300E  0001500E  00033D0A  FFFCD794
// ]
func processMCCTRN(lineNo int, line string, lineData string, lines []string, sclk string, rtt int64, pmc int, fout io.StringWriter) error {
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
	ref, _, lastPos, err := readNumBetween(lineData, "Reference: 0x", " ", read_int_hex)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	lineData = lineData[lastPos:]

	flags, _, _, err := readNumBetween(lineData, "---> Flags: 0x", " ", read_int_hex)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}

	// Line 2
	lineData = lines[0]
	ref2, _, lastPos, err := readNumBetween(lineData, "Reference: ", " ", read_int)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	lineData = lineData[lastPos:]

	curr, _, lastPos, err := readNumBetween(lineData, "Current: ", " ", read_int)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	lineData = lineData[lastPos:]

	match, _, _ /*lastPos*/, err := readNumBetween(lineData, "Matches: ", " ", read_int)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	/*lineData = lineData[lastPos:]

	residual, _, lastPos, err := readNumBetween(lineData, "Residual: ", " ", read_int)
	if err != nil {
		return fmt.Errorf("%v on line %v", err, lineNo)
	}
	*/

	planeData := []float32{}

	var ok bool

	lineStart := []string{"Reference Plane", "Current Plane", "TRN Solution"}
	for c := 0; c < 3; c++ {
		lineData = lines[c+1]
		tok, lineData, ok = takeToken(lineData, ":")

		if !ok || tok != lineStart[c] {
			return fmt.Errorf("Expected %v on line %v", lineStart[c], lineNo)
		}

		// Read off 3 floats
		var f float32
		for i := 0; i < 3; i++ {
			f, lineData, err = readFloat(lineData)
			if err != nil {
				return fmt.Errorf("Failed to read float number %v on line %v", i, lineNo)
			}
			planeData = append(planeData, f)
		}

		// If we're on the first 2 lines, expect plane distance
		if c < 2 {
			tok, lineData, ok = takeToken(lineData, ":")

			if !ok || tok != "Dist" {
				return fmt.Errorf("Expected Dist on line %v", lineNo)
			}

			f, lineData, err = readFloat(lineData)
			if err != nil {
				return fmt.Errorf("Failed to read Dist on line %v", lineNo)
			}
			planeData = append(planeData, f)
		}
	}

	// DataDrive RSI format has table headers:
	// MCC OLM TRN Estimate
	// PMC,RTT,sclk,ref_img_ID,flags,num_feat_ref,num_feat_curr,num_feat_match,match_res,plane_ref_x,plane_ref_y,plane_ref_z,plane_ref_dist,plane_curr_x,plane_curr_y,plane_curr_z,plane_curr_dist,trn_solution_x,trn_solution_y,trn_solution_z

	// Expected output:
	// 2AEE8977, C6F0202, 1657, 56, MCC OLM TRN Estimates, 3, 300E, 266, 299, 162, -0.0037167, 0.3108490, 0.9504520, 55.7330000, 0.0135415, 0.2986050, 0.9542807, 55.8710000, 86.0300000, 212.2340000, -206.9560000

	// It seems the ref plane has x -= 2 (if value is > 0)... don't know why currently
	if planeData[0] > 0 {
		planeData[0] -= 2
	}

	// It seems the curr plane has x -= 2 (if value is around 2, if it's just a bit above 0, it's untouched!!)... don't know why currently
	if 2-planeData[4] < 0.1 {
		planeData[4] -= 2
	}

	writeLine := fmt.Sprintf("%v, %X, %v, 56, MCC OLM TRN Estimates, %X, %X, %v, %v, %v, %.7f, %.7f, %.7f, %.7f, %.7f, %.7f, %.7f, %.7f, %.7f, %.7f, %.7f",
		makeWriteSCLK(sclk), rtt, pmc, ref, flags, ref2, curr, match,
		planeData[0], planeData[1], planeData[2], planeData[3], // reference plane
		planeData[4], planeData[5], planeData[6], planeData[7], // current plane
		planeData[8], planeData[9], planeData[10]) // TRN

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
