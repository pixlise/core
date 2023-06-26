package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandlePseudoIntensityReq(req *protos.PseudoIntensityReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.PseudoIntensityResp, error) {
    return nil, errors.New("HandlePseudoIntensityReq not implemented yet")
}
