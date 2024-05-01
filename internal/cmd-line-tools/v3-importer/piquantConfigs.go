package main

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
)

type SrcPiquantVersionConfig struct {
	Version            string      `json:"version"`
	ChangedUnixTimeSec int64       `json:"changedUnixTimeSec"`
	Creator            SrcUserInfo `json:"creator"`
}

// Config contains the docker container to use for PIQUANT. Separate from config.json because users can configure this in UI
const piquantVersionFileName = "piquant-version.json"

func migratePiquantVersion(configBucket string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	coll := dest.Collection(dbCollections.PiquantVersionName)
	err := coll.Drop(context.TODO())
	if err != nil {
		return err
	}

	ver := SrcPiquantVersionConfig{}
	err = fs.ReadJSON(configBucket, filepaths.GetConfigFilePath(piquantVersionFileName), &ver, false)
	if err != nil {
		return err
	}

	outVer := &protos.PiquantVersion{
		Id:              "current",
		Version:         ver.Version,
		ModifiedUnixSec: uint32(ver.ChangedUnixTimeSec),
		ModifierUserId:  utils.FixUserId(ver.Creator.UserID),
	}
	_, err = coll.InsertOne(context.TODO(), outVer)
	if err != nil {
		return err
	}

	fmt.Printf("Piquant version written: %v\n", ver.Version)
	return nil
}

/* Decided to leave PIQUANT configs in S3 because that way PIQUANT docker container has authenticated direct access
func migratePiquantConfigs(configBucket string, fs fileaccess.FileAccess, dest *mongo.Database) error {
	const collectionName = "piquantConfigs"

	err := dest.Collection(collectionName).Drop(context.TODO())
	if err != nil {
		return err
	}

	piquantConfigPaths, err := fs.ListObjects(configBucket, filepaths.RootDetectorConfig)
	if err != nil {
		fatalError(err)
	}

	// This one is directly compatible with the protobuf-defined struct!
	destCfgs := []interface{}{}

	for _, p := range piquantConfigPaths {
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

	result, err := dest.Collection(collectionName).InsertMany(context.TODO(), destCfgs)
	if err != nil {
		return err
	}

	fmt.Printf("Piquant configs inserted: %v\n", len(result.InsertedIDs))
	return nil
}
*/
