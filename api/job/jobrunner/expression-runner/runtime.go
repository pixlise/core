package expressionrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/client"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// We need a way for Go functions called from lua to find the state they're calling into so we know what
// quant/scan etc to load
var contextIdLuaVarName = "execContextId"

func (e *expressionRunner) defineRuntime(L *lua.LState, contextId int, makeMapSuffix string) {
	L.SetGlobal(contextIdLuaVarName, lua.LNumber(contextId))

	L.SetGlobal("element", L.NewFunction(element))
	L.SetGlobal("elementSum", L.NewFunction(elementSum))
	L.SetGlobal("data", L.NewFunction(data))
	L.SetGlobal("spectrum", L.NewFunction(spectrum))
	L.SetGlobal("spectrumDiff", L.NewFunction(spectrumDiff))
	L.SetGlobal("pseudo", L.NewFunction(pseudo))
	L.SetGlobal("housekeeping", L.NewFunction(housekeeping))
	L.SetGlobal("diffractionPeaks", L.NewFunction(diffractionPeaks))
	L.SetGlobal("roughness", L.NewFunction(roughness))
	L.SetGlobal("position", L.NewFunction(position))
	L.SetGlobal("makeMap"+makeMapSuffix, L.NewFunction(makeMap))
	L.SetGlobal("exists", L.NewFunction(exists))
	L.SetGlobal("writeCache", L.NewFunction(writeCache))
	L.SetGlobal("readCache", L.NewFunction(readCache))
	L.SetGlobal("readMap", L.NewFunction(readMap))
	L.SetGlobal("atomicMass", L.NewFunction(atomicMass))
}

func getContext(L *lua.LState) *expressionRunner {
	// Get the context global
	ctxId := L.GetGlobal(contextIdLuaVarName)
	if ctxId == nil {
		return nil
	}

	return getExpressionContext(int(ctxId.(lua.LNumber)))
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

		result = append(result, makePMCDataValue(values.Values[c].PMC, valToSave, values.Values[c].IsUndefined, ""))
		valRange.Expand(valToSave)
	}

	return makePMCDataValuesWithMinMax(result, valRange, false)
}

func reportLuaRuntimeError(L *lua.LState, err error) int {
	L.RaiseError("PIXLISE-Lua Runtime error: %v", err)
	return 0
}

func funcStart(L *lua.LState) (*expressionRunner, time.Time) {
	return getContext(L), time.Now()
}

func (e *expressionRunner) funcPrintArgs(funcName string, args ...interface{}) {
	f := ""
	for c := range args {
		if c > 0 {
			f = f + ","
		}
		f = f + "%v"
	}
	//f = fmt.Sprintf(f, args)
	if e.traceRuntimeCalls {
		e.Log().Debugf("    Lua runtime:   "+funcName+"("+f+")", args...)
	}
}

func (e *expressionRunner) funcEnd(startTime time.Time) {
	runtime := time.Since(startTime)

	if e.traceRuntimeCalls {
		e.Log().Debugf("                -> %vms", runtime.Milliseconds())
	}

	e.totalGoFunctionRuntimeNs += uint64(runtime.Nanoseconds())
}

func element(L *lua.LState) int { // args(symbol, column, detector)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	symbol := L.ToString(1)
	column := L.ToString(2)
	detector := L.ToString(3)

	e.funcPrintArgs("element", symbol, column, detector)

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

	L.Push(makeLuaTable(result))
	return 1
}

func elementSum(L *lua.LState) int { // args(column, detector)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	column := L.ToString(1)
	detector := L.ToString(2)

	e.funcPrintArgs("elementSum[NOOP]", column, detector)

	return reportLuaRuntimeError(L, fmt.Errorf("elementSum not implemented yet"))
}

func data(L *lua.LState) int { // args(column, detector)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	column := L.ToString(1)
	detector := L.ToString(2)

	e.funcPrintArgs("data", column, detector)

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

	L.Push(makeLuaTable(result))
	return 1
}

func spectrum(L *lua.LState) int { // args(startChannel, endChannel, detector)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	startChannel := L.ToInt(1)
	endChannel := L.ToInt(2)
	detector := L.ToString(3)

	e.funcPrintArgs("spectrum", startChannel, endChannel, detector)

	if err := e.ensureFetchedScanNormalSpectra(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	// Sanity checks
	pmcCount := len(e.pmcToLocationIndex)
	if pmcCount != len(e.locationIndexToPMC) {
		return reportLuaRuntimeError(L, fmt.Errorf("PMC lookup size is invalid: %v vs expected: %v", pmcCount, len(e.locationIndexToPMC)))
	}

	// Check detectors have the same lookup size
	for d, s := range e.spectra {
		if pmcCount != len(s) {
			return reportLuaRuntimeError(L, fmt.Errorf("Spectra detector %v lookup size is invalid: %v vs expected: %v", d, len(s), pmcCount))
		}
	}

	detectorSpectra, ok := e.spectra[detector]
	if !ok {
		return reportLuaRuntimeError(L, fmt.Errorf("Failed to find spectra for detector %v", detector))
	}

	// Run through all spectra for this detector and add up items within channel range
	result := PMCDataValues{}
	foundRange := false

	channelEndToReadTo := endChannel
	if channelEndToReadTo > e.spectrumCounts {
		channelEndToReadTo = e.spectrumCounts
	}

	for c, pmc := range e.locationIndexToPMC {
		// Now grab the values in the channel
		spectrum := detectorSpectra[c]

		// Loop through & add it
		sum := int32(0)
		for ch := startChannel; ch < channelEndToReadTo; ch++ {
			sum += spectrum[ch]
		}

		result.AddValue(makePMCDataValue(pmc, float64(sum), false, ""))
		foundRange = true
	}

	if !foundRange {
		return reportLuaRuntimeError(L, fmt.Errorf("spectrum: Failed to find scan %v spectrum %v range between %v and %v", e.scanId, detector, startChannel, endChannel))
	}

	L.Push(makeLuaTable(result))
	return 1
}

func spectrumDiff(L *lua.LState) int { // args(startChannel, endChannel, op)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	startChannel := L.ToInt(1)
	endChannel := L.ToInt(2)
	op := L.ToString(3)

	e.funcPrintArgs("spectrumDiff[NOOP]", startChannel, endChannel, op)

	return reportLuaRuntimeError(L, fmt.Errorf("spectrumDiff not implemented yet"))
}

func pseudo(L *lua.LState) int { // args(elem)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	elem := L.ToString(1)

	e.funcPrintArgs("pseudo[NOOP]", elem)

	return reportLuaRuntimeError(L, fmt.Errorf("pseudo not implemented yet"))
}

func housekeeping(L *lua.LState) int { // args(column)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	column := L.ToString(1)

	e.funcPrintArgs("housekeeping", column)

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

		result.AddValue(makePMCDataValue(pmc, value, false, ""))
	}

	L.Push(makeLuaTable(result))
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
func diffractionPeaks(L *lua.LState) int { // args(eVstart, eVend)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	startChannel := L.ToInt(1)
	endChannel := L.ToInt(2)

	e.funcPrintArgs("diffractionPeaks", startChannel, endChannel)

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
		result.AddValue(makePMCDataValue(int(pmc), float64(sum), false, ""))
	}

	L.Push(makeLuaTable(result))
	return 1
}

func roughness(L *lua.LState) int { // args()
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	e.funcPrintArgs("roughness")

	if err := e.ensureFetchedScan(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	if err := e.ensureFetchedDiffraction(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	result := PMCDataValues{}
	result.IsBinary = true // pre-set for detection in addValue

	for _, item := range e.roughnessItems {
		result.AddValue(makePMCDataValue(int(item.Id), float64(item.GlobalDifference), false, ""))
	}

	// Also run through user-defined roughness items
	if len(e.manualPeaks) > 0 {
		for _, peak := range e.manualPeaks {
			// ONLY negative means it's a user-entered roughness item!
			if peak.EnergykeV < 0 {
				result.AddValue(makePMCDataValue(int(peak.Pmc), float64(client.RoughnessItemThreshold), false, ""))
			}
		}
	}

	L.Push(makeLuaTable(result))
	return 1
}

func position(L *lua.LState) int { // args(axis)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	axis := L.ToString(1)

	e.funcPrintArgs("position[NOOP]", axis)

	return reportLuaRuntimeError(L, fmt.Errorf("position not implemented yet"))
}

func makeMap(L *lua.LState) int { // args(value)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	// Look this up in the quant we're expecting
	if err := e.ensureFetchedQuant(); err != nil {
		return reportLuaRuntimeError(L, err)
	}

	value := L.ToNumber(1)

	e.funcPrintArgs("makeMap", value)

	result := PMCDataValues{}
	result.IsBinary = true // pre-set for detection in addValue
	if len(e.quantData.LocationSet) > 0 {
		for _, locItem := range e.quantData.LocationSet[0].Location {
			result.AddValue(makePMCDataValue(int(locItem.Pmc), float64(value), false, ""))
		}
	}

	L.Push(makeLuaTable(result))
	return 1
}

func exists(L *lua.LState) int { // args(dataType, column)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	dataType := L.ToString(1)
	column := L.ToString(2)

	e.funcPrintArgs("exists", dataType, column)

	// Check if the data is available
	if dataType == "element" || dataType == "detector" || dataType == "data" {
		// Look this up in the quant we're expecting
		if err := e.ensureFetchedQuant(); err != nil {
			return reportLuaRuntimeError(L, err)
		}

		if dataType == "element" {
			L.Push(lua.LBool(utils.ItemInSlice(column, utils.GetMapKeys(e.elementColumns))))
			return 1
		} else if dataType == "detector" {
			var found lua.LBool
			for _, locSet := range e.quantData.LocationSet {
				if column == locSet.Detector {
					found = true
					break
				}
			}

			L.Push(found)
			return 1
		} else { // "data"
			L.Push(lua.LBool(utils.ItemInSlice(column, e.quantData.Labels)))
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
				L.Push(lua.LBool(false))
				return 1
			}

			// Verify it's a type usable in the expression language
			L.Push(lua.LBool(e.scan.MetaTypes[idx] == protos.Experiment_MT_FLOAT || e.scan.MetaTypes[idx] == protos.Experiment_MT_INT))
			return 1
		} else {
			var found lua.LBool
			for _, item := range e.scan.PseudoIntensityRanges {
				if item.Name == column {
					found = true
					break
				}
			}

			L.Push(found)
			return 1
		}
	}

	return reportLuaRuntimeError(L, fmt.Errorf("Unknown data type %v for exists()", dataType))
}

var memoPrefix = "exprcachev1_"

func writeCache(L *lua.LState) int { // args(k, v)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	k := L.ToString(1)

	e.funcPrintArgs("writeCache", k)

	// Assume what's provided is a table
	v := L.ToTable(2)

	if v != nil {
		// Read the table recursively and turn it into JSON
		readMap, readArray := readLuaTable(L, v)

		var b []byte
		var err error

		if len(readArray) > 0 {
			b, err = json.Marshal(readArray)
		} else {
			b, err = json.Marshal(readMap)
		}
		/*
			if len(readArray) > 0 {
				// b, err = json.Marshal(readArray)
				b, err = json.MarshalIndent(readArray, "", utils.PrettyPrintIndentForJSON)
			} else {
				// b, err = json.Marshal(readMap)
				b, err = json.MarshalIndent(readMap, "", utils.PrettyPrintIndentForJSON)
			}

			if err != nil {
				return reportLuaRuntimeError(L, fmt.Errorf("writeCache failed to save json string for %v: %v", k, err))
			}

			// We've got it as JSON, save it to DB as cache item
			// FOR TESTING: Compare to existing
			result := e.svcs.MongoDB.Collection(dbCollections.MemoisedItemsName).FindOne(context.TODO(), bson.M{"_id": memoPrefix + k}, options.FindOne())

			if result.Err() != nil {
				if result.Err() == mongo.ErrNoDocuments {
					// not found in cache
					return 0
				}

				// Some other error
				return reportLuaRuntimeError(L, fmt.Errorf("writeCache error for %v: %v", k, result.Err()))
			}

			// We read it
			item := &protos.MemoisedItem{}

			if err := result.Decode(item); err != nil {
				return reportLuaRuntimeError(L, fmt.Errorf("writeCache decode error for %v: %v", k, err))
			}

			// Take existing and pretty print it so we can compare
			existing := map[string]interface{}{}
			err = json.Unmarshal(item.Data, &existing)
			if err != nil {
				return reportLuaRuntimeError(L, fmt.Errorf("writeCache failed to read cached data for %v: %v", k, err))
			}

			existingPretty, err := json.MarshalIndent(existing, "", utils.PrettyPrintIndentForJSON)
			if err != nil {
				return reportLuaRuntimeError(L, fmt.Errorf("writeCache failed to compare existing for %v: %v", k, err))
			}

			if string(b) != string(existingPretty) {
				// Testing these files shows they are almost matches, just floating point rounding errors about 10 past the .
				// and index is different in each section, so likely it's compatible
				os.WriteFile("newCache_"+k+".txt", b, 0777)
				os.WriteFile("expected_"+k+".txt", existingPretty, 0777)
			} else {
				fmt.Printf("Matches existing cache")
			}
		*/

		// Save this to DB
		timestamp := uint32(e.minimalSvcs.TimeStamper.GetTimeNowSec())
		item := &protos.MemoisedItem{
			Key:                 memoPrefix + k,
			ExprId:              e.expressionId,
			QuantId:             e.quantId,
			ScanId:              e.scanId,
			MemoTimeUnixSec:     timestamp,
			LastReadTimeUnixSec: timestamp,
			Data:                b,
			DataSize:            uint32(len(b)),
			//NoGC: false,
		}

		ctx := context.TODO()
		coll := e.minimalSvcs.MongoDB.Collection(dbCollections.MemoisedItemsName)
		opt := options.Update().SetUpsert(true)

		result, err := coll.UpdateByID(ctx, item.Key, bson.D{{Key: "$set", Value: item}}, opt)
		if err != nil {
			// Some other error
			return reportLuaRuntimeError(L, fmt.Errorf("writeCache error for %v: %v", k, err))
		}

		if result.UpsertedCount == 0 && (result.MatchedCount != result.ModifiedCount) {
			e.minimalSvcs.Log.Errorf("writeCache for: %v got unexpected DB write result: %+v", item.Key, result)
		}
	}

	return 0 // No-Op
}

func readCache(L *lua.LState) int { // args(k, w)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	k := L.ToString(1)
	w := L.ToBool(2)
	e.funcPrintArgs("readCache", k, w)

	result := e.minimalSvcs.MongoDB.Collection(dbCollections.MemoisedItemsName).FindOne(context.TODO(), bson.M{"_id": memoPrefix + k}, options.FindOne())

	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			// not found in cache
			return 0
		}

		// Some other error
		return reportLuaRuntimeError(L, fmt.Errorf("readCache error for %v: %v", k, result.Err()))
	}

	// We read it
	item := &protos.MemoisedItem{}

	if err := result.Decode(item); err != nil {
		return reportLuaRuntimeError(L, fmt.Errorf("readCache decode error for %v: %v", k, result.Err()))
	}

	// Read the data field & decode it into a (nested) Lua table that we can upload
	temp := make(map[string]interface{})
	if err := json.Unmarshal(item.Data, &temp); err != nil {
		return reportLuaRuntimeError(L, fmt.Errorf("readCache decode error data of for %v: %v", k, err))
	}

	ltable := makeLuaTableGeneric(L, temp)
	L.Push(ltable)
	return 1
	// lastread = ltable
	// return 0
}

func readMap(L *lua.LState) int { // args(k)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	k := L.ToString(1)

	e.funcPrintArgs("readMap[NOOP]", k)

	return reportLuaRuntimeError(L, fmt.Errorf("readMap not implemented yet"))
}

func atomicMass(L *lua.LState) int { // args(k)
	e, trc := funcStart(L)
	defer e.funcEnd(trc)
	if e == nil {
		return 0
	}

	symbol := L.ToString(1)

	e.funcPrintArgs("atomicMass", symbol)

	mass := PTable.GetMolecularMass(symbol)
	L.Push(lua.LNumber(mass))
	return 1
}
