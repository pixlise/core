package sdfToRSI

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"
)

func readAheadLines(scanner *bufio.Scanner, lineNo int, lineCount int) ([]string, error) {
	lines := []string{}
	for c := 0; c < lineCount; c++ {
		if !scanner.Scan() {
			return []string{}, fmt.Errorf("Failed while reading ahead %v lines (from line %v) at line %v", lineCount, lineNo, lineNo+c+1)
		}

		lines = append(lines, scanner.Text())
	}

	return lines, nil
}

func checkMCCDetector(line string) (string, bool, error) {
	// Expecting: "00384 : 320A0896  14320032  0A0000C8  32640002  01732A13  FFFF2FFF  00000000  FFFF0000 "
	// We confirm it starts with 00384, then check for the 5th word, chars 4,5
	id, lineData, ok := takeToken(line, ":")
	if ok && id == "00384 " {
		parts := strings.Split(lineData, " ")

		// Only keep ones that have valid stuff
		wordParts := []string{}
		for _, part := range parts {
			if len(part) == 8 {
				wordParts = append(wordParts, part)
			} else if len(part) > 0 {
				return "", true, fmt.Errorf("Read invalid word: %v", part)
			}
		}

		if len(wordParts) != 8 {
			return "", true, errors.New("Failed to read detector config")
		}

		// Read the right one
		det := wordParts[4][4:6]
		if det == "00" {
			return "0", true, nil
		} else if det == "25" {
			return "A", true, nil
		} else if det == "2A" {
			return "B", true, nil
		}
		return "", true, fmt.Errorf("Invalid detector: %v", det)
	}

	return "", false, nil
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
	line = strings.TrimLeft(line, sep)
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
		// Prints: Error: strconv.ParseFloat: parsing "2.L3": invalid syntax
		return 0, remainder, fmt.Errorf("Error: %v", err)
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

const (
	read_int     = iota
	read_int_hex = iota
	read_float   = iota
)

// For example, reading a value marked with VVV "Reference: 0xVVV,"
func readNumBetween(line string, prefix string, suffix string, readType int) (int64, float32, int, error) {
	pos := strings.Index(line, prefix)
	if pos < 0 {
		return 0, 0, 0, fmt.Errorf("failed to find value after %v", prefix)
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
		return 0, 0, 0, fmt.Errorf("failed to read suffix '%v' after '%v'", suffix, prefix)
	}
	lastPos += len(tok)

	if readType == read_float {
		val, err := strconv.ParseFloat(tok, 32)
		if err != nil {
			return 0, 0, 0, fmt.Errorf("failed to read float value after '%v'", prefix)
		}

		return 0, float32(val), lastPos, nil
	}

	radix := 10
	if readType == read_int_hex {
		radix = 16
	}

	val, err := strconv.ParseInt(tok, radix, 32)
	if err != nil {
		return 0, 0, 0, fmt.Errorf("failed to read int value after '%v'", prefix)
	}

	return val, 0, lastPos, nil
}

// Expected SCLK format: 2006-002T15:04:05
func makeWriteSCLInt(readSCLK string) int64 {
	unixSec := readTimestamp(readSCLK)

	// Work out where we are relative to the epoch above, and add to it to work out SCLK to use
	secSinceEpoch := unixSec - 1666967479

	sclk := 720239544 + secSinceEpoch

	return sclk
}

func makeWriteSCLK(readSCLK string) string {
	sclk := makeWriteSCLInt(readSCLK)

	// Return as hex
	return fmt.Sprintf("%X", sclk)
}

func readTimestamp(ts string) int64 {
	// Read the SCLK value as a string time, expecting eg 2022-301T14:31:18
	t, err := time.Parse("2006-002T15:04:05", ts)
	if err != nil {
		log.Fatalf("Failed to parse read SCLK: %v. Error: %v", ts, err)
	}

	// Bit of a hack/approximation. We assume that SCLK is in seconds, and our reference is:
	// 2022-301T14:31:19 ==  Friday, October 28, 2022 14:31:19 == 1666967479 == 0x2AEDFBB8 == 720239544
	return t.Unix()
}
