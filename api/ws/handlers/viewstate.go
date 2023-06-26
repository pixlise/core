package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleViewStateReq(req *protos.ViewStateReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ViewStateResp, error) {
    return nil, errors.New("HandleViewStateReq not implemented yet")
}
func HandleViewStateItemWriteReq(req *protos.ViewStateItemWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ViewStateItemWriteResp, error) {
    return nil, errors.New("HandleViewStateItemWriteReq not implemented yet")
}
