package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleSendUserNotificationReq(req *protos.SendUserNotificationReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.SendUserNotificationResp, error) {
    return nil, errors.New("HandleSendUserNotificationReq not implemented yet")
}
func HandleUserNotificationReq(req *protos.UserNotificationReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserNotificationResp, error) {
    return nil, errors.New("HandleUserNotificationReq not implemented yet")
}
