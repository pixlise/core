package expressionrunner

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/pixlise/core/v4/api/dbCollections"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func SeedDB[T any](id string, jsonPath string, collName string, item *T, db *mongo.Database) error {
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

func ReadOne[T any](collectionName string, filter bson.M, intoItem *T, db *mongo.Database) error {
	ctx := context.TODO()
	coll := db.Collection(collectionName)

	dbResult := coll.FindOne(ctx, filter, options.FindOne())
	if dbResult.Err() != nil {
		return fmt.Errorf("Failed to read %v from collection %v: %v", filter, collectionName, dbResult.Err())
	}

	return dbResult.Decode(intoItem)
}

func SeedDBForExpressionTest(jsonRoot, scanId, quantId, exprId string, modIds, modVers []string, db *mongo.Database) {
	// Seed DB with scan info
	err := SeedDB(scanId, filepath.Join(jsonRoot, "scan", scanId+".json"), dbCollections.ScansName, &protos.ScanItem{}, db)
	if err != nil {
		log.Fatal(err)
	}
	err = SeedDB("PIXL", filepath.Join(jsonRoot, "scan", "PIXL.json"), dbCollections.DetectorConfigsName, &protos.DetectorConfig{}, db)
	if err != nil {
		log.Fatal(err)
	}

	// Seed DB with expression stuff
	err = SeedDB(exprId, filepath.Join(jsonRoot, "expr", exprId+".json"), dbCollections.ExpressionsName, &protos.DataExpression{}, db)
	if err != nil {
		log.Fatal(err)
	}

	for c, modId := range modIds {
		err = SeedDB(modId, filepath.Join(jsonRoot, "module", modId+".json"), dbCollections.ModulesName, &protos.DataModuleDB{}, db)
		if err != nil {
			log.Fatal(err)
		}

		err = SeedDB(modId+"-"+modVers[c], filepath.Join(jsonRoot, "modVer", fmt.Sprintf("%v-%v.json", modId, modVers[c])), dbCollections.ModuleVersionsName, &protos.DataModuleVersionDB{}, db)
		if err != nil {
			log.Fatal(err)
		}
	}

	// Seed quant
	err = SeedDB(quantId, filepath.Join(jsonRoot, "quant", quantId+".json"), dbCollections.QuantificationsName, &protos.QuantificationSummary{}, db)
	if err != nil {
		log.Fatal(err)
	}
}
