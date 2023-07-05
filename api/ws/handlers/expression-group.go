package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleExpressionGroupDeleteReq(req *protos.ExpressionGroupDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.ExpressionGroupDeleteResp](req.Id, protos.ObjectType_OT_EXPRESSION_GROUP, dbCollections.ExpressionGroupsName, hctx)
}

func HandleExpressionGroupListReq(req *protos.ExpressionGroupListReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupListResp, error) {
	return nil, errors.New("HandleExpressionGroupListReq not implemented yet")
}

func HandleExpressionGroupGetReq(req *protos.ExpressionGroupGetReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ExpressionGroup](false, req.Id, protos.ObjectType_OT_EXPRESSION_GROUP, dbCollections.ExpressionGroupsName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB)
	return &protos.ExpressionGroupGetResp{
		Group: dbItem,
	}, nil
}

func HandleExpressionGroupWriteReq(req *protos.ExpressionGroupWriteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupWriteResp, error) {
	return nil, errors.New("HandleExpressionGroupSetReq not implemented yet")
}
