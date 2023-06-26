package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleTagCreateReq(req *protos.TagCreateReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.TagCreateResp, error) {
    return nil, errors.New("HandleTagCreateReq not implemented yet")
}
func HandleTagDeleteReq(req *protos.TagDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.TagDeleteResp, error) {
    return nil, errors.New("HandleTagDeleteReq not implemented yet")
}
func HandleTagListReq(req *protos.TagListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.TagListResp, error) {
    return nil, errors.New("HandleTagListReq not implemented yet")
}
