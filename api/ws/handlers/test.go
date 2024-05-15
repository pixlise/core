package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleRunTestReq(req *protos.RunTestReq, hctx wsHelpers.HandlerContext) ([]*protos.RunTestResp, error) {
	return nil, errors.New("HandleRunTestReq not implemented yet")
}
