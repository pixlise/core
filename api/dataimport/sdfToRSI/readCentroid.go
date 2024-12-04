package sdfToRSI

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

func processCentroid(lineNo int, line string, lineData string, scanner *bufio.Scanner, currentDetector string, sclk string, rtt int64, pmc int, fout io.StringWriter) error {
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
		return fmt.Errorf("CenSLI_struct missing third line on line: %v, \"%v\"", lineNo, line)
	}

	line = scanner.Text()
	lineNo++

	id, res, err := processCentroidLine3(lineNo, line, sliNum)
	if err != nil {
		return err
	}

	// ONLY output if we have a detector already!
	if currentDetector != "A" && currentDetector != "B" {
		fmt.Printf("Skipping writing MCC SLI Estimates, detector unknown, on line: %v", lineNo)
		return nil
	}

	// DataDrive RSI format has table headers:
	// SLI Estimates
	// SCLK,RTT,PMC,SLI_A enabled,SLI_B enabled,pixel_x,pixel_y,intensity,x,y,z,ID,Residual

	// Example output:
	// 2AEE2547, C6F0202, 2, 57, MCC SLI Estimates B, 183.552124, 39.678978, 961.000000, -0.009290, -0.013930, 0.059620, 74.000000, 0.300000

	_, err = fout.WriteString(fmt.Sprintf("%v, %X, %v, 57, MCC SLI Estimates %v, %.6f, %.6f, %.6f, %.6f, %.6f, %.6f, %.6f, %.6f\n",
		makeWriteSCLK(sclk), rtt, pmc, currentDetector, pixX, pixY, intensity, x, y, z, float32(id), float32(res)/10))
	return err
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
	tokSep := []string{",", " "}
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
		tok, lineData, ok = takeToken(lineData, tokSep[c])

		if !ok {
			return 0, 0, fmt.Errorf("Failed to read %v on line: %v, \"%v\"", idToken, lineNo, line)
		}

		val, err := strconv.ParseInt(tok, 16, 32)
		if err != nil {
			return 0, 0, fmt.Errorf("Failed to read hex %v on line: %v, \"%v\"", idToken, lineNo, line)
		}

		vals = append(vals, val)
	}

	return vals[0], vals[1], nil
}
