package expressionrunner

import (
	"context"
	"fmt"
	"path"
	"slices"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var DefaultDetectorId = "Default"
var EnergyCalibrationDetector = "A"

func (e *expressionRunner) ensureFetchedQuant() error {
	if e.quantData == nil {
		if err := e.fetchQuant(); err != nil || e.quantData == nil {
			e.Log().Errorf("Expression runner could not fetch quant: %v", err)
			return err
		}
	}

	return nil
}

func (e *expressionRunner) fetchQuant() error {
	// Read quant from DB
	ctx := context.TODO()
	coll := e.minimalSvcs.MongoDB.Collection(dbCollections.QuantificationsName)

	filter := bson.M{"_id": e.quantId}
	opts := options.FindOne().SetProjection(bson.M{"status.outputfilepath": true, "_id": true})

	result := coll.FindOne(ctx, filter, opts)
	if result.Err() != nil {
		return result.Err()
	}

	quantSummary := &protos.QuantificationSummary{}
	if err := result.Decode(quantSummary); err != nil {
		return err
	}

	quantPath := path.Join(quantSummary.Status.OutputFilePath, e.quantId+".bin")
	quantData, err := wsHelpers.ReadQuantificationFile(e.quantId, quantPath, e.minimalSvcs)
	if err != nil {
		return err
	}

	e.quantData = quantData

	// Pre-compute some lookup stuff
	e.pureElementColumnLookup, e.elementColumns = buildPureElementLookup(quantData.Labels)

	// Find average calibration
	evStartPos := slices.Index(quantData.Labels, "eVstart")
	evPerChanPos := slices.Index(quantData.Labels, "eV/ch")

	if evStartPos == -1 {
		return fmt.Errorf("Scan %v quant %v has no eVstart column", e.scanId, e.quantId)
	}
	if evPerChanPos == -1 {
		return fmt.Errorf("Scan %v quant %v has no eV/ch column", e.scanId, e.quantId)
	}

	e.quantEVCalibration = &protos.ClientEnergyCalibration{DetectorCalibrations: map[string]*protos.ClientSpectrumEnergyCalibration{}}

	for _, locSet := range e.quantData.LocationSet {
		totalStart := float32(0)
		totalPerChan := float32(0)

		for _, loc := range locSet.Location {
			totalStart += loc.Values[evStartPos].Fvalue
			totalPerChan += loc.Values[evPerChanPos].Fvalue
		}

		e.quantEVCalibration.DetectorCalibrations[locSet.Detector] = &protos.ClientSpectrumEnergyCalibration{
			StarteV:      totalStart / float32(len(locSet.Location)),
			PerChanneleV: totalPerChan / float32(len(locSet.Location)),
		}
	}

	return nil
}

func (e *expressionRunner) getQuantColIndex(dataLabel string) (int, *float64) {
	idx := slices.Index(e.quantData.Labels, dataLabel)

	var toElemConvert *float64
	if idx < 0 {
		// Since PIQUANT supporting carbonate/oxides, we may need to calculate this from an existing column
		// (we do this for carbonate->element or oxide->element)
		// Check what's being requested, to see if we can convert it
		calcFrom, ok := e.pureElementColumnLookup[dataLabel]
		if ok {
			// we've found a column to look up from (eg dataLabel=Si_% and this found calcFrom=SiO2_%)
			// Get the index of that column
			idx = slices.Index(e.quantData.Labels, calcFrom)

			if idx >= 0 {
				// Also get the conversion factor we'll have to use
				sepIdx := strings.Index(calcFrom, "_")
				if sepIdx > -1 {
					// Following the above examples, this should become SiO2
					oxideOrCarbonate := calcFrom[0:sepIdx]

					elemState := PTable.GetElementOxidationState(oxideOrCarbonate)
					if elemState != nil && !elemState.IsElement {
						toElemConvert = &elemState.ConversionToElementWeightPct
					}
				}
			}
		}
	}

	return idx, toElemConvert
}

func (e *expressionRunner) getQuantifiedDataValues(detectorId string, colIdx int, mult *float64, isPctColumn bool) (PMCDataValues, error) {
	result := PMCDataValues{}
	result.IsBinary = true // pre-set for detection in addValue
	detectorFound := false

	// NOTE: if requesting "default" detector, just pick the first one
	if detectorId == DefaultDetectorId && len(e.quantData.LocationSet) > 0 {
		detectorId = e.quantData.LocationSet[0].Detector
	}

	// Loop through all our locations by PMC, find the quant value, store
	for _, quantLocSet := range e.quantData.LocationSet {
		if quantLocSet.Detector == detectorId {
			for _, quantLoc := range quantLocSet.Location {
				valueItem := quantLoc.Values[colIdx]
				value := float64(valueItem.Fvalue)
				undef := false

				if e.quantData.Types[colIdx] == protos.Quantification_QT_INT {
					value = float64(valueItem.Ivalue)
				}

				// If we're a _% column, and the value is -1, ignore it. This is due to
				// multi-quantifications where we decided to use -1 to signify a missing
				// value (due to combining 2 quants with different element lists, therefore
				// ending up with some PMCs that have no value for a given element in the
				// quantification).
				if isPctColumn && value == -1 {
					value = 0
					undef = true
				} else if mult != nil {
					value *= *mult
				}

				result.AddValue(PMCDataValue{PMC: int(quantLoc.Pmc), Value: value, IsUndefined: undef})
			}

			detectorFound = true
		}
	}

	if !detectorFound {
		detectors := map[string]bool{}
		for _, quantLocSet := range e.quantData.LocationSet {
			detectors[quantLocSet.Detector] = true
		}

		return result, fmt.Errorf(`Scan %v quantification does not contain data for detector: "%v". It only contains detector(s): "%v"`, e.scanId, detectorId, strings.Join(utils.GetMapKeys(detectors), ","))
	}

	return result, nil
}

func (e *expressionRunner) GetQuantifiedDataForDetector(detectorId string, dataLabel string) (PMCDataValues, error) {
	quantColIdx, quantColToElemConvert := e.getQuantColIndex(dataLabel)

	if quantColIdx < 0 {
		return PMCDataValues{}, fmt.Errorf(`Scan %v quantification does not contain column: "%v". Please select (or create) a quantification with the relevant element.`, e.scanId, dataLabel)
	}

	return e.getQuantifiedDataValues(detectorId, quantColIdx, quantColToElemConvert, strings.HasSuffix(dataLabel, "_%"))
}

// Returns detectorIdIdx, readtypeIdx
func (e *expressionRunner) getDetectorReadTypeMetaIdxs() (int, int, error) {
	detectorIdIdx := -1
	readtypeIdx := -1
	for c, label := range e.scan.MetaLabels {
		if label == "DETECTOR_ID" {
			detectorIdIdx = c
			if e.scan.MetaTypes[c] != protos.Experiment_MT_STRING {
				return -1, -1, fmt.Errorf("Scan %v has invalid meta field type for %v: %v", e.scanId, label, e.scan.MetaTypes[c])
			}
		} else if label == "READTYPE" {
			readtypeIdx = c
			if e.scan.MetaTypes[c] != protos.Experiment_MT_STRING {
				return -1, -1, fmt.Errorf("Scan %v has invalid meta field type for %v: %v", e.scanId, label, e.scan.MetaTypes[c])
			}
		}

		if readtypeIdx > -1 && detectorIdIdx > -1 {
			break
		}
	}

	return detectorIdIdx, readtypeIdx, nil
}
