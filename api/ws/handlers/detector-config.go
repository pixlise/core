package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleDetectorConfigReq(req *protos.DetectorConfigReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.DetectorConfigResp, error) {
    return nil, errors.New("HandleDetectorConfigReq not implemented yet")
}
