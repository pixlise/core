package wsHandler

import (
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleScanBeamImageLocationsReq(req *protos.ScanBeamImageLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ScanBeamImageLocationsResp, error) {
	if err := wsHelpers.CheckStringField(&req.Image, "Image", 1, 255); err != nil {
		return nil, err
	}

	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}

	// Find the coordinates for this image
	ijs := []*protos.Coordinate2D{}

	for imgIdx, img := range exprPB.AlignedContextImages {
		if img.Image == req.Image {
			// Found the image, return coordinates for this
			for _, c := range indexes {
				loc := exprPB.Locations[c]

				ij := &protos.Coordinate2D{}
				if imgIdx == 0 {
					ij.I = loc.Beam.ImageI
					ij.J = loc.Beam.ImageJ
				} else {
					ij.I = loc.Beam.ContextLocations[imgIdx-1].I
					ij.J = loc.Beam.ContextLocations[imgIdx-1].J
				}

				ijs = append(ijs, ij)
			}

			break
		}
	}

	return &protos.ScanBeamImageLocationsResp{
		BeamImageLocations: &protos.ImageLocations{
			ImageFileName: req.Image,
			Locations:     ijs,
		},
	}, nil
}

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
