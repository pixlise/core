package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleTagCreateReq(req *protos.TagCreateReq, hctx wsHelpers.HandlerContext) (*protos.TagCreateResp, error) {
	return nil, errors.New("HandleTagCreateReq not implemented yet")
}

func HandleTagDeleteReq(req *protos.TagDeleteReq, hctx wsHelpers.HandlerContext) (*protos.TagDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.TagDeleteResp](req.TagId, protos.ObjectType_OT_TAG, dbCollections.TagsName, hctx)
}

func HandleTagListReq(req *protos.TagListReq, hctx wsHelpers.HandlerContext) (*protos.TagListResp, error) {
	return nil, errors.New("HandleTagListReq not implemented yet")
}
