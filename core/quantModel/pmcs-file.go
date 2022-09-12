// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package quantModel

import (
	"fmt"
	"path"
	"strings"

	"github.com/pixlise/core/v2/api/services"
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
	PMCsPerNode := int(spectraPerNode)

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
