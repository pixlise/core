package expressionrunner

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/piquant"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/sessionuser"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/periodictable"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
	lua "github.com/yuin/gopher-lua"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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

func RunExpression(expressionId string, scanId string, quantId string, svcs *services.APIServices, debug bool) (*PMCDataValues, error) {
	r, err := makeExpressionRunner(expressionId, scanId, quantId, svcs)
	if err != nil {
		return nil, err
	}

	// If we're debugging:
	r.writeLuaSource = true
	//r.debugUseLocalSourceFile = true

	return r.Run()
}

func keVToChannel(energy float32, calibration *protos.ClientSpectrumEnergyCalibration) int {
	return int(math.Floor(float64((energy*1000 - calibration.StarteV) / calibration.PerChanneleV)))
}

type expressionRunner struct {
	// Parameters that triggered us
	expressionId, scanId, quantId string

	// Tools
	svcs *services.APIServices

	// Loaded quant stuff
	quantData               *protos.Quantification
	pureElementColumnLookup map[string]string
	elementColumns          map[string][]string

	// One calibration per detector, PIXL only has A & B
	quantEVCalibration *protos.ClientEnergyCalibration

	// Scan stuff
	scan *protos.Experiment

	// Diffraction stuff
	diffractionFile *protos.Diffraction
	allPeaks        []*protos.ClientDiffractionPeak
	roughnessItems  []*protos.ClientRoughnessItem
	manualPeaks     map[string]*protos.ManualDiffractionPeak

	debugUseLocalSourceFile bool
	writeLuaSource          bool
}

// For now this is the only thing in our code that requires the periodic table
// If this changes, this probably needs to be put into services.APIServices
var PTable *periodictable.PeriodicTableDB

func makeExpressionRunner(expressionId string, scanId string, quantId string, svcs *services.APIServices) (*expressionRunner, error) {
	runner := &expressionRunner{
		expressionId:            expressionId,
		scanId:                  scanId,
		quantId:                 quantId,
		svcs:                    svcs,
		pureElementColumnLookup: map[string]string{},
		elementColumns:          map[string][]string{},
		allPeaks:                []*protos.ClientDiffractionPeak{},
		roughnessItems:          []*protos.ClientRoughnessItem{},
		manualPeaks:             map[string]*protos.ManualDiffractionPeak{},
	}

	if PTable == nil {
		PTable = periodictable.MakePeriodicTable(svcs.Log)
	}
	return runner, nil
}

func (e *expressionRunner) Run() (*PMCDataValues, error) {
	contextId := addExpressionContext(e)
	defer clearExpressionContext(contextId)

	// Get the scan and the detector config
	scan := &protos.ScanItem{}
	err := readOne(dbCollections.ScansName, bson.M{"_id": e.scanId}, scan, e.svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	if len(scan.InstrumentConfig) <= 0 {
		return nil, fmt.Errorf("Scan %v has no instrument config", e.scanId)
	}

	detectorConfig, _ /*versions*/, err := piquant.ReadConfig(scan.InstrumentConfig, e.svcs)
	if err != nil {
		return nil, err
	}

	// Retrieve the expression source and all of its modules first
	allSource, err := e.fetchSourceCode()

	// Add constants as required to Lua
	allSource = fmt.Sprintf(`local elevAngle = %v
local quantId = "%v"
local scanId = "%v"
local maxSpectrumChannel = %v
local instrument = "%v"
local userId = "%v"
`,
		detectorConfig.ElevAngle,
		e.quantId,
		e.scanId,
		4096,
		"PIXL_FM",
		sessionuser.PIXLISESystemUserId) + allSource

	// Replace table.unpack with unpack because gopher-lua is 5.1, table.unpack came in 5.2 but they're the same thing apparently
	allSource = strings.ReplaceAll(allSource, "table.unpack(", "unpack(")
	allSource = strings.ReplaceAll(allSource, "getmetatable(obj) == Estimate", "#obj > 0")

	if e.debugUseLocalSourceFile {
		// For debugging purposes we can read the file instead of just write it!
		b, err := os.ReadFile("all-source.txt")
		if err != nil {
			return nil, fmt.Errorf("Failed to read all-source.txt: %v", err)
		}
		allSource = string(b)
	} else {
		// write out the source code we run in lua
		if e.writeLuaSource {
			if err = os.WriteFile("all-source.txt", []byte(allSource), 0777); err != nil {
				return nil, err
			}
		}
	}

	return e.runSource(allSource, contextId)
}

func (e *expressionRunner) Log() logger.ILogger {
	return e.svcs.Log
}

func (e *expressionRunner) fetchSourceCode() (string, error) {
	// Read expression
	expr := &protos.DataExpression{}
	err := readOne(dbCollections.ExpressionsName, bson.M{"_id": e.expressionId}, expr, e.svcs.MongoDB)
	if err != nil {
		return "", err
	}

	if expr.SourceLanguage != "LUA" {
		return "", fmt.Errorf("Error: Expression %v is not Lua", e.expressionId)
	}

	allSource := ""

	// Read built-in modules
	builtInModules := []string{"./built-in-modules/Map.lua", "./built-in-modules/DebugHelp.lua"}
	for _, modPath := range builtInModules {
		modSrcFile, err := os.ReadFile(modPath)
		if err != nil {
			return "", err
		}

		modSrc := snipReturnModuleLine(string(modSrcFile))
		allSource = allSource + "\n" + modSrc + "\n"
	}

	// Read modules
	for _, modRef := range expr.ModuleReferences {
		_, modVer, err := readModule(modRef.ModuleId, modRef.Version, e.svcs)
		if err != nil {
			return "", err
		}

		modSrc := snipReturnModuleLine(modVer.SourceCode)
		allSource = allSource + "\n" + modSrc + "\n"
	}

	allSource = allSource + expr.SourceCode
	return allSource, nil
}

func (e *expressionRunner) runSource(source string, contextId int) (*PMCDataValues, error) {
	L := lua.NewState()
	defer L.Close()

	e.defineRuntime(L, contextId)

	//L.NewFunction()
	if err := L.DoString(source); err != nil {
		return nil, err
	}

	// Get the result if there is one
	resultTable := L.ToTable(-1)

	if resultTable == nil {
		return nil, fmt.Errorf("Expression %v did not return map data", e.expressionId)
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

func snipReturnModuleLine(src string) string {
	pos := strings.LastIndex(src, "return ")
	if pos > -1 {
		return src[0:pos]
	}
	return src
}

func readOne[T any](collectionName string, filter bson.M, intoItem *T, db *mongo.Database) error {
	ctx := context.TODO()
	coll := db.Collection(collectionName)

	dbResult := coll.FindOne(ctx, filter, options.FindOne())
	if dbResult.Err() != nil {
		return dbResult.Err()
	}

	return dbResult.Decode(intoItem)
}

func readModule(moduleId string, version *protos.SemanticVersion, svcs *services.APIServices) (*protos.DataModuleDB, *protos.DataModuleVersionDB, error) {
	mod := &protos.DataModuleDB{}
	err := readOne(dbCollections.ModulesName, bson.M{"_id": moduleId}, mod, svcs.MongoDB)

	if err != nil {
		return nil, nil, err
	}

	modVer := &protos.DataModuleVersionDB{}
	filter := bson.M{"_id": fmt.Sprintf("%v-v%v.%v.%v", moduleId, version.Major, version.Minor, version.Patch)}
	err = readOne(dbCollections.ModuleVersionsName, filter, modVer, svcs.MongoDB)

	if err != nil {
		return nil, nil, err
	}

	return mod, modVer, nil
}
