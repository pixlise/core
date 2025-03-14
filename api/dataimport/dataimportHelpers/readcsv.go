package dataImportHelpers

import (
	"bufio"
	"encoding/csv"
	"fmt"
	"io"
	"os"

	"github.com/pixlise/core/v4/core/logger"
)

func ReadCSV(filePath string, headerIdx int, sep rune, jobLog logger.ILogger) ([][]string, error) {
	csvFile, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}

	seekPos := int64(0)
	if headerIdx > 0 {
		n := 0
		for n < headerIdx {
			n = n + 1
			row1, err := bufio.NewReader(csvFile).ReadSlice('\n')
			if err != nil {
				return nil, err
			}
			seekPos += int64(len(row1))
			_, err = csvFile.Seek(seekPos, io.SeekStart)
			if err != nil {
				return nil, err
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
	var lineRecord []string
	for {
		lineRecord, err = r.Read()
		if err == io.EOF {
			break
		}

		if err != nil {
			if csverr, ok := err.(*csv.ParseError); !ok && csverr.Err != csv.ErrFieldCount {
				return nil, err
			}
		}

		rows = append(rows, lineRecord)
	}

	if len(rows) <= 0 {
		return rows, fmt.Errorf("Read 0 rows from: %v", filePath)
	}
	return rows, nil
}
