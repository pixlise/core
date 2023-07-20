package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleScanBeamImageLocationsReq(req *protos.ScanBeamImageLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ScanBeamImageLocationsResp, error) {
	return nil, errors.New("HandleScanBeamImageLocationsReq not implemented yet")
}

func HandleScanBeamLocationsReq(req *protos.ScanBeamLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ScanBeamLocationsResp, error) {
	exprPB, startLocIdx, endLocIdx, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}

	beams := []*protos.Coordinate3D{}
	for c := startLocIdx; c < endLocIdx; c++ {
		loc := exprPB.Locations[c]

		var beamSave *protos.Coordinate3D

		if loc.Beam != nil {
			beamSave = &protos.Coordinate3D{
				X: loc.Beam.X,
				Y: loc.Beam.Y,
				Z: loc.Beam.Z,
			}
		}

		beams = append(beams, beamSave)
	}

	return &protos.ScanBeamLocationsResp{
		BeamLocations: beams,
	}, nil
}
