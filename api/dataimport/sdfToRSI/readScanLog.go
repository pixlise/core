package sdfToRSI

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Expects:
// 1655: |   -0.1360000    0.1310000    0.2462323  0x00120000 |    0.0000000    0.0000000    0.0000000    1657
func processScanLog(lineNo int, lineData string, sclk string, rtt int64, pmc int, fout io.StringWriter, pmcsAlreadyWritten map[int]bool) error {
	// Check line starts with a number followed by ": | ", otherwise it's not a scan log line we're interested in
	pos := strings.Index(lineData, ": | ")
	if pos < 0 {
		return nil // ignore, it's probably another scanlog line
	}

	tok, lineData, ok := takeToken(lineData, ": | ")
	if !ok {
		return fmt.Errorf("Error reading scan log start of line")
	}

	// If tok is a number, we're reading it!
	_ /*scanLogLine*/, err := strconv.Atoi(tok)
	if err != nil {
		return fmt.Errorf("Expected scanlog line to start with number, got: %v", tok)
	}

	// TODO: Check that scanLogLine is incrementing??
	// Now read the rest of the values
	fValues := []float32{}
	hexval := ""
	for c := 0; c < 6; c++ {
		var f float32
		f, lineData, err = readFloat(lineData)

		if err != nil {
			return fmt.Errorf("Failed to read scanlog float %v", c)
		}

		fValues = append(fValues, f)

		if c == 2 {
			// Expect a hex value, which we... seem to print as hex without 0x and cut off the last 2 bytes??
			tok, lineData, ok = takeToken(lineData, " ")
			if !ok || !strings.HasPrefix(tok, "0x") {
				return fmt.Errorf("Expected hex value")
			}

			hexval = tok

			// gobble up a |
			tok, lineData, ok = takeToken(lineData, " ")
			if !ok || tok != "|" {
				return fmt.Errorf("Expected separating |")
			}
		}
	}

	readPMC, _, err := readInt(lineData)
	if err != nil {
		return fmt.Errorf("Expected PMC at end ofline")
	}

	// DataDrive RSI format has table headers:
	// GV Report: GrandScan logged coordinates
	// SCLK,RTT,PMC,scan_x,scan_y,scan_z,word_1,word_2,word_3,GV_PMC,task_mask
	//
	// Also (only 1 line of) ???
	// Scan Log
	// SCLK,RTT,PMC,n_scan,n_log,x_center_BF,y_center_BF,z_center_BF,x_pivot,y_pivot,z_pivot,r_focus,x_scan,y_scan,z_scan,mask,x_corr,y_corr,z_corr

	// Output example:
	// TODO: is "3" at start a mistake??
	// TODO: Also PMC in the line doesn't match PMC of the line where scan log was written... seems to be -1?
	// TODO: what's the last value? always seems to be 0

	// 3???, C6F0202, 1657, 34, _Grand_Scan_Log, -0.136000, 0.131000, 0.246230, 0.000000, 0.000000, 0.000000, 1657, 0012, 0

	// If we've got duplicates, don't write again!
	if !pmcsAlreadyWritten[int(readPMC)] {
		_, err = fout.WriteString(fmt.Sprintf("3, %X, %v, 34, _Grand_Scan_Log, %.6f, %.6f, %.6f, %.6f, %.6f, %.6f, %v, %v, 0\n",
			/*makeWriteSCLK(sclk),*/ rtt, pmc, fValues[0], fValues[1], fValues[2], fValues[3], fValues[4], fValues[5], readPMC, hexval[2:6]))
		if err != nil {
			return fmt.Errorf("Failed to write to output CSV: %v", err)
		} else {
			pmcsAlreadyWritten[int(readPMC)] = true
		}
	}

	return nil
}
