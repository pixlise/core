package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleDataModuleGetReq(req *protos.DataModuleGetReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleGetResp, error) {
	return nil, errors.New("HandleDataModuleGetReq not implemented yet")
}

func HandleDataModuleListReq(req *protos.DataModuleListReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_DATA_MODULE, hctx)
	if err != nil {
		return nil, err
	}

	ids := utils.GetMapKeys(idToOwner)

	filter := bson.M{"_id": bson.M{"$in": ids}}
	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ModulesName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.DataModule{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Transform to map of output values
	itemMap := map[string]*protos.DataModule{}
	for _, item := range items {
		if owner, ok := idToOwner[item.Id]; ok {
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}
		itemMap[item.Id] = item
	}

	return &protos.DataModuleListResp{
		Modules: itemMap,
	}, nil
}
func HandleDataModuleWriteReq(req *protos.DataModuleWriteReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleWriteResp, error) {
	return nil, errors.New("HandleDataModuleWriteReq not implemented yet")
}
