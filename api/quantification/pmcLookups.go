package quantification

import (
	"fmt"
	"strconv"

	protos "github.com/pixlise/core/v3/generated-protos"
)

func getDetectorMetaValue(metaLabel string, detector *protos.Experiment_Location_DetectorSpectrum, dataset *protos.Experiment) (protos.Experiment_MetaDataType, *protos.Experiment_Location_MetaDataItem, error) {
	// Look up the label
	c := int32(0)
	found := false
	for _, label := range dataset.MetaLabels {
		if label == metaLabel {
			found = true
			break
		}
		c++
	}

	if !found {
		return protos.Experiment_MT_FLOAT, nil, fmt.Errorf("Failed to find detector meta value: %v", metaLabel)
	}

	typeStored := dataset.MetaTypes[c]

	for _, metaValue := range detector.Meta {
		if metaValue.LabelIdx == c {
			return typeStored, metaValue, nil
		}
	}

	return protos.Experiment_MT_FLOAT, nil, fmt.Errorf("Failed to read detector meta value %v for type: %v", metaLabel, typeStored)
}

func makePMCHasDwellLookup(dataset *protos.Experiment) (map[int32]bool, error) {
	lookup := map[int32]bool{}

	for locIdx, loc := range dataset.Locations {
		pmcI, err := strconv.Atoi(loc.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to read location PMC: %v, idx: %v. Error: %v", loc.Id, locIdx, err)
		}

		// Look up if this PMC has dwell spectra, if so, add to lookup
		for _, det := range loc.Detectors {
			// Get the read type, telling us if it has normal or dwell
			metaType, metaVar, err := getDetectorMetaValue("READTYPE", det, dataset)

			// We may fail to read some stuff, there may be no spectrum or metadata in this PMC, that's OK
			if err == nil && metaType == protos.Experiment_MT_STRING && metaVar.Svalue == "Dwell" {
				lookup[int32(pmcI)] = true
				break
			}
		}
	}

	return lookup, nil
}

func makeLocToPMCLookup(dataset *protos.Experiment, onlyWithNormalOrDwellSpectra bool) (map[int32]int32, error) {
	locIdxToPMCLookup := map[int32]int32{}

	for locIdx, loc := range dataset.Locations {
		pmcI, err := strconv.Atoi(loc.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to read location PMC: %v, idx: %v. Error: %v", loc.Id, locIdx, err)
		}

		// If we want to only filter to PMCs that have a spectrum defined, do the check here
		if onlyWithNormalOrDwellSpectra {
			foundSpectrum := false
			for _, det := range loc.Detectors {
				// Check if we get a normal or dwell
				metaType, metaVar, err := getDetectorMetaValue("READTYPE", det, dataset)

				// We may fail to read some stuff, there may be no spectrum or metadata in this PMC, that's OK
				if err == nil && metaType == protos.Experiment_MT_STRING && (metaVar.Svalue == "Normal" || metaVar.Svalue == "Dwell") {
					foundSpectrum = true
					break
				}
			}

			if !foundSpectrum {
				// We want ONLY PMCs that have a normal or dwell spectra, did not find one, so skip saving this in the lookup
				continue
			}
		}

		locIdxToPMCLookup[int32(locIdx)] = int32(pmcI)
	}

	return locIdxToPMCLookup, nil
}
