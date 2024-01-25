package main

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/fileaccess"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

func migrateDetectorConfigs(configBucket string, destConfigBucket string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.DetectorConfigsName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	detectorConfigPaths, err := fs.ListObjects(configBucket, filepaths.RootDetectorConfig)
	if err != nil {
		fatalError(err)
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

			destCfgs = append(destCfgs, &cfg)
		}
	}

	result, err := coll.InsertMany(context.TODO(), destCfgs)
	if err != nil {
		return err
	}

	fmt.Printf("Detector configs inserted: %v\n", len(result.InsertedIDs))

	// Copy to the destination bucket too
	failOnError := make([]bool, len(detectorConfigPaths))
	for c := range failOnError {
		failOnError[c] = true
	}

	s3Copy(fs, configBucket, detectorConfigPaths, destConfigBucket, detectorConfigPaths, failOnError)
	fmt.Printf("%v detector config files copied to dest bucket: %v\n", len(detectorConfigPaths), destConfigBucket)

	return nil
}
