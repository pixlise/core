package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleUserNotificationSettingsReq(req *protos.UserNotificationSettingsReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserNotificationSettingsResp, error) {
    return nil, errors.New("HandleUserNotificationSettingsReq not implemented yet")
}
func HandleUserNotificationSettingsWriteReq(req *protos.UserNotificationSettingsWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserNotificationSettingsWriteResp, error) {
    return nil, errors.New("HandleUserNotificationSettingsWriteReq not implemented yet")
}
