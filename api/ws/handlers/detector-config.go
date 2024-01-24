package wsHandler

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
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

	return &protos.DetectorConfigResp{
		Config:                cfg,
		PiquantConfigVersions: piquant.GetPiquantConfigVersions(hctx.Svcs, req.Id),
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
