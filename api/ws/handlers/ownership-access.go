package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleGetOwnershipReq(req *protos.GetOwnershipReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.GetOwnershipResp, error) {
    return nil, errors.New("HandleGetOwnershipReq not implemented yet")
}
func HandleObjectEditAccessReq(req *protos.ObjectEditAccessReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ObjectEditAccessResp, error) {
    return nil, errors.New("HandleObjectEditAccessReq not implemented yet")
}
