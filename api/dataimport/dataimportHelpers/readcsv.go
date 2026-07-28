package dataImportHelpers

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
)

func ReadCSV(filePath string, headerIdx int, sep rune) ([][]string, error) {
	csvFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	fields, err := ReadCSVData(csvFile, headerIdx, sep)
	if err != nil {
		return fields, fmt.Errorf("ReadCSV \"%v\": %v", filePath, err)
	}

	return fields, nil
}

func ReadCSVData(csvFile io.Reader, headerIdx int, sep rune) ([][]string, error) {
	// Gobble up this many lines at the start...
	for c := 0; c < headerIdx; c++ {
		b := []byte{0}
		for {
			n, err := csvFile.Read(b)
			if err != nil {
				return [][]string{}, fmt.Errorf("Failed to skip CSV header lines: %v", err)
			}

			if n != 1 {
				return [][]string{}, fmt.Errorf("Failed read byte from CSV header line: %v", c+1)
			}

			if b[0] == '\n' {
				break
			}
		}
	}

	r := csv.NewReader(csvFile)
	r.TrimLeadingSpace = true
	r.Comma = sep

	// Some of our CSV files contain multiple tables, that we detect during parsing, so instead of using
	// ReadAll() here, which blows up when the # cols differs, we read each line, and if we get the error
	// "wrong number of fields", we can ignore it and keep reading
	rows := [][]string{}
	var err error
	var lineRecord []string
	for {
		lineRecord, err = r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			if csverr, ok := err.(*csv.ParseError); !ok && (csverr == nil || csverr.Err != csv.ErrFieldCount) {
				return nil, err
			}
		}

		rows = append(rows, lineRecord)
	}

	if len(rows) <= 0 {
		return rows, fmt.Errorf("Read 0 CSV rows")
	}
	return rows, nil
}
