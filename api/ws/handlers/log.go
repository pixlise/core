package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleLogGetLevelReq(req *protos.LogGetLevelReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.LogGetLevelResp, error) {
    return nil, errors.New("HandleLogGetLevelReq not implemented yet")
}
func HandleLogReadReq(req *protos.LogReadReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.LogReadResp, error) {
    return nil, errors.New("HandleLogReadReq not implemented yet")
}
func HandleLogSetLevelReq(req *protos.LogSetLevelReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.LogSetLevelResp, error) {
    return nil, errors.New("HandleLogSetLevelReq not implemented yet")
}
