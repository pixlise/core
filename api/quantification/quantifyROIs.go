package quantification

import (
	"errors"
	"fmt"
	"path"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/fileaccess"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func makePMCListFilesForQuantROI(
	hctx wsHelpers.HandlerContext,
	combinedSpectra bool,
	cfg config.APIConfig,
	datasetFileName string,
	jobDataPath string,
	params *protos.QuantStartingParameters,
	dataset *protos.Experiment,
) (string, int32, []roiItemWithPMCs, error) {
	// We're quantifying by ROIs, so we are actually adding all spectra in the ROI before quantifying once. First we need to download the ROIs
	// We will also need the dataset file so we can convert our roi LocIdx to PMCs
	locIdxToPMCLookup, err := makeLocToPMCLookup(dataset, true)
	if err != nil {
		return "", 0, []roiItemWithPMCs{}, err
	}

	rois, err := getROIs(params.UserParams.Command, params.UserParams.ScanId, params.UserParams.RoiIDs, hctx, locIdxToPMCLookup, dataset)
	if err != nil {
		return "", 0, rois, err
	}

	// Save list to file in S3 for piquant to pick up
	quantCount := int32(len(rois))
	if !combinedSpectra {
		quantCount *= 2
	}

	pmcHasDwellLookup, err := makePMCHasDwellLookup(dataset)
	if err != nil {
		return "", 0, rois, err
	}

	contents, err := makeROIPMCListFileContents(rois, datasetFileName, combinedSpectra, params.UserParams.IncludeDwells, pmcHasDwellLookup)
	if err != nil {
		return "", 0, rois, fmt.Errorf("Error when preparing quant ROI node list. Error: %v", err)
	}

	pmcListName, err := savePMCList(hctx.Svcs, params.PiquantJobsBucket, contents, 0, jobDataPath)
	if err != nil {
		return "", 0, rois, err
	}

	return pmcListName, quantCount, rois, nil
}

func makeROIPMCListFileContents(rois []roiItemWithPMCs, DatasetFileName string, combinedDetectors bool, includeDwells bool, pmcHasDwellLookup map[int32]bool) (string, error) {
	// Serialise the data for the list
	var sb strings.Builder
	sb.WriteString(DatasetFileName + "\n")

	for _, roi := range rois {
		sb.WriteString(fmt.Sprintf("%v:", roi.Id))

		if combinedDetectors {
			// We output all PMCs on one row, because we want to sum them all THEN quantify
			// 123|Normal|A,123|Normal|B,124|Normal|A,124|Normal|B
			for c, pmc := range roi.PMCs {
				divider := ""
				if c > 0 {
					divider = ","
				}
				sb.WriteString(fmt.Sprintf("%v%v|Normal|A,%v|Normal|B", divider, pmc, pmc))
				if includeDwells && pmcHasDwellLookup[int32(pmc)] {
					sb.WriteString(fmt.Sprintf(",%v|Dwell|A,%v|Dwell|B", pmc, pmc))
				}
			}
			sb.WriteString("\n")
		} else {
			// We output all PMCs on one row, but A then B rows, because we want to sum them all (per detector) THEN quantify
			// 123|Normal|A,124|Normal|A
			// 123|Normal|A,124|Normal|B
			detectors := []string{"A", "B"}
			for detIdx, det := range detectors {
				for c, pmc := range roi.PMCs {
					divider := ""
					if c > 0 {
						divider = ","
					}
					sb.WriteString(fmt.Sprintf("%v%v|Normal|%v", divider, pmc, det))
					if includeDwells && pmcHasDwellLookup[int32(pmc)] {
						sb.WriteString(fmt.Sprintf(",%v|Dwell|%v", pmc, det))
					}
				}
				sb.WriteString("\n")

				if detIdx < len(detectors)-1 {
					sb.WriteString(fmt.Sprintf("%v:", roi.Id))
				}
			}
		}
	}

	return sb.String(), nil
}

func processQuantROIsToPMCs(fs fileaccess.FileAccess, jobsBucket string, jobPath string, header string, piquantCSVFile string, combinedQuant bool, rois []roiItemWithPMCs) (string, error) {
	// PIQUANT has summed then quantified the spectra belonging to PMCs in each ROI. We now have to take those rows
	// and copy them so each PMC in the ROI has a copy of the quantification row.
	jobOutputPath := path.Join(jobPath, "output")

	var sb strings.Builder

	// Write header:
	sb.WriteString(header + "\n")

	roiIdxToLineLookup := make([][]string, len(rois), len(rois))

	// Read in the piquant generated output that we're going to process
	// Make the assumed output path
	piquantOutputPath := path.Join(jobOutputPath, piquantCSVFile+"_result.csv")

	data, err := fs.ReadObject(jobsBucket, piquantOutputPath)
	if err != nil {
		return "", errors.New("Failed to read map CSV: " + piquantOutputPath)
	}

	// Read all rows in. We want to sort these by PMC, so store the rows in map by PMC
	rows := strings.Split(string(data), "\n")

	// We have the data, append it to our output data
	dataStartRow := 2 // PIQUANT CSV outputs usually have 2 rows of header data...
	fileNameColIdx := -1
	colCount := 0

	for i, row := range rows {
		// Ignore first row
		if i == 0 {
			continue
		}

		// Ensure PMC is 1st column
		if i == 1 {
			cols := strings.Split(row, ",")
			colCount = len(cols) // save for later
			for colIdx, col := range cols {
				colClean := strings.Trim(col, " \t")
				if colClean == "filename" {
					fileNameColIdx = colIdx
					break
				}
			}

			if fileNameColIdx < 0 {
				return "", fmt.Errorf("Map csv: %v, does not contain a filename column (used to match up ROIs)", piquantOutputPath)
			}
		}

		// Copy the header row
		if i < dataStartRow {
			sb.WriteString(row + "\n")
		} else {
			if len(row) > 0 {
				// Read the file name column and work out the ROI ID
				values := strings.Split(row, ",")

				// Verify we have the right amount
				if len(values) != colCount {
					return "", fmt.Errorf("Unexpected column count on map CSV: %v, line %v", piquantOutputPath, i+1)
				}

				fileName := strings.Trim(values[fileNameColIdx], " \t")

				// We expect file names of the form:
				// Normal_A_roiid
				// or Normal_Combined_roiid
				// This way we can confirm we're reading what we expect, and we know which roi to match to
				fileNameBits := strings.Split(fileName, "_")
				if len(fileNameBits) != 3 || fileNameBits[0] != "Normal" || (fileNameBits[1] != "Combined" && fileNameBits[1] != "A" && fileNameBits[1] != "B") || len(fileNameBits[2]) <= 0 {
					return "", fmt.Errorf("Invalid file name read: %v from map CSV: %v, line %v", fileName, piquantOutputPath, i+1)
				}

				// Work out the index of the ROI this applies to
				roiIdx := -1
				for idx, roi := range rois {
					if roi.Id == fileNameBits[2] {
						roiIdx = idx
						break
					}
				}

				// Make sure we found it...
				if roiIdx < 0 {
					return "", fmt.Errorf("CSV contained unexpected roi: \"%v\" when processing map CSV: %v", fileNameBits[2], piquantOutputPath)
				}

				// Also parse & validate PMC so we can read the rest of the row after it!
				pmcPos := strings.Index(row, ",")
				if pmcPos < 1 {
					return "", fmt.Errorf("Failed to process map CSV: %v, no PMC at line %v", piquantOutputPath, i+1)
				}

				pmcStr := row[0:pmcPos]
				pmc64, err := strconv.ParseInt(pmcStr, 10, 32)
				if err != nil {
					return "", fmt.Errorf("Failed to process map CSV: %v, invalid PMC %v at line %v", piquantOutputPath, pmcStr, i+1)
				}

				// Add line to the lookup
				roiIdxToLineLookup[roiIdx] = append(roiIdxToLineLookup[roiIdx], row[pmcPos:])

				// Sanity check: Verify that the PMC read exists in the ROI we think we're reading for
				pmc := int(pmc64)

				pmcFound := false
				for _, roiPMC := range rois[roiIdx].PMCs {
					if roiPMC == pmc {
						pmcFound = true
						break
					}
				}

				if !pmcFound {
					return "", fmt.Errorf("PMC %v in CSV: %v doesn't exist in ROI: %v", pmcStr, piquantOutputPath, rois[roiIdx].Name)
				}
			}
		}
	}

	// Now run through ROIs and write out line copies for each PMC
	for c, roi := range rois {
		for _, pmc := range roi.PMCs {
			for _, row := range roiIdxToLineLookup[c] {
				sb.WriteString(fmt.Sprintf("%v%v\n", pmc, row))
			}
		}
	}

	return sb.String(), nil
}
