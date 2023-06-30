package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleLogGetLevelReq(req *protos.LogGetLevelReq, hctx wsHelpers.HandlerContext) (*protos.LogGetLevelResp, error) {
    return nil, errors.New("HandleLogGetLevelReq not implemented yet")
}
func HandleLogReadReq(req *protos.LogReadReq, hctx wsHelpers.HandlerContext) (*protos.LogReadResp, error) {
    return nil, errors.New("HandleLogReadReq not implemented yet")
}
func HandleLogSetLevelReq(req *protos.LogSetLevelReq, hctx wsHelpers.HandlerContext) (*protos.LogSetLevelResp, error) {
    return nil, errors.New("HandleLogSetLevelReq not implemented yet")
}
