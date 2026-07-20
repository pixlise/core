package expressionrunner

import (
	"fmt"
	"strconv"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/client"
)

func (e *expressionRunner) ensureFetchedScan() error {
	if e.scan == nil {
		if err := e.fetchScan(); err != nil || e.scan == nil {
			e.Log().Errorf("Expression runner could not fetch scan: %v", err)
			return err
		}
	}

	return nil
}

func (e *expressionRunner) fetchScan() error {
	// Gather all the stuff we need
	var err error
	e.scan, err = wsHelpers.ReadDatasetFile(e.scanId, e.svcs, true)
	return err
}

func (e *expressionRunner) ensureFetchedScanNormalSpectra() error {
	err := e.ensureFetchedScan()
	if err != nil {
		return err
	}

	// If we've got our lookups already, stop here
	if e.detectorIdMetaIdx > -1 && e.readTypeMetaIdx > -1 &&
		len(e.pmcToLocationIndex) > 0 && len(e.locationIndexToPMC) > 0 &&
		len(e.spectra) > 0 {
		return nil
	}

	// Now we run through the scan and fetch stuff that will make later lookups quicker

	// Get indexes and data types for our meta fields used to identify the spectrum
	e.detectorIdMetaIdx, e.readTypeMetaIdx, err = e.getDetectorReadTypeMetaIdxs()
	if err != nil {
		return err
	}

	e.pmcToLocationIndex = map[int]uint32{}
	e.locationIndexToPMC = []int{}
	e.spectra = map[string][][]int32{}

	// Read and decode spectra so they are in memory for quick lookups going forward
	metaIdxs := []int{e.detectorIdMetaIdx, e.readTypeMetaIdx}
	for c, loc := range e.scan.Locations {
		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return fmt.Errorf("Failed to read PMC: \"%v\" for scan: %v", loc.Id, e.scanId)
		}

		foundNormal := false
		for _, det := range loc.Detectors {
			metaValues := getScanDetectorMetaValues(metaIdxs, det)

			// Check if we have values
			if m, ok := metaValues[e.detectorIdMetaIdx]; ok {
				// We read a detector id! act on it
				detectorId := m.Svalue

				// Make sure we have spectra storage allocated for this detector
				if len(e.spectra[detectorId]) <= 0 {
					e.spectra[detectorId] = [][]int32{}
				}

				// Is this a normal spectrum?
				if m2, ok := metaValues[e.readTypeMetaIdx]; ok && m2.Svalue == "Normal" {
					foundNormal = true

					// Store this spectrum
					spectrum := client.ZeroRunDecode(det.Spectrum)

					e.spectrumCounts = len(spectrum)
					e.spectra[detectorId] = append(e.spectra[detectorId], spectrum)
				}
			}
		}

		if foundNormal {
			// Store in the lookups
			e.pmcToLocationIndex[pmc] = uint32(c)
			e.locationIndexToPMC = append(e.locationIndexToPMC, pmc)
		}
	}

	return nil
}
