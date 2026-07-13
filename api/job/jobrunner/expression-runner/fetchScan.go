package expressionrunner

import "github.com/pixlise/core/v4/api/ws/wsHelpers"

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
	dataset, err := wsHelpers.ReadDatasetFile(e.scanId, e.svcs, true)
	if err != nil {
		return err
	}

	e.scan = dataset
	return nil
}
