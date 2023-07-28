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

func HandleExpressionGroupDeleteReq(req *protos.ExpressionGroupDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.ExpressionGroupDeleteResp](req.Id, protos.ObjectType_OT_EXPRESSION_GROUP, dbCollections.ExpressionGroupsName, hctx)
}

func HandleExpressionGroupListReq(req *protos.ExpressionGroupListReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_EXPRESSION_GROUP, hctx)
	if err != nil {
		return nil, err
	}

	ids := utils.GetMapKeys(idToOwner)

	filter := bson.M{"_id": bson.M{"$in": ids}}
	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionGroupsName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ExpressionGroup{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Transform to map of output values
	itemMap := map[string]*protos.ExpressionGroup{}
	for _, item := range items {
		if owner, ok := idToOwner[item.Id]; ok {
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}
		itemMap[item.Id] = item
	}

	return &protos.ExpressionGroupListResp{
		Groups: itemMap,
	}, nil
}

func HandleExpressionGroupGetReq(req *protos.ExpressionGroupGetReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ExpressionGroup](false, req.Id, protos.ObjectType_OT_EXPRESSION_GROUP, dbCollections.ExpressionGroupsName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return &protos.ExpressionGroupGetResp{
		Group: dbItem,
	}, nil
}

func HandleExpressionGroupWriteReq(req *protos.ExpressionGroupWriteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupWriteResp, error) {
	return nil, errors.New("HandleExpressionGroupSetReq not implemented yet")
}
