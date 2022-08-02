// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package dataset

import (
	"fmt"
	"strconv"

	protos "github.com/pixlise/core/generated-protos"
)

func MakePMCBeamLookup(dataset *protos.Experiment) map[int32]protos.Experiment_Location_BeamLocation {
	// We use the dataset to look up PMC locations
	pmcBeamLocLookup := map[int32]protos.Experiment_Location_BeamLocation{}
	for _, loc := range dataset.Locations {
		pmc, err := strconv.Atoi(loc.Id)
		if err == nil && loc.Beam != nil {
			pmcBeamLocLookup[int32(pmc)] = *loc.Beam
		}
	}
	return pmcBeamLocLookup
}

// Maybe the name is not that clear, creates a lookup for PMC->index in beam ij array, eg if PMC6 has a context image
// and it's the first set of coordinates in the beam ij's, 6 will map to 0
func MakePMCBeamIndexLookup(dataset *protos.Experiment) map[int32]int32 {
	pmcBeamIndexLookup := map[int32]int32{}

	// NOTE: the FIRST one refers to the i/j's that are in the Beam structure (legacy ones)
	//       while subsequent ones refer to the i/j's in the array, so the first index we
	//       store is -1, which means to look at the coord not in the array
	for idx, img := range dataset.AlignedContextImages {
		pmcBeamIndexLookup[img.Pmc] = int32(idx - 1)
	}

	return pmcBeamIndexLookup
}

func MakeLocToPMCLookup(dataset *protos.Experiment, onlyWithNormalOrDwellSpectra bool) (map[int32]int32, error) {
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
				metaType, metaVar, err := GetDetectorMetaValue("READTYPE", det, dataset)

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

func MakePMCHasDwellLookup(dataset *protos.Experiment) (map[int32]bool, error) {
	lookup := map[int32]bool{}

	for locIdx, loc := range dataset.Locations {
		pmcI, err := strconv.Atoi(loc.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to read location PMC: %v, idx: %v. Error: %v", loc.Id, locIdx, err)
		}

		// Look up if this PMC has dwell spectra, if so, add to lookup
		for _, det := range loc.Detectors {
			// Get the read type, telling us if it has normal or dwell
			metaType, metaVar, err := GetDetectorMetaValue("READTYPE", det, dataset)

			// We may fail to read some stuff, there may be no spectrum or metadata in this PMC, that's OK
			if err == nil && metaType == protos.Experiment_MT_STRING && metaVar.Svalue == "Dwell" {
				lookup[int32(pmcI)] = true
				break
			}
		}
	}

	return lookup, nil
}

func GetPMCsForLocationIndexes(locationIndexes []int32, dataset *protos.Experiment) ([]int, error) {
	// Get a lookup for all of them...
	lookup, err := MakeLocToPMCLookup(dataset, false)
	pmcs := []int{}
	errorCount := 0

	if err != nil {
		return pmcs, err
	}

	for _, idx := range locationIndexes {
		if pmc, ok := lookup[idx]; ok {
			pmcs = append(pmcs, int(pmc))
		} else {
			errorCount++
		}
	}

	if errorCount > 0 {
		return []int{}, fmt.Errorf("Failed to get %v ROI PMCs", errorCount)
	}

	return pmcs, nil
}
