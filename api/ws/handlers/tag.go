package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleTagCreateReq(req *protos.TagCreateReq, hctx wsHelpers.HandlerContext) (*protos.TagCreateResp, error) {
    return nil, errors.New("HandleTagCreateReq not implemented yet")
}
func HandleTagDeleteReq(req *protos.TagDeleteReq, hctx wsHelpers.HandlerContext) (*protos.TagDeleteResp, error) {
    return nil, errors.New("HandleTagDeleteReq not implemented yet")
}
func HandleTagListReq(req *protos.TagListReq, hctx wsHelpers.HandlerContext) (*protos.TagListResp, error) {
    return nil, errors.New("HandleTagListReq not implemented yet")
}
