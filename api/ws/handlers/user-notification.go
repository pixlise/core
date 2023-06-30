package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleSendUserNotificationReq(req *protos.SendUserNotificationReq, hctx wsHelpers.HandlerContext) (*protos.SendUserNotificationResp, error) {
    return nil, errors.New("HandleSendUserNotificationReq not implemented yet")
}
func HandleUserNotificationReq(req *protos.UserNotificationReq, hctx wsHelpers.HandlerContext) (*protos.UserNotificationResp, error) {
    return nil, errors.New("HandleUserNotificationReq not implemented yet")
}
