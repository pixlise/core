package wsHandler

import (
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleScanBeamLocationsReq(req *protos.ScanBeamLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ScanBeamLocationsResp, error) {
	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}

	beams := []*protos.Coordinate3D{}
	for _, c := range indexes {
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
