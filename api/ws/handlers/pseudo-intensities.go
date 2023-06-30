package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandlePseudoIntensityReq(req *protos.PseudoIntensityReq, hctx wsHelpers.HandlerContext) (*protos.PseudoIntensityResp, error) {
    return nil, errors.New("HandlePseudoIntensityReq not implemented yet")
}
