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

package quantModel

import (
	"fmt"
	"path"
	"strings"

	"github.com/pixlise/core/api/services"
)

func savePMCList(svcs *services.APIServices, jobBucket string, contents string, nodeNumber int, jobDataPath string) (string, error) {
	pmcListName := fmt.Sprintf("node%05d.pmcs", nodeNumber)
	savePath := path.Join(jobDataPath, pmcListName)

	err := svcs.FS.WriteObject(jobBucket, savePath, []byte(contents))
	if err != nil {
		// Couldn't save it, no point continuing, we don't want a quantification with a section missing!
		return pmcListName, fmt.Errorf("Error when writing node PMC list: %v. Error: %v", savePath, err)
	}

	return pmcListName, nil
}

func makeQuantJobPMCLists(PMCs []int32, spectraPerNode int32) [][]int32 {
	var result [][]int32 = make([][]int32, 1)
	PMCsPerNode := int(spectraPerNode / 2) // Assuming A & B spectra per node

	writeList := 0
	for c, PMC := range PMCs {
		if writeList >= len(result) {
			result = append(result, make([]int32, 0))
		}

		result[writeList] = append(result[writeList], PMC)

		if len(result[writeList]) > 0 && (len(result[writeList]) >= PMCsPerNode || c >= len(PMCs)) {
			writeList = writeList + 1
		}
	}

	return result
}

func makeROIPMCListFileContents(rois []ROIWithPMCs, DatasetFileName string, combinedDetectors bool, includeDwells bool, pmcHasDwellLookup map[int32]bool) (string, error) {
	// Serialise the data for the list
	var sb strings.Builder
	sb.WriteString(DatasetFileName + "\n")

	for _, roi := range rois {
		sb.WriteString(fmt.Sprintf("%v:", roi.ID))

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
					sb.WriteString(fmt.Sprintf("%v:", roi.ID))
				}
			}
		}
	}

	return sb.String(), nil
}

func makeIndividualPMCListFileContents(PMCs []int32, DatasetFileName string, combinedDetectors bool, includeDwells bool, pmcHasDwellLookup map[int32]bool) (string, error) {
	// Serialise the data for the list
	var sb strings.Builder
	sb.WriteString(DatasetFileName + "\n")

	if combinedDetectors {
		// We're outputting rows of the form:
		// 123|Normal|A,123|Normal|B
		// In future, if we want to combine Dwells, multiple PMCs or control A & B quantification
		// separately, we'll need more parameters to this function!
		for _, pmc := range PMCs {
			sb.WriteString(fmt.Sprintf("%v|Normal|A,%v|Normal|B", pmc, pmc))
			if includeDwells && pmcHasDwellLookup[pmc] {
				sb.WriteString(fmt.Sprintf(",%v|Dwell|A,%v|Dwell|B", pmc, pmc))
			}
			sb.WriteString("\n")
		}
	} else {
		// We're outputting rows of the form:
		// 123|Normal|A
		// 123|Normal|B
		// To produce separate A and B quantifications
		for _, pmc := range PMCs {
			sb.WriteString(fmt.Sprintf("%v|Normal|A", pmc))
			if includeDwells && pmcHasDwellLookup[pmc] {
				sb.WriteString(fmt.Sprintf(",%v|Dwell|A", pmc))
			}
			sb.WriteString("\n")

			sb.WriteString(fmt.Sprintf("%v|Normal|B", pmc))
			if includeDwells && pmcHasDwellLookup[pmc] {
				sb.WriteString(fmt.Sprintf(",%v|Dwell|B", pmc))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String(), nil
}
