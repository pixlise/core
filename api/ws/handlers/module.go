package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleDataModuleReq(req *protos.DataModuleReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.DataModuleResp, error) {
    return nil, errors.New("HandleDataModuleReq not implemented yet")
}
func HandleDataModuleListReq(req *protos.DataModuleListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.DataModuleListResp, error) {
    return nil, errors.New("HandleDataModuleListReq not implemented yet")
}
func HandleDataModuleWriteReq(req *protos.DataModuleWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.DataModuleWriteResp, error) {
    return nil, errors.New("HandleDataModuleWriteReq not implemented yet")
}
