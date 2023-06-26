package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleScanImageLocationsReq(req *protos.ScanImageLocationsReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ScanImageLocationsResp, error) {
    return nil, errors.New("HandleScanImageLocationsReq not implemented yet")
}
