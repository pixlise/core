package expressionrunner

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services/servicesMock"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

/*func Example_expressionrunner_RunExpression_Simple() {
	err := runSource("print(\"hello\")")
	fmt.Printf("%v\n", err)

	// Output:
	// hello
	// <nil>
}*/

func Example_expressionrunner_RunExpression_Expression_Naltsos() {
	exprId := "u59sahioy18frfl9"
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

	runExpressionTest(scanId, quantId, exprId, modIds, modVers, false)

	// Output:
	// RunExpession error: <nil>
	// Got map size 121
	// Returned map matches expected output from PIXLISE
}

func Example_expressionrunner_RunExpression_Expression_Naltsos_UseMemoisation() {
	exprId := "u59sahioy18frfl9"
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

	runExpressionTest(scanId, quantId, exprId, modIds, modVers, true)

	// Output:
	// RunExpession error: <nil>
	// Got map size 121
	// Returned map matches expected output from PIXLISE
}

func Example_expressionrunner_RunExpression_Expression_CastleGeyser() {
	exprId := "u59sahioy18frfl9"
	scanId := "393871873"
	quantId := "quant-pvkostn8a2u6j7cj"

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

	runExpressionTest(scanId, quantId, exprId, modIds, modVers, false)

	// Output:
	// RunExpession error: <nil>
	// Got map size 3333
	// Returned map matches expected output from PIXLISE
}

func Example_expressionrunner_RunExpression_Expression_CastleGeyser_UseMemoisation() {
	exprId := "u59sahioy18frfl9"
	scanId := "393871873"
	quantId := "quant-pvkostn8a2u6j7cj"

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

	runExpressionTest(scanId, quantId, exprId, modIds, modVers, true)

	// Output:
	// RunExpession error: <nil>
	// Got map size 3333
	// Returned map matches expected output from PIXLISE
}

func runExpressionTest(scanId, quantId, exprId string, modIds, modVers []string, seedMemo bool) {
	idGen := idgen.MockIDGenerator{
		IDs: []string{"id123"},
	}
	logLev := logger.LogDebug
	svcs := servicesMock.MakeMockSvcsWithFS("./test-files/", &idGen, &logLev)
	ts := []int64{}
	for c := 0; c < 30; c++ {
		ts = append(ts, int64(1783596971+c))
	}
	svcs.TimeStamper = &timestamper.MockTimeNowStamper{QueuedTimeStamps: ts}

	svcs.MongoDB = wstestlib.GetDBWithEnvironment("backendexpr")

	err := svcs.MongoDB.Drop(context.TODO())
	if err != nil {
		log.Fatal(err)
	}

	// Seed DB with scan info
	err = seedDB(scanId, fmt.Sprintf("./test-files/scan/%v.json", scanId), dbCollections.ScansName, &protos.ScanItem{}, svcs.MongoDB)
	if err != nil {
		log.Fatal(err)
	}
	err = seedDB("PIXL", "./test-files/scan/PIXL.json", dbCollections.DetectorConfigsName, &protos.DetectorConfig{}, svcs.MongoDB)
	if err != nil {
		log.Fatal(err)
	}

	// Seed DB with expression stuff
	err = seedDB(exprId, fmt.Sprintf("./test-files/expr/%v.json", exprId), dbCollections.ExpressionsName, &protos.DataExpression{}, svcs.MongoDB)
	if err != nil {
		log.Fatal(err)
	}

	for c, modId := range modIds {
		err = seedDB(modId, fmt.Sprintf("./test-files/module/%v.json", modId), dbCollections.ModulesName, &protos.DataModuleDB{}, svcs.MongoDB)
		if err != nil {
			log.Fatal(err)
		}

		err = seedDB(modId+"-"+modVers[c], fmt.Sprintf("./test-files/modVer/%v-%v.json", modId, modVers[c]), dbCollections.ModuleVersionsName, &protos.DataModuleVersionDB{}, svcs.MongoDB)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Seed quant
	err = seedDB(quantId, fmt.Sprintf("./test-files/quant/%v.json", quantId), dbCollections.QuantificationsName, &protos.QuantificationSummary{}, svcs.MongoDB)
	if err != nil {
		log.Fatal(err)
	}

	if seedMemo {
		fs := fileaccess.FSAccess{}
		memPath := "./test-files/memoisation"
		files, err := fs.ListObjects(memPath, "")
		if err != nil {
			log.Fatal(err)
		}

		for _, f := range files {
			if strings.Contains(f, scanId) {
				// Read this into our memoisation table
				err = seedDB(f[0:len(f)-5] /*.json*/, filepath.Join(memPath, f), dbCollections.MemoisedItemsName, &protos.MemoisedItem{}, svcs.MongoDB)
				if err != nil {
					log.Fatal(err)
				}
			}
		}
	}

	m, _, _, err := RunExpression(exprId, scanId, quantId, &svcs, true, false)
	sz := 0
	if m != nil {
		sz = len(m.Values)
	}
	fmt.Printf("RunExpession error: %v\n", err)

	if err == nil {
		fmt.Printf("Got map size %v\n", sz)

		// Compare with the CSV we got from PIXLISE
		err = compareMapWithCSV(m, fmt.Sprintf("./test-files/PIXLISE_output_%v.csv", scanId))
		if err != nil {
			fmt.Printf("Failed to match output to expected: %v", err)
		} else {
			fmt.Printf("Returned map matches expected output from PIXLISE")
		}
	}
}

func compareMapWithCSV(m *PMCDataValues, csvPath string) error {
	csvFile, err := os.Open(csvPath)
	if err != nil {
		return fmt.Errorf("ReadExpectedCSV: %v\n", err)
	}

	r := csv.NewReader(csvFile)
	eM, err := r.ReadAll()
	if err != nil {
		return fmt.Errorf("ReadExpectedCSV2: %v\n", err)
	}

	for c, line := range eM {
		if len(line) != 2 {
			return fmt.Errorf("CSV line %v has wrong number of fields", c)
		}

		pmc, err := strconv.Atoi(line[0])
		if err != nil {
			return fmt.Errorf("CSV line %v has bad pmc: %v", c, line[0])
		}
		value, err := strconv.ParseFloat(line[1], 64)
		if err != nil {
			return fmt.Errorf("CSV line %v has bad value: %v", c, line[1])
		}

		if m.Values[c].PMC != pmc {
			return fmt.Errorf("Expression output item %v PMC mismatch, expected %v, got %v", c, line[0], m.Values[c].PMC)
		}

		if math.Abs(m.Values[c].Value-value) > 0.000001 {
			return fmt.Errorf("Expression output item %v value mismatch, expected %v, got %v", c, line[1], m.Values[c].Value)
		}
	}

	return nil
}

func seedDB[T any](id string, jsonPath string, collName string, item *T, db *mongo.Database) error {
	f, err := os.ReadFile(jsonPath)
	if err != nil {
		return err
	}

	json.Unmarshal([]byte(f), item)

	ctx := context.TODO()
	_, err = db.Collection(collName).UpdateByID(ctx, id, bson.D{{Key: "$set", Value: item}}, options.Update().SetUpsert(true))
	if err != nil {
		log.Fatal(err)
	}

	return nil
}
