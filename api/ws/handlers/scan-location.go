package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleScanLocationReq(req *protos.ScanLocationReq, hctx wsHelpers.HandlerContext) (*protos.ScanLocationResp, error) {
    return nil, errors.New("HandleScanLocationReq not implemented yet")
}
