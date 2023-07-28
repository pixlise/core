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

func HandleExpressionGetReq(req *protos.ExpressionGetReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGetResp, error) {
	return nil, errors.New("HandleExpressionGetReq not implemented yet")
}

func HandleExpressionDeleteReq(req *protos.ExpressionDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.ExpressionDeleteResp](req.Id, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
}

func HandleExpressionListReq(req *protos.ExpressionListReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_EXPRESSION, hctx)
	if err != nil {
		return nil, err
	}

	ids := utils.GetMapKeys(idToOwner)

	filter := bson.M{"_id": bson.M{"$in": ids}}
	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionsName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.DataExpression{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Transform to map of output values
	itemMap := map[string]*protos.DataExpression{}
	for _, item := range items {
		if owner, ok := idToOwner[item.Id]; ok {
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}
		itemMap[item.Id] = item
	}

	return &protos.ExpressionListResp{
		Expressions: itemMap,
	}, nil
}
func HandleExpressionWriteReq(req *protos.ExpressionWriteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionWriteResp, error) {
	return nil, errors.New("HandleExpressionWriteReq not implemented yet")
}
func HandleExpressionWriteExecStatReq(req *protos.ExpressionWriteExecStatReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionWriteExecStatResp, error) {
	return nil, errors.New("HandleExpressionWriteExecStatReq not implemented yet")
}
func HandleExpressionWriteResultReq(req *protos.ExpressionWriteResultReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionWriteResultResp, error) {
	return nil, errors.New("HandleExpressionWriteResultReq not implemented yet")
}
