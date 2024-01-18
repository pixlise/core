package quantification

import (
	"errors"
	"fmt"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/quantification/quantRunner"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/fileaccess"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func makePMCListFilesForQuantPMCs(
	svcs *services.APIServices,
	combinedSpectra bool,
	cfg config.APIConfig,
	datasetFileName string,
	jobDataPath string,
	quantStartSettings *protos.QuantStartingParameters,
	dataset *protos.Experiment) ([]string, int32, error) {
	pmcFiles := []string{}
	userParams := quantStartSettings.UserParams

	// Work out how many quants we're running, therefore how many nodes we need to generate in a reasonable time frame
	spectraCount := int32(len(userParams.Pmcs))
	if !combinedSpectra {
		spectraCount *= 2
	}

	nodeCount := quantRunner.EstimateNodeCount(spectraCount, int32(len(userParams.Elements)), int32(userParams.RunTimeSec), int32(quantStartSettings.CoresPerNode), cfg.MaxQuantNodes)

	if cfg.NodeCountOverride > 0 {
		nodeCount = cfg.NodeCountOverride
		svcs.Log.Infof("Using node count override: %v", nodeCount)
	}

	// NOTE: if we're running anything but the map command, the result is pretty quick, so we don't need to farm it out to multiple nodes
	if userParams.Command != "map" {
		nodeCount = 1
	}

	spectraPerNode := quantRunner.FilesPerNode(spectraCount, nodeCount)
	pmcsPerNode := spectraPerNode
	if !combinedSpectra {
		// If we're separate, we have 2x as many spectra as PMCs, so here we calculate how many
		// pmcs per node accurately for the next step to generate the right number of PMC lists
		pmcsPerNode /= 2
	}

	svcs.Log.Debugf("spectraPerNode: %v, PMCs per node: %v for %v spectra, nodes: %v", spectraPerNode, pmcsPerNode, spectraCount, nodeCount)

	// Generate the lists and save to S3
	pmcLists := makeQuantJobPMCLists(userParams.Pmcs, int(pmcsPerNode))

	pmcHasDwellLookup, err := makePMCHasDwellLookup(dataset)
	if err != nil {
		return []string{}, 0, err
	}

	for i, pmcList := range pmcLists {
		// Serialise the data for the list
		contents, err := makeIndividualPMCListFileContents(pmcList, datasetFileName, combinedSpectra, userParams.IncludeDwells, pmcHasDwellLookup)

		if err != nil {
			return pmcFiles, 0, fmt.Errorf("Error when preparing node PMC list: %v. Error: %v", i, err)
		}

		pmcListName, err := savePMCList(svcs, quantStartSettings.PiquantJobsBucket, contents, i, jobDataPath)
		if err != nil {
			return []string{}, 0, err
		}

		pmcFiles = append(pmcFiles, pmcListName)
	}

	return pmcFiles, spectraPerNode, nil
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

func makeQuantJobPMCLists(PMCs []int32, pmcsPerNode int) [][]int32 {
	var result [][]int32 = make([][]int32, 1)

	writeList := 0
	for c, PMC := range PMCs {
		if writeList >= len(result) {
			result = append(result, make([]int32, 0))
		}

		result[writeList] = append(result[writeList], PMC)

		if len(result[writeList]) > 0 && (len(result[writeList]) >= pmcsPerNode || c >= len(PMCs)) {
			writeList = writeList + 1
		}
	}

	return result
}

func combineQuantOutputs(fs fileaccess.FileAccess, jobsBucket string, jobPath string, header string, pmcFilesUsed []string) (string, error) {
	// Try to load each PMC file, if any fail, fail due to 1 node either not finishing/crashing/etc
	jobOutputPath := path.Join(jobPath, "output")

	var sb strings.Builder

	// Write header:
	sb.WriteString(header + "\n")

	pmcLineLookup := map[int][]string{}
	pmcs := []int{}

	for c, v := range pmcFilesUsed {
		// Make the assumed output path
		piquantOutputPath := path.Join(jobOutputPath, v+"_result.csv")

		data, err := fs.ReadObject(jobsBucket, piquantOutputPath)
		if err != nil {
			return "", errors.New("Failed to combine map segment: " + piquantOutputPath)
		}

		// Read all rows in. We want to sort these by PMC, so store the rows in map by PMC
		rows := strings.Split(string(data), "\n")

		// We have the data, append it to our output data
		dataStartRow := 2 // PIQUANT CSV outputs usually have 2 rows of header data...

		for i, row := range rows {
			// Ensure PMC is 1st column
			if i == 1 && !strings.HasPrefix(row, "PMC,") {
				return "", fmt.Errorf("Map segment: %v, did not have PMC as first column", piquantOutputPath)
			}

			// If we're reading the first file, output its headers to the output file
			if c <= 0 && i > 0 && i < dataStartRow {
				sb.WriteString(row + "\n")
			}

			// Normal rows: save to our map so we can sort them before writing
			if i >= dataStartRow && len(row) > 0 {
				pmcPos := strings.Index(row, ",")
				if pmcPos < 1 {
					return "", fmt.Errorf("Failed to combine map segment: %v, no PMC at line %v", piquantOutputPath, i+1)
				}

				pmcStr := row[0:pmcPos]
				pmc64, err := strconv.ParseInt(pmcStr, 10, 32)
				if err != nil {
					return "", fmt.Errorf("Failed to combine map segment: %v, invalid PMC %v at line %v", piquantOutputPath, pmcStr, i+1)
				}

				pmc := int(pmc64)
				if _, ok := pmcLineLookup[pmc]; !ok {
					// Add an array for this PMC
					pmcLineLookup[pmc] = []string{}

					// Also save in pmc list so it can be sorted
					pmcs = append(pmcs, pmc)
				}

				// add it to the list of lines for this row
				pmcLineLookup[pmc] = append(pmcLineLookup[pmc], row)
			}
		}
	}

	// Sort the PMCs and read from map into file
	sort.Ints(pmcs)

	// Read PMCs in order and write to file
	for _, pmc := range pmcs {
		rows, ok := pmcLineLookup[pmc]
		if !ok {
			return "", fmt.Errorf("Failed to save row for PMC: %v", pmc)
		}

		for _, row := range rows {
			sb.WriteString(row + "\n")
		}
	}

	return sb.String(), nil
}
