package sdfToRSI

import (
	"fmt"
	"io"
)

func processGV(lineNo int, line string, lineData string, sclk string, rtt int64, pmc int, fout io.StringWriter) error {
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

	// DataDrive RSI format has table headers:
	// GV Report: Base Frame SLI spot coordinates
	// SCLK,RTT,PMC,SLI_x,SLI_y,SLI_z

	// Output example:
	// 2AEE3F04, C6F0202, 404, 5, _MCC_SLI_SpotList_BF, -0.136560, 0.137250, 0.246220

	_, err = fout.WriteString(fmt.Sprintf("%v, %X, %v, 5, _MCC_SLI_SpotList_BF, %.6f, %.6f, %.6f\n",
		makeWriteSCLK(sclk), rtt, pmc, vals[0], vals[1], vals[2]))
	if err != nil {
		return fmt.Errorf("Failed to write to output CSV: %v", err)
	}

	return nil
}
