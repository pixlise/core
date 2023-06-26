package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleImageDeleteReq(req *protos.ImageDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ImageDeleteResp, error) {
    return nil, errors.New("HandleImageDeleteReq not implemented yet")
}
func HandleImageListReq(req *protos.ImageListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ImageListResp, error) {
    return nil, errors.New("HandleImageListReq not implemented yet")
}
func HandleImageSetDefaultReq(req *protos.ImageSetDefaultReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ImageSetDefaultResp, error) {
    return nil, errors.New("HandleImageSetDefaultReq not implemented yet")
}
func HandleImageUploadReq(req *protos.ImageUploadReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ImageUploadResp, error) {
    return nil, errors.New("HandleImageUploadReq not implemented yet")
}
