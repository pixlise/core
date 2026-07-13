package expressionrunner

import (
	"fmt"
	"slices"
	"strconv"
	"strings"

	lua "github.com/Shopify/go-lua"
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// We need a way for Go functions called from lua to find the state they're calling into so we know what
// quant/scan etc to load
var contextIdLuaVarName = "execContextId"

func (e *expressionRunner) defineRuntime(L *lua.State, contextId int) {
	L.PushInteger(contextId)
	L.SetGlobal(contextIdLuaVarName)

	L.Register("element", element)
	L.Register("elementSum", elementSum)
	L.Register("data", data)
	L.Register("spectrum", spectrum)
	L.Register("spectrumDiff", spectrumDiff)
	L.Register("pseudo", pseudo)
	L.Register("housekeeping", housekeeping)
	L.Register("diffractionPeaks", diffractionPeaks)
	L.Register("roughness", roughness)
	L.Register("position", position)
	L.Register("makeMap", makeMap)
	L.Register("exists", exists)
	L.Register("writeCache", writeCache)
	L.Register("readCache", readCache)
	L.Register("readMap", readMap)
}

func getContext(L *lua.State) *expressionRunner {
	// Get the context global
	L.Global(contextIdLuaVarName)
	ctxId, ok := L.ToInteger(-1)
	if !ok {
		return nil
	}

	return getExpressionContext(ctxId)
}

func convertToMmol(formula string, values PMCDataValues) PMCDataValues {
	result := []PMCDataValue{}

	conversion := float64(1)

	/* REMOVED Because this was a more special case, see the new FeO-T workaround below
	   // Also note, FeO-T can be converted to Fe2O3 by multiplying by 1.111 according to email from Balz Kamber
	   if(formula == "FeO-T")
	   {
	       conversion = 1.111;
	       formula = "Fe2O3";
	   }
	*/
	/* Modified because it now turns out we have other special cases such as FeCO3-T, so lets make it generic...
	   if(formula == "FeO-T")
	   {
	       // We don't know what flavour of FeO we're dealing with, just the total. Mike discovered that the above 1.111 conversion
	       // was giving back values 2x as large as expected. Just treat it like FeO
	       formula = "FeO";
	   }
	*/

	// We are dealing with a "total" quantification of something, eg FeO, so here we just treat it like the element being quantified!
	formula = strings.TrimSuffix(formula, "-T")

	mass := PTable.GetMolecularMass(formula)
	if mass > 0 {
		// Success parsing it, work out the conversion factor:
		// This came from an email from Joel Hurowitz:
		// weight % (eg 30%) -> decimal (div by 100)
		// divide by mass
		// mult by 1000 to give mol/kg
		conversion *= 10 / mass // AKA: 1/100/mass*1000;
	}

	valRange := scan.MinMax{}
	for c := 0; c < len(values.Values); c++ {
		valToSave := float64(0)
		if !values.Values[c].IsUndefined {
			valToSave = values.Values[c].Value * conversion
		}

		result = append(result, MakePMCDataValue(values.Values[c].PMC, valToSave, values.Values[c].IsUndefined, ""))
		valRange.Expand(valToSave)
	}

	return makePMCDataValuesWithMinMax(result, valRange, false)
}

func reportLuaRuntimeError(L *lua.State, err error) int {
	L.PushFString("PIXLISE-Lua Runtime error: %v", err)
	L.Error()
	return 0
}

func element(L *lua.State) int { // args(symbol, column, detector)
	e := getContext(L)
	if e == nil {
		return 0
	}

	symbol, ok := L.ToString(1)
	if !ok {
		reportLuaRuntimeError(L, fmt.Errorf("Expected symbol string"))
	}
	column, ok := L.ToString(2)
	if !ok {
		reportLuaRuntimeError(L, fmt.Errorf("Expected column string"))
	}
	detector, ok := L.ToString(3)
	if !ok {
		reportLuaRuntimeError(L, fmt.Errorf("Expected detector string"))
	}

	asMmol := false
	if column == "%-as-mmol" {
		column = "%"
		asMmol = true
	}

	// Look this up in the quant we're expecting
	if err := e.ensureFetchedQuant(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	dataLabel := symbol + "_" + column

	result, err := e.GetQuantifiedDataForDetector(detector, dataLabel)
	if err != nil {
		return reportLuaRuntimeError(L, err)
	}

	if asMmol {
		result = convertToMmol(symbol, result)
	}

	pushAsLuaTable(result, L)
	return 1
}

func elementSum(L *lua.State) int { // args(column, detector)
	e := getContext(L)
	if e == nil {
		return 0
	}

	// column := L.ToString(1)
	// detector := L.ToString(2)
	return reportLuaRuntimeError(L, fmt.Errorf("elementSum not implemented yet"))
}

func data(L *lua.State) int { // args(column, detector)
	e := getContext(L)
	if e == nil {
		return 0
	}

	column, _ := L.ToString(1)
	detector, _ := L.ToString(2)

	if err := e.ensureFetchedScan(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	idx := slices.Index(e.scan.MetaLabels, column)
	if idx > -1 {
		// We're returning spectrum metadata
		return reportLuaRuntimeError(L, fmt.Errorf("data() - returning spectrum meta data not implemented yet"))
		// L.Push(lua.LBool(false))
		// return 1
	}

	result, err := e.GetQuantifiedDataForDetector(detector, column)
	if err != nil {
		return reportLuaRuntimeError(L, err)
	}

	pushAsLuaTable(result, L)
	return 1
}

func spectrum(L *lua.State) int { // args(startChannel, endChannel, detector)
	e := getContext(L)
	if e == nil {
		return 0
	}

	startChannel, _ := L.ToInteger(1)
	endChannel, _ := L.ToInteger(2)
	detector, _ := L.ToString(3)

	if err := e.ensureFetchedScan(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	// Get indexes and data types for our meta fields used to identify the spectrum
	detectorIdIdx, readtypeIdx, err := e.getDetectorReadTypeMetaIdxs()
	if err != nil {
		return reportLuaRuntimeError(L, err)
	}

	result := PMCDataValues{}
	foundRange := false

	metaIdxs := []int{detectorIdIdx, readtypeIdx}
	for _, loc := range e.scan.Locations {
		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return reportLuaRuntimeError(L, fmt.Errorf("Failed to read PMC: \"%v\" for scan: %v", loc.Id, e.scanId))
		}

		for _, det := range loc.Detectors {
			metaValues := getScanDetectorMetaValues(metaIdxs, det)

			// Check if we have values
			if m, ok := metaValues[detectorIdIdx]; ok && m.Svalue == detector {
				// We're interested in this detector! Do we care about this spectrum type?
				if m2, ok := metaValues[readtypeIdx]; ok && m2.Svalue == "Normal" {
					// Include this spectrum
					spectrum := client.ZeroRunDecode(det.Spectrum)

					// Now grab the values in the channel
					channelEndToReadTo := endChannel
					if channelEndToReadTo > len(spectrum) {
						channelEndToReadTo = len(spectrum)
					}

					// Loop through & add it
					sum := int32(0)
					for ch := startChannel; ch < channelEndToReadTo; ch++ {
						sum += spectrum[ch]
					}

					result.AddValue(MakePMCDataValue(pmc, float64(sum), false, ""))
					foundRange = true
				}
			}
		}
	}

	if !foundRange {
		return reportLuaRuntimeError(L, fmt.Errorf("spectrum: Failed to find scan %v spectrum %v range between %v and %v", e.scanId, detector, startChannel, endChannel))
	}

	pushAsLuaTable(result, L)
	return 1
}

func spectrumDiff(L *lua.State) int { // args(startChannel, endChannel, op)
	e := getContext(L)
	if e == nil {
		return 0
	}

	// startChannel := L.ToInteger(1)
	// endChannel := L.ToInteger(2)
	// op := L.ToString(3)
	return reportLuaRuntimeError(L, fmt.Errorf("spectrumDiff not implemented yet"))
}

func pseudo(L *lua.State) int { // args(elem)
	e := getContext(L)
	if e == nil {
		return 0
	}

	// elem := L.ToString(1)
	return reportLuaRuntimeError(L, fmt.Errorf("pseudo not implemented yet"))
}

func housekeeping(L *lua.State) int { // args(column)
	e := getContext(L)
	if e == nil {
		return 0
	}

	column, _ := L.ToString(1)

	if err := e.ensureFetchedScan(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	metaIdx := slices.Index(e.scan.MetaLabels, column)
	if metaIdx < 0 {
		return reportLuaRuntimeError(L, fmt.Errorf(`Scan %v does not include housekeeping data with column name: "%v"`, e.scanId, column))
	}

	// Verify it's a type usable in the expression language
	if e.scan.MetaTypes[metaIdx] != protos.Experiment_MT_FLOAT && e.scan.MetaTypes[metaIdx] != protos.Experiment_MT_INT {
		return reportLuaRuntimeError(L, fmt.Errorf("Non-numeric data type for housekeeping data column: %v", column))
	}

	// Run through all locations & build it
	result := PMCDataValues{}
	result.IsBinary = true // pre-set for detection in addValue

	for _, loc := range e.scan.Locations {
		if len(loc.Meta) <= 0 {
			continue
		}
		/*
			// To determine if it's a PMC we want to include,
			loc.PseudoIntensities
			if loc.Beam == nil || len(loc.Detectors) <= 0 {
				// Not containing spectra
				continue
			}

			if !hasSpectra(loc, readtypeIdx, detectorIdIdx, "Normal") {
				// Not containing normal spectra
				continue
			}
		*/

		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return reportLuaRuntimeError(L, fmt.Errorf("Failed to read PMC: \"%v\"", loc.Id))
		}

		value := float64(loc.Meta[metaIdx].Fvalue)
		if e.scan.MetaTypes[metaIdx] != protos.Experiment_MT_INT {
			value = float64(loc.Meta[metaIdx].Ivalue)
		}

		result.AddValue(MakePMCDataValue(pmc, value, false, ""))
	}

	pushAsLuaTable(result, L)
	return 1
}

func getScanDetectorMetaValues(metaIdxs []int /*dataType protos.Experiment_MetaDataType,*/, det *protos.Experiment_Location_DetectorSpectrum) map[int]*protos.Experiment_Location_MetaDataItem {
	results := map[int]*protos.Experiment_Location_MetaDataItem{}

	for _, meta := range det.Meta {
		// Is it one we're interested in?
		if utils.ItemInSlice(int(meta.LabelIdx), metaIdxs) {
			results[int(meta.LabelIdx)] = meta
		}
	}

	return results
}

/*
	func hasSpectra(loc *protos.Experiment_Location, readtypeIdx int, detectorIdIdx int, spectraType string) bool {
		foundSpectraType := false

		for _, det := range loc.Detectors {
			for _, m := range det.Meta {
				if m.LabelIdx >= int32(len(dataset.MetaLabels)) {
					return nil, fmt.Errorf("LabelIdx %v out of range when reading meta", m.LabelIdx)
				}

				label := dataset.MetaLabels[m.LabelIdx]
				if m.LabelIdx == int32(detectorIdIdx) {
					// Verify type
					if t := metaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
						detectorId = m.Svalue
					} else {
						return nil, fmt.Errorf("Unexpected %v when reading detector id", t)
					}
				} else if m.LabelIdx == int32(readtypeIdx) {
					// Verify type
					if t := metaTypes[m.LabelIdx]; t == protos.Experiment_MT_STRING {
						// These are hard-coded string values
						if m.Svalue == spectraType {
							foundSpectraType = true
							break
						}
					}
				}
			}

			if foundSpectraType {
				break
			}
		}

		return foundSpectraType
	}
*/
func diffractionPeaks(L *lua.State) int { // args(eVstart, eVend)
	e := getContext(L)
	if e == nil {
		return 0
	}

	startChannel, _ := L.ToInteger(1)
	endChannel, _ := L.ToInteger(2)

	if err := e.ensureFetchedScan(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	// Ensure we have the quant already too, because thats where we get energy calibration from
	if err := e.ensureFetchedQuant(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	if err := e.ensureFetchedDiffraction(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	// Run through all our diffraction peak data and return the sum of all peaks within the given channel range

	// First, add them up per PMC
	pmcDiffractionCount := map[int]int{}

	// Fill the PMCs first
	detectorIdIdx, readtypeIdx, err := e.getDetectorReadTypeMetaIdxs()
	if err != nil {
		return reportLuaRuntimeError(L, err)
	}

	metaIdxs := []int{detectorIdIdx, readtypeIdx}
	for _, loc := range e.scan.Locations {
		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return reportLuaRuntimeError(L, fmt.Errorf("Failed to read PMC: \"%v\" for scan: %v", loc.Id, e.scanId))
		}

		for _, det := range loc.Detectors {
			metaValues := getScanDetectorMetaValues(metaIdxs, det)

			// Check if we have values
			if m, ok := metaValues[detectorIdIdx]; ok && m.Svalue == "A" {
				// We're interested in this detector! Do we care about this spectrum type?
				if m2, ok := metaValues[readtypeIdx]; ok && m2.Svalue == "Normal" {
					// Include this spectrum
					pmcDiffractionCount[pmc] = 0
				}
			}
		}
	}

	for _, peak := range e.allPeaks {
		withinChannelRange := (startChannel == -1 || int(peak.Peak.PeakChannel) >= startChannel) && (endChannel == -1 || int(peak.Peak.PeakChannel) < endChannel)
		if peak.Status != DiffractionStatusNotAnomaly && withinChannelRange {
			prev, ok := pmcDiffractionCount[int(peak.Id)]
			if !ok {
				prev = 0
			}
			pmcDiffractionCount[int(peak.Id)] = prev + 1
		}
	}

	// Also loop through user-defined peaks
	// If we can convert the user peak keV to a channel, do it and compare
	if e.quantEVCalibration != nil && len(e.quantEVCalibration.DetectorCalibrations) > 0 && len(e.manualPeaks) > 0 {
		for detector, calibration := range e.quantEVCalibration.DetectorCalibrations {
			if detector == EnergyCalibrationDetector {
				for _, peak := range e.manualPeaks {
					// ONLY look at positive energies, negative means it's a user-entered roughness item!
					if peak.EnergykeV > 0 {
						ch := keVToChannel(peak.EnergykeV, calibration)
						if ch >= startChannel && ch < endChannel {
							pmcDiffractionCount[int(peak.Pmc)] = pmcDiffractionCount[int(peak.Pmc)] + 1
						}
					}
				}

				break
			}
		}
	}

	// Now turn these into data values
	result := PMCDataValues{}
	result.IsBinary = true // pre-set for detection in addValue

	for pmc, sum := range pmcDiffractionCount {
		result.AddValue(MakePMCDataValue(int(pmc), float64(sum), false, ""))
	}

	pushAsLuaTable(result, L)
	return 1
}

func roughness(L *lua.State) int { // args()
	e := getContext(L)
	if e == nil {
		return 0
	}

	if err := e.ensureFetchedScan(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	if err := e.ensureFetchedDiffraction(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	result := PMCDataValues{}
	result.IsBinary = true // pre-set for detection in addValue

	for _, item := range e.roughnessItems {
		result.AddValue(MakePMCDataValue(int(item.Id), float64(item.GlobalDifference), false, ""))
	}

	// Also run through user-defined roughness items
	if len(e.manualPeaks) > 0 {
		for _, peak := range e.manualPeaks {
			// ONLY negative means it's a user-entered roughness item!
			if peak.EnergykeV < 0 {
				result.AddValue(MakePMCDataValue(int(peak.Pmc), float64(client.RoughnessItemThreshold), false, ""))
			}
		}
	}

	pushAsLuaTable(result, L)
	return 1
}

func position(L *lua.State) int { // args(axis)
	e := getContext(L)
	if e == nil {
		return 0
	}

	// axis := L.ToString(1)
	return reportLuaRuntimeError(L, fmt.Errorf("position not implemented yet"))
}

func makeMap(L *lua.State) int { // args(value)
	e := getContext(L)
	if e == nil {
		return 0
	}

	// Look this up in the quant we're expecting
	if err := e.ensureFetchedQuant(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	value, _ := L.ToNumber(1)

	result := PMCDataValues{}
	result.IsBinary = true // pre-set for detection in addValue
	if len(e.quantData.LocationSet) > 0 {
		for _, locItem := range e.quantData.LocationSet[0].Location {
			result.AddValue(MakePMCDataValue(int(locItem.Pmc), float64(value), false, ""))
		}
	}

	pushAsLuaTable(result, L)
	return 1
}

func exists(L *lua.State) int { // args(dataType, column)
	e := getContext(L)
	if e == nil {
		return 0
	}

	dataType, _ := L.ToString(1)
	column, _ := L.ToString(2)

	// Check if the data is available
	if dataType == "element" || dataType == "detector" || dataType == "data" {
		// Look this up in the quant we're expecting
		if err := e.ensureFetchedQuant(); err != nil {
			return reportLuaRuntimeError(L, err)
		}

		if dataType == "element" {
			L.PushBoolean(utils.ItemInSlice(column, utils.GetMapKeys(e.elementColumns)))
			return 1
		} else if dataType == "detector" {
			var found bool
			for _, locSet := range e.quantData.LocationSet {
				if column == locSet.Detector {
					found = true
					break
				}
			}

			L.PushBoolean(found)
			return 1
		} else { // "data"
			L.PushBoolean(utils.ItemInSlice(column, e.quantData.Labels))
			return 1
		}
	} else if dataType == "housekeeping" || dataType == "pseudo" {
		if err := e.ensureFetchedScan(); err != nil {
			return reportLuaRuntimeError(L, err)
		}

		// Read what's needed
		if dataType == "housekeeping" {
			idx := slices.Index(e.scan.MetaLabels, column)
			if idx < 0 {
				L.PushBoolean(false)
				return 1
			}

			// Verify it's a type usable in the expression language
			L.PushBoolean(e.scan.MetaTypes[idx] == protos.Experiment_MT_FLOAT || e.scan.MetaTypes[idx] == protos.Experiment_MT_INT)
			return 1
		} else {
			var found bool
			for _, item := range e.scan.PseudoIntensityRanges {
				if item.Name == column {
					found = true
					break
				}
			}

			L.PushBoolean(found)
			return 1
		}
	}

	return reportLuaRuntimeError(L, fmt.Errorf("Unknown data type %v for exists()", dataType))
}

func writeCache(L *lua.State) int { // args(k, v)
	return 0 // No-Op
}

func readCache(L *lua.State) int { // args(k, w)
	return 0 // No-Op
}

func readMap(L *lua.State) int { // args(k)
	e := getContext(L)
	if e == nil {
		return 0
	}

	//k := L.ToString(1)
	return reportLuaRuntimeError(L, fmt.Errorf("readMap not implemented yet"))
}
