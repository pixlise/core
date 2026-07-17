package expressionrunner

import (
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/client"
)

// var DiffractionStatusUnspecified = "unspecified"
var DiffractionStatusNotAnomaly = "not-anomaly"

func (e *expressionRunner) ensureFetchedDiffraction() error {
	if e.diffractionFile == nil {
		if err := e.fetchDiffraction(); err != nil || e.diffractionFile == nil {
			e.Log().Errorf("Expression runner could not fetch diffraction: %v", err)
			return err
		}
	}

	return nil
}

func (e *expressionRunner) fetchDiffraction() error {
	// Gather all the stuff we need
	dataset, err := wsHelpers.ReadDatasetFile(e.scanId, e.svcs, true)
	if err != nil {
		return err
	}

	e.diffractionFile, err = wsHelpers.ReadDiffractionFile(e.scanId, e.svcs)
	if err != nil {
		return err
	}

	// Simulate getting the stuff via the 3 API calls the client uses(/used)
	requestedIndexes := []int32{}
	detectedPeaks, err := wsHelpers.GetDetectedDiffractionPeaks(requestedIndexes, dataset, e.diffractionFile)
	if err != nil {
		return err
	}

	e.manualPeaks, err = wsHelpers.GetDiffractionPeakManualList(e.scanId, e.svcs)
	if err != nil {
		return err
	}

	detectedPeakStatuses, err := wsHelpers.GetDiffractionPeakStatusList(e.scanId, e.svcs)
	if err != nil {
		return err
	}

	e.allPeaks, e.roughnessItems = client.ReadDiffractionData(detectedPeaks, detectedPeakStatuses, e.quantEVCalibration)
	return nil
}
