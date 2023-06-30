package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleSpectrumReq(req *protos.SpectrumReq, hctx wsHelpers.HandlerContext) (*protos.SpectrumResp, error) {
    return nil, errors.New("HandleSpectrumReq not implemented yet")
}
