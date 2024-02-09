package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v4/generated-protos"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
)

func HandleSysNotificationReq(req *protos.SysNotificationReq, hctx wsHelpers.HandlerContext) (*protos.SysNotificationResp, error) {
    return nil, errors.New("HandleSysNotificationReq not implemented yet")
}
