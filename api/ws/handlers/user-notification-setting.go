package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleUserNotificationSettingsReq(req *protos.UserNotificationSettingsReq, hctx wsHelpers.HandlerContext) (*protos.UserNotificationSettingsResp, error) {
    return nil, errors.New("HandleUserNotificationSettingsReq not implemented yet")
}
func HandleUserNotificationSettingsWriteReq(req *protos.UserNotificationSettingsWriteReq, hctx wsHelpers.HandlerContext) (*protos.UserNotificationSettingsWriteResp, error) {
    return nil, errors.New("HandleUserNotificationSettingsWriteReq not implemented yet")
}
