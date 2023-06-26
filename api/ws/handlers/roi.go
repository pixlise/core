package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleRegionOfInterestReq(req *protos.RegionOfInterestReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.RegionOfInterestResp, error) {
    return nil, errors.New("HandleRegionOfInterestReq not implemented yet")
}
func HandleRegionOfInterestDeleteReq(req *protos.RegionOfInterestDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.RegionOfInterestDeleteResp, error) {
    return nil, errors.New("HandleRegionOfInterestDeleteReq not implemented yet")
}
func HandleRegionOfInterestListReq(req *protos.RegionOfInterestListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.RegionOfInterestListResp, error) {
    return nil, errors.New("HandleRegionOfInterestListReq not implemented yet")
}
func HandleRegionOfInterestWriteReq(req *protos.RegionOfInterestWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.RegionOfInterestWriteResp, error) {
    return nil, errors.New("HandleRegionOfInterestWriteReq not implemented yet")
}
