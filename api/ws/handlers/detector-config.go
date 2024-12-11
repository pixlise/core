package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/piquant"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleDetectorConfigReq(req *protos.DetectorConfigReq, hctx wsHelpers.HandlerContext) (*protos.DetectorConfigResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, 255); err != nil {
		return nil, err
	}

	cfg, err := piquant.GetDetectorConfig(req.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	// Read versions
	versions := piquant.GetPiquantConfigVersions(hctx.Svcs, req.Id)
	if len(versions) <= 0 {
		return nil, fmt.Errorf("DetectorConfig %v has no versions defined", req.Id)
	}

	latestVersion := versions[len(versions)-1]

	// Read PIQUANT config file
	piquantCfg, err := piquant.GetPIQUANTConfig(hctx.Svcs, req.Id, latestVersion)
	if err != nil {
		return nil, err
	}

	// Retrieve elevAngle
	cfgPath := filepaths.GetDetectorConfigPath(req.Id, latestVersion, piquantCfg.ConfigFile)
	piquantCfgFile, err := hctx.Svcs.FS.ReadObject(hctx.Svcs.Config.ConfigBucket, cfgPath)
	if err != nil {
		return nil, err
	}

	// Find the value
	piquantCfgFileStr := string(piquantCfgFile)
	angle, err := piquant.ReadFieldFromPIQUANTConfigMSA(piquantCfgFileStr, "#ELEVANGLE")
	if err != nil {
		hctx.Svcs.Log.Errorf("Failed to read ELEVANGLE from Piquant config file: %v, trying emerg_angle", cfgPath)

		// EM config has a value "emerg_angle" which is also set to 70, maybe it's an interchangeable name?
		angle, err = piquant.ReadFieldFromPIQUANTConfigMSA(piquantCfgFileStr, "emerg_angle")
		if err != nil {
			return nil, fmt.Errorf("Failed to read ELEVANGLE and emerg_angle from Piquant config file: %v", cfgPath)
		}
	}

	cfg.ElevAngle = angle

	return &protos.DetectorConfigResp{
		Config:                cfg,
		PiquantConfigVersions: versions,
	}, nil
}

func HandleDetectorConfigListReq(req *protos.DetectorConfigListReq, hctx wsHelpers.HandlerContext) (*protos.DetectorConfigListResp, error) {
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DetectorConfigsName)

	filter := bson.D{}
	opts := options.Find().SetProjection(bson.D{
		{Key: "id", Value: true},
	})
	cursor, err := coll.Find(context.TODO(), filter, opts)

	if err != nil {
		return nil, err
	}

	configs := []*protos.DetectorConfig{}
	err = cursor.All(context.TODO(), &configs)
	if err != nil {
		return nil, err
	}

	configList := []string{}
	for _, cfg := range configs {
		configList = append(configList, cfg.Id)
	}

	return &protos.DetectorConfigListResp{
		Configs: configList,
	}, nil
}
