package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleSpectrumReq(req *protos.SpectrumReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.SpectrumResp, error) {
    return nil, errors.New("HandleSpectrumReq not implemented yet")
}
