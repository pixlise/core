package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleScanImageLocationsReq(req *protos.ScanImageLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ScanImageLocationsResp, error) {
    return nil, errors.New("HandleScanImageLocationsReq not implemented yet")
}
