package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleExpressionGetReq(req *protos.ExpressionGetReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGetResp, error) {
	return nil, errors.New("HandleExpressionGetReq not implemented yet")
}

func HandleExpressionDeleteReq(req *protos.ExpressionDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.ExpressionDeleteResp](req.Id, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
}

func HandleExpressionListReq(req *protos.ExpressionListReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionListResp, error) {
	return nil, errors.New("HandleExpressionListReq not implemented yet")
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
