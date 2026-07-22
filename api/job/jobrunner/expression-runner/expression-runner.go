package expressionrunner

import (
	"fmt"
	"math"
	"os"
	"time"

	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/periodictable"
	"github.com/pixlise/core/v4/core/scan"
	"github.com/pixlise/core/v4/core/timestamper"
	protos "github.com/pixlise/core/v4/generated-protos"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/mongo"
)

// Runs expressions written in the Lua programming language
// NOTE: Expressions were originally developed as a way for the front-end to allow more user-configurability by allowing
//       users to combine multiple sources of data (eg quantifications, housekeeping data, etc) and draw charts/maps based
//       on that. It grew far more complicated than we ever thought and we're now running 3000+ lines of Lua code on the
//       browser. This was never thought to be sustainable but with caching mechanisms we were able to reduce the issues it
//       caused us. Now we are transitioning to running them in the back-end, as the client-side already treats them as an
//       async operation!
// To run an expression:
// We need to replicate the runtime environment the browser client app was providing to Lua. Originally the idea was to run
// this using the lua interpreter binary, but because Lua reaches out and requests data, it makes more sense to run the
// Lua vm directly in Go and that way Lua requesting data is a function call in Go and we can use our existing retrieval
// mechanisms

// Comparison of speed:
// Browser:
// Expression: Al2O3 (mmol/g) [u59sahioy18frfl9] - Total JS function time: 2,095.7ms, LUA expression "Al2O3 (mmol/g)" took: 17,001.1ms
//
// Go API:
// Expression: Al2O3 (mmol/g) [u59sahioy18frfl9]:
//
// Running as tests (execute, not debug) and including implementing memoisation:
// Nothing saved, fresh recalc: 188750ms (4145ms in Go runtime)
// One saved (non quant) item:  20355ms (1599ms in Go runtime)
// Two saved items:             1000ms

// Returns:
// - map result as PMCDataValues structure
// - total runtime in ms
// - total time spent in runtime (Go implemented Lua functions)
// - error, if any
func RunExpression(expressionId string, scanId string, quantId string,
	log logger.ILogger, db *mongo.Database, ts timestamper.ITimeStamper, remoteFS fileaccess.FileAccess,
	configBucket string, usersBucket string, datasetsBucket string,
	saveCode bool, debug bool) (*PMCDataValues, uint64, uint64, error) {
	ch := make(chan exprResult)

	// Create a temporary fake svcs, because the config reading code requires a bunch of things from it
	// but at this point we may not have an actual svcs - we may be running on a job node!
	minimalSvcs := services.APIServices{
		MongoDB: db,
		FS:      remoteFS,
		Config: config.APIConfig{
			ConfigBucket:   configBucket,
			UsersBucket:    usersBucket,
			DatasetsBucket: datasetsBucket,
		},
		Log:         log,
		TimeStamper: ts,
	}

	go runExpressionInternal(ch, expressionId, scanId, quantId, &minimalSvcs, saveCode, debug)
	result := <-ch

	return result.values, result.totalRuntimeMs, result.totalGoFunctionRuntimeMs, result.err
}

type exprResult struct {
	values                   *PMCDataValues
	totalRuntimeMs           uint64
	totalGoFunctionRuntimeMs uint64
	err                      error
}

func runExpressionInternal(ch chan exprResult,
	expressionId string, scanId string, quantId string,
	minimalSvcs *services.APIServices,
	saveCode bool, debug bool) {
	r, err := makeExpressionRunner(expressionId, scanId, quantId, minimalSvcs)
	if err != nil {
		ch <- exprResult{}
		return
	}

	if debug {
		// If we're debugging:
		r.debugUseLocalSourceFile = true
		r.writeLuaSource = false // don't overwrite!
	} else {
		r.writeLuaSource = saveCode
	}

	// Fetch scan item so we can get the config

	luaSrc, _, err := FetchSourceCode(expressionId, scanId, quantId, sessionuser.PIXLISESystemUserId, minimalSvcs)
	if err != nil {
		result := exprResult{
			err: err,
		}

		ch <- result
		return
	}

	mapResult, err := r.Run(luaSrc)
	result := exprResult{
		values:                   mapResult,
		totalRuntimeMs:           r.totalRuntimeMs,
		totalGoFunctionRuntimeMs: r.totalGoFunctionRuntimeNs / 1000000,
		err:                      err,
	}

	ch <- result
}

func keVToChannel(energy float32, calibration *protos.ClientSpectrumEnergyCalibration) int {
	return int(math.Floor(float64((energy*1000 - calibration.StarteV) / calibration.PerChanneleV)))
}

type expressionRunner struct {
	// Parameters that triggered us
	expressionId, scanId, quantId string

	// Tools
	minimalSvcs *services.APIServices

	// Loaded quant stuff
	quantData               *protos.Quantification
	pureElementColumnLookup map[string]string
	elementColumns          map[string][]string

	// One calibration per detector, PIXL only has A & B
	quantEVCalibration *protos.ClientEnergyCalibration

	// Scan stuff
	scan *protos.Experiment

	detectorIdMetaIdx int
	readTypeMetaIdx   int

	// Here we only consider locations that have 1 or more normal spectra defined!
	// Map of PMC -> "location index"
	pmcToLocationIndex map[int]uint32
	// Map of location index -> PMC
	locationIndexToPMC []int

	// Map of Detector ID -> array by "location index" array of counts
	spectra        map[string][][]int32
	spectrumCounts int

	// Diffraction stuff
	diffractionFile *protos.Diffraction
	allPeaks        []*protos.ClientDiffractionPeak
	roughnessItems  []*protos.ClientRoughnessItem
	manualPeaks     map[string]*protos.ManualDiffractionPeak

	debugUseLocalSourceFile bool
	writeLuaSource          bool

	totalGoFunctionRuntimeNs uint64
	totalRuntimeMs           uint64
}

// For now this is the only thing in our code that requires the periodic table
// If this changes, this probably needs to be put into services.APIServices
var PTable *periodictable.PeriodicTableDB

func makeExpressionRunner(expressionId string, scanId string, quantId string, minimalSvcs *services.APIServices) (*expressionRunner, error) {
	runner := &expressionRunner{
		expressionId:            expressionId,
		scanId:                  scanId,
		quantId:                 quantId,
		minimalSvcs:             minimalSvcs,
		pureElementColumnLookup: map[string]string{},
		elementColumns:          map[string][]string{},
		allPeaks:                []*protos.ClientDiffractionPeak{},
		roughnessItems:          []*protos.ClientRoughnessItem{},
		manualPeaks:             map[string]*protos.ManualDiffractionPeak{},
	}

	if PTable == nil {
		PTable = periodictable.MakePeriodicTable(minimalSvcs.Log)
	}
	return runner, nil
}

func (e *expressionRunner) Run(luaSourceCode string) (*PMCDataValues, error) {
	contextId := addExpressionContext(e)
	defer clearExpressionContext(contextId)

	if e.debugUseLocalSourceFile {
		// For debugging purposes we can read the file instead of just write it!
		b, err := os.ReadFile("all-source.txt")
		if err != nil {
			return nil, fmt.Errorf("Failed to read all-source.txt: %v", err)
		}
		luaSourceCode = string(b)
	} else {
		// write out the source code we run in lua
		if e.writeLuaSource {
			if err := os.WriteFile("all-source.txt", []byte(luaSourceCode), 0777); err != nil {
				return nil, err
			}
		}
	}

	makeMapSuffix := "Raw"
	return e.runSource(luaSourceCode, contextId, makeMapSuffix)
}

func (e *expressionRunner) doString(source string, L *lua.LState) error {
	// We start timing from here... Any calls into our functions will be monitored and tallied so we can report
	// properly how much time was spent in Go vs in Lua VM
	e.totalGoFunctionRuntimeNs = 0
	startTimeMs := time.Now()

	defer func() {
		endTimeMs := time.Since(startTimeMs)
		e.totalRuntimeMs = uint64(endTimeMs.Milliseconds())
	}()

	if err := L.DoString(source); err != nil {
		return err
	}

	return nil
}

func (e *expressionRunner) runSource(source string, contextId int, makeMapSuffix string) (*PMCDataValues, error) {
	L := lua.NewState()
	defer L.Close()

	e.defineRuntime(L, contextId, makeMapSuffix)

	err := e.doString(source, L)
	if err != nil {
		return nil, err
	}

	// Get the result if there is one
	resultTable := L.ToTable(-1)

	if resultTable == nil {
		return nil, fmt.Errorf("Expression %v did not return map data. Stack top: \"%v\"", e.expressionId, L.ToString(-1))
	}

	if resultTable.Len() != 2 {
		return nil, fmt.Errorf("Expression %v did not return map data in expected format", e.expressionId)
	}

	pmcs := resultTable.RawGet(lua.LNumber(1)).(*lua.LTable)
	values := resultTable.RawGet(lua.LNumber(2)).(*lua.LTable)

	if pmcs == nil {
		return nil, fmt.Errorf("Expression %v did not return map data PMC list as expected", e.expressionId)
	}

	if values == nil {
		return nil, fmt.Errorf("Expression %v did not return map data values list as expected", e.expressionId)
	}

	if pmcs.Len() != values.Len() {
		return nil, fmt.Errorf("Expression %v did not return map data with equal number of pmcs and values", e.expressionId)
	}

	// Now we can loop through and retrieve the values
	resultValues := []PMCDataValue{}
	valueRange := scan.MinMax{}

	for c := 0; c < pmcs.Len(); c++ {
		pmc := int(lua.LVAsNumber(pmcs.RawGet(lua.LNumber(c + 1))))
		value := float64(lua.LVAsNumber(values.RawGet(lua.LNumber(c + 1))))

		resultValues = append(resultValues, makePMCDataValue(pmc, value, false, ""))

		valueRange.Expand(value)
	}

	result := makePMCDataValuesWithMinMax(resultValues, valueRange, false)
	return &result, nil
}

func (e *expressionRunner) Log() logger.ILogger {
	return e.minimalSvcs.Log
}
