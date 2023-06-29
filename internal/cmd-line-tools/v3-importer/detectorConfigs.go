package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/core/fileaccess"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func migrateDetectorConfigs(configBucket string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.DetectorConfigsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	detectorConfigPaths, err := fs.ListObjects(configBucket, filepaths.RootDetectorConfig)
	if err != nil {
		log.Fatal(err)
	}

	// This one is directly compatible with the protobuf-defined struct!
	destCfgs := []interface{}{}

	for _, p := range detectorConfigPaths {
		if strings.HasSuffix(p, "pixlise-config.json") {
			// Config name is one back in file path
			name := filepath.Base(filepath.Dir(p))

			cfg := protos.DetectorConfig{}
			err = fs.ReadJSON(configBucket, p, &cfg, false)
			if err != nil {
				return err
			}

			cfg.Id = name

			destCfgs = append(destCfgs, cfg)
		}
	}

	result, err := coll.InsertMany(context.TODO(), destCfgs)
	if err != nil {
		return err
	}

	fmt.Printf("Detector configs inserted: %v\n", len(result.InsertedIDs))
	return nil
}
