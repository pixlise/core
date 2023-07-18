package wsHandler

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func HandleDetectorConfigReq(req *protos.DetectorConfigReq, hctx wsHelpers.HandlerContext) (*protos.DetectorConfigResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	coll := hctx.Svcs.MongoDB.Collection(dbCollections.DetectorConfigsName)

	result := coll.FindOne(context.TODO(), bson.M{"_id": req.Id})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.Id)
		}
		return nil, result.Err()
	}

	cfg := protos.DetectorConfig{}
	err := result.Decode(&cfg)
	if err != nil {
		return nil, err
	}

	return &protos.DetectorConfigResp{
		Config: &cfg,
	}, nil
}
