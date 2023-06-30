package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleGetOwnershipReq(req *protos.GetOwnershipReq, hctx wsHelpers.HandlerContext) (*protos.GetOwnershipResp, error) {
    return nil, errors.New("HandleGetOwnershipReq not implemented yet")
}
func HandleObjectEditAccessReq(req *protos.ObjectEditAccessReq, hctx wsHelpers.HandlerContext) (*protos.ObjectEditAccessResp, error) {
    return nil, errors.New("HandleObjectEditAccessReq not implemented yet")
}
