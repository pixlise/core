package wsHandler

import (
	"errors"
	"fmt"
	"strings"

	"github.com/pixlise/core/v3/api/quantification"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
)

// Users can also upload a compatible CSV file which we can convert into a quantification that's usable inside PIXLISE
// We expect the name, comments and csvData fields to be set. CSV format is as follows:
// <csv title line>
// <csv column headers>
// <csv row 0>
// ...
// <csv row n>

func HandleQuantUploadReq(req *protos.QuantUploadReq, hctx wsHelpers.HandlerContext) (*protos.QuantUploadResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 50); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Comments, "Comments", 1, wsHelpers.DescriptionFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.CsvData, "CsvData", 1, 10*1024*1024); err != nil {
		return nil, err
	}

	csvRows := strings.Split(req.CsvData, "\n")
	colLookup, err := parseQuantCSVColumns(csvRows)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	quantMode := quantification.QuantModeCombinedManualUpload

	// We know the filename column exists due to parseCSVColumns above
	if isABQuant(csvRows, colLookup["filename"]) {
		quantMode = quantification.QuantModeABManualUpload
	}

	quantId, err := quantification.ImportQuantCSV(hctx, req.ScanId, hctx.SessUser.User, req.CsvData, "user-supplied", "upload", req.Name, quantMode, req.Comments)
	if err != nil {
		return nil, err
	}
	return &protos.QuantUploadResp{CreatedQuantId: quantId}, nil
}

func parseQuantCSVColumns(csvRows []string) (map[string]int, error) {
	colMap := map[string]int{}

	if len(csvRows) <= 2 {
		return map[string]int{}, errors.New("CSV must contain more than 2 lines")
	}

	// Expect certain columns
	cols := strings.Split(csvRows[1], ",")

	// Build a map so it's easier to look up

	hasWeightCol := false
	for c, col := range cols {
		colClean := strings.Trim(col, " \t")
		colMap[colClean] = c

		if strings.HasSuffix(colClean, "_%") {
			hasWeightCol = true
		}
	}

	if !hasWeightCol {
		return map[string]int{}, errors.New("CSV did not contain any _% columns")
	}

	// An example of valid:
	// PMC, CaO_%, SiO2_%, FeO-T_%, CaO_int, SiO2_int, FeO-T_int, CaO_err, SiO2_err, FeO-T_err, total_counts, livetime, chisq, eVstart, eV/ch, res, iter, filename, Events, Triggers, SCLK, RTT
	// We require AT LEAST:
	reqCols := []string{"PMC", "livetime", "filename", "SCLK", "RTT"} // and one _% column
	for _, col := range reqCols {
		if _, ok := colMap[col]; !ok {
			return map[string]int{}, fmt.Errorf("CSV missing column: \"%v\"", col)
		}
	}

	return colMap, nil
}

func isABQuant(csvRows []string, filenameColumnIdx int) bool {
	if len(csvRows) < 3 {
		return false
	}

	// Check near first, middle and near-last rows to see if we find A and B detectors
	earlyRow := strings.Split(csvRows[2], ",")
	earlyIsCombined := false

	midRow := strings.Split(csvRows[(2+len(csvRows)-2)/2], ",")
	midIsCombined := false

	lastRow := strings.Split(csvRows[len(csvRows)-1], ",")
	lastIsCombined := false

	if len(earlyRow) > filenameColumnIdx {
		if strings.HasSuffix(earlyRow[filenameColumnIdx], "_Combined") {
			earlyIsCombined = true
		}
	}

	if len(midRow) > filenameColumnIdx {
		if strings.HasSuffix(midRow[filenameColumnIdx], "_Combined") {
			midIsCombined = true
		}
	}

	if len(lastRow) > filenameColumnIdx {
		if strings.HasSuffix(lastRow[filenameColumnIdx], "_Combined") {
			lastIsCombined = true
		}
	}

	return !earlyIsCombined && !midIsCombined && !lastIsCombined
}
