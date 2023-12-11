package quantification

import (
	"errors"

	protos "github.com/pixlise/core/v3/generated-protos"
)

func calculateTotals[T int | int32 | uint32](quantFile *protos.Quantification, roiPMCs []T) (map[string]float32, error) {
	columns := []string{}
	totals := map[string]float32{}

	// Ensure we're dealing with a Combined quant
	if len(quantFile.LocationSet) != 1 || quantFile.LocationSet[0].Detector != "Combined" {
		return totals, errors.New("Quantification must be for Combined detectors")
	}

	// Make a quick lookup for PMCs
	roiPMCSet := map[int]bool{} // REFACTOR: TODO: Make generic version of utils.SetStringsInMap() for this
	for _, pmc := range roiPMCs {
		roiPMCSet[int(pmc)] = true
	}

	// Decide on columns we're working with
	columns = getWeightPercentColumnsInQuant(quantFile)
	columnIdxs := []int{}
	for _, col := range columns {
		columnIdxs = append(columnIdxs, int(getQuantColumnIndex(quantFile, col)))
	}

	if len(columns) <= 0 {
		return totals, errors.New("Quantification has no weight %% columns")
	}

	// Read the combined locations for the PMCs in the ROI
	foundPMCCount := 0
	for _, loc := range quantFile.LocationSet[0].Location {
		// If PMC is in ROI
		if roiPMCSet[int(loc.Pmc)] {
			// Run through all columns and add them to our totals
			for c := 0; c < len(columnIdxs); c++ {
				column := columns[c]
				value := loc.Values[columnIdxs[c]]

				totals[column] += value.Fvalue
			}

			foundPMCCount++
		}
	}

	// Here we finally turn these into averages
	result := map[string]float32{}
	for k, v := range totals {
		if foundPMCCount > 0 {
			result[k] = v / float32(foundPMCCount)
		}
	}

	if len(result) <= 0 {
		return result, errors.New("Quantification had no valid data for ROI PMCs")
	}

	return result, nil
}
