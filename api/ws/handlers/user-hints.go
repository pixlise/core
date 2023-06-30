package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleUserDismissHintReq(req *protos.UserDismissHintReq, hctx wsHelpers.HandlerContext) (*protos.UserDismissHintResp, error) {
    return nil, errors.New("HandleUserDismissHintReq not implemented yet")
}
func HandleUserHintsReq(req *protos.UserHintsReq, hctx wsHelpers.HandlerContext) (*protos.UserHintsResp, error) {
    return nil, errors.New("HandleUserHintsReq not implemented yet")
}
func HandleUserHintsToggleReq(req *protos.UserHintsToggleReq, hctx wsHelpers.HandlerContext) (*protos.UserHintsToggleResp, error) {
    return nil, errors.New("HandleUserHintsToggleReq not implemented yet")
}
