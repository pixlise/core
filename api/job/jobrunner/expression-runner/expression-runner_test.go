package expressionrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services/servicesMock"
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

func Example_expressionrunner_RunExpression_Expression() {
	exprId := "u59sahioy18frfl9"
	scanId := "048300551"
	quantId := "quant-ggy6zxhn23p7rlv9"

	idGen := idgen.MockIDGenerator{
		IDs: []string{"id123"},
	}
	logLev := logger.LogDebug
	svcs := servicesMock.MakeMockSvcsWithFS("./test-files/", &idGen, &logLev)
	ts := []int64{}
	for c := 0; c < 100; c++ {
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

	modIds := []string{"idc2d7xifmbpqk8o", "ng46r8vwzr3z28ui", "f6hrn69g5tuyiq3m", "yg7o9dkue0orim26"}
	modVers := []string{"v1.3.0", "v0.8.0", "v0.33.0", "v3.5.5"}

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

	err = RunExpression(exprId, scanId, quantId, &svcs)
	fmt.Printf("%v\n", err)

	// Output:
	// hello
	// <nil>
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
