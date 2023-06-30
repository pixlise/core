package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleViewStateReq(req *protos.ViewStateReq, hctx wsHelpers.HandlerContext) (*protos.ViewStateResp, error) {
    return nil, errors.New("HandleViewStateReq not implemented yet")
}
func HandleViewStateItemWriteReq(req *protos.ViewStateItemWriteReq, hctx wsHelpers.HandlerContext) (*protos.ViewStateItemWriteResp, error) {
    return nil, errors.New("HandleViewStateItemWriteReq not implemented yet")
}
