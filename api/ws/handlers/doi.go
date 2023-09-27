package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandlePublishExpressionToZenodoReq(req *protos.PublishExpressionToZenodoReq, hctx wsHelpers.HandlerContext) (*protos.PublishExpressionToZenodoResp, error) {
	return nil, errors.New("HandlePublishExpressionToZenodoReq not implemented yet")
}
func HandleZenodoDOIGetReq(req *protos.ZenodoDOIGetReq, hctx wsHelpers.HandlerContext) (*protos.ZenodoDOIGetResp, error) {
	return nil, errors.New("HandleZenodoDOIGetReq not implemented yet")
}
