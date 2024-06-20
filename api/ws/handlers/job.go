package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v4/generated-protos"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
)

func HandleJobListReq(req *protos.JobListReq, hctx wsHelpers.HandlerContext) (*protos.JobListResp, error) {
    return nil, errors.New("HandleJobListReq not implemented yet")
}
