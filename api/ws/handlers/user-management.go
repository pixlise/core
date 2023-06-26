package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleUserAddRoleReq(req *protos.UserAddRoleReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserAddRoleResp, error) {
    return nil, errors.New("HandleUserAddRoleReq not implemented yet")
}
func HandleUserDeleteRoleReq(req *protos.UserDeleteRoleReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserDeleteRoleResp, error) {
    return nil, errors.New("HandleUserDeleteRoleReq not implemented yet")
}
func HandleUserListReq(req *protos.UserListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserListResp, error) {
    return nil, errors.New("HandleUserListReq not implemented yet")
}
func HandleUserRoleListReq(req *protos.UserRoleListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserRoleListResp, error) {
    return nil, errors.New("HandleUserRoleListReq not implemented yet")
}
func HandleUserRolesListReq(req *protos.UserRolesListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.UserRolesListResp, error) {
    return nil, errors.New("HandleUserRolesListReq not implemented yet")
}
