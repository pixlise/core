package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleScanLocationReq(req *protos.ScanLocationReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ScanLocationResp, error) {
    return nil, errors.New("HandleScanLocationReq not implemented yet")
}
