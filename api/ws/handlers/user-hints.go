package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleUserDismissHintReq(req *protos.UserDismissHintReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserDismissHintResp, error) {
    return nil, errors.New("HandleUserDismissHintReq not implemented yet")
}
func HandleUserHintsReq(req *protos.UserHintsReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserHintsResp, error) {
    return nil, errors.New("HandleUserHintsReq not implemented yet")
}
func HandleUserHintsToggleReq(req *protos.UserHintsToggleReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserHintsToggleResp, error) {
    return nil, errors.New("HandleUserHintsToggleReq not implemented yet")
}
