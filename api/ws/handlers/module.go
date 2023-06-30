package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleDataModuleReq(req *protos.DataModuleReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleResp, error) {
    return nil, errors.New("HandleDataModuleReq not implemented yet")
}
func HandleDataModuleListReq(req *protos.DataModuleListReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleListResp, error) {
    return nil, errors.New("HandleDataModuleListReq not implemented yet")
}
func HandleDataModuleWriteReq(req *protos.DataModuleWriteReq, hctx wsHelpers.HandlerContext) (*protos.DataModuleWriteResp, error) {
    return nil, errors.New("HandleDataModuleWriteReq not implemented yet")
}
