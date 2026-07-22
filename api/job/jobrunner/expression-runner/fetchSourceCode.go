package expressionrunner

import (
	"context"
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/piquant"
	"github.com/pixlise/core/v4/api/services"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func FetchSourceCode(expressionId string, scanId string, quantId string, userId string, minimalSvcs *services.APIServices) (string, *protos.DataExpression, error) {
	// Read expression
	expr := &protos.DataExpression{}
	err := ReadOne(dbCollections.ExpressionsName, bson.M{"_id": expressionId}, expr, minimalSvcs.MongoDB)
	if err != nil {
		return "", nil, err
	}

	// Get the scan and the detector config
	scan := &protos.ScanItem{}
	err = ReadOne(dbCollections.ScansName, bson.M{"_id": scanId}, scan, minimalSvcs.MongoDB)
	if err != nil {
		return "", nil, err
	}

	if len(scan.InstrumentConfig) <= 0 {
		return "", nil, fmt.Errorf("Scan %v has no instrument config", scanId)
	}

	detectorConfig, _ /*versions*/, err := piquant.ReadConfig(scan.InstrumentConfig, minimalSvcs)
	if err != nil {
		return "", nil, err
	}

	if expr.SourceLanguage != "LUA" {
		return "", nil, fmt.Errorf("Error: Expression %v is not Lua", expressionId)
	}

	allSource := ""

	// Read built-in modules
	// builtInModules := []string{"./built-in-modules/Map.lua", "./built-in-modules/DebugHelp.lua"}
	// for _, modPath := range builtInModules {
	// 	modSrcFile, err := os.ReadFile(modPath)
	// 	if err != nil {
	// 		return "", err
	// 	}
	builtInModules := []string{debugModule, mapModule}
	for _, modSrcFile := range builtInModules {
		modSrc := snipReturnModuleLine(string(modSrcFile))
		allSource = allSource + "\n" + modSrc + "\n"
	}

	// Read modules
	for _, modRef := range expr.ModuleReferences {
		_, modVer, err := readModule(modRef.ModuleId, modRef.Version, minimalSvcs.MongoDB)
		if err != nil {
			return "", nil, err
		}

		modSrc := snipReturnModuleLine(modVer.SourceCode)
		allSource = allSource + "\n" + modSrc + "\n"
	}

	allSource = applyLocalsAndTweaks(allSource+expr.SourceCode, detectorConfig.ElevAngle, scanId, quantId, userId)

	return allSource, expr, nil
}

func applyLocalsAndTweaks(rawSource string, elevAngle float32, scanId string, quantId string, userId string) string {
	// Add constants as required to Lua
	makeMapLuaCache := `local lastMap = {}
local function makeMap(value)
	-- If we have one saved, just set the value in the same kind of map
	if #lastMap > 0 then
		local values = {}
		for k, v in ipairs(lastMap[2]) do
			if v == nil then
				values[k] = nil
			else
				values[k] = value
			end
		end
		return { lastMap[1], values }
	end

	local m = makeMapRaw(value)
-- Cache it
lastMap = m
	return m
end
`
	source := fmt.Sprintf(`local elevAngle = %v
local quantId = "%v"
local scanId = "%v"
local maxSpectrumChannel = %v
local instrument = "%v"
local userId = "%v"
%v
`, elevAngle,
		quantId,
		scanId,
		4096,
		"PIXL_FM",
		userId,
		makeMapLuaCache) + rawSource

	// Replace table.unpack with unpack because gopher-lua is 5.1, table.unpack came in 5.2 but they're the same thing apparently
	source = strings.ReplaceAll(source, "table.unpack(", "unpack(")
	source = strings.ReplaceAll(source, "getmetatable(obj) == Estimate", "obj.typeIsEstimate and obj.typeIsEstimate == true")
	// Due to the above, we need to specify this
	source = strings.ReplaceAll(source, "local estimate = {}", "local estimate = {typeIsEstimate = true}")

	return source
}

func snipReturnModuleLine(src string) string {
	pos := strings.LastIndex(src, "return ")
	if pos > -1 {
		return src[0:pos]
	}
	return src
}

func ReadOne[T any](collectionName string, filter bson.M, intoItem *T, db *mongo.Database) error {
	ctx := context.TODO()
	coll := db.Collection(collectionName)

	dbResult := coll.FindOne(ctx, filter, options.FindOne())
	if dbResult.Err() != nil {
		return dbResult.Err()
	}

	return dbResult.Decode(intoItem)
}

func readModule(moduleId string, version *protos.SemanticVersion, db *mongo.Database) (*protos.DataModuleDB, *protos.DataModuleVersionDB, error) {
	mod := &protos.DataModuleDB{}
	err := ReadOne(dbCollections.ModulesName, bson.M{"_id": moduleId}, mod, db)

	if err != nil {
		return nil, nil, err
	}

	modVer := &protos.DataModuleVersionDB{}
	filter := bson.M{"_id": fmt.Sprintf("%v-v%v.%v.%v", moduleId, version.Major, version.Minor, version.Patch)}
	err = ReadOne(dbCollections.ModuleVersionsName, filter, modVer, db)

	if err != nil {
		return nil, nil, err
	}

	return mod, modVer, nil
}
