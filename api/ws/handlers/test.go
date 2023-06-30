package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleRunTestReq(req *protos.RunTestReq, hctx wsHelpers.HandlerContext) (*protos.RunTestResp, error) {
    return nil, errors.New("HandleRunTestReq not implemented yet")
}
