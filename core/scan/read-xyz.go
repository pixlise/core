package scan

import protos "github.com/pixlise/core/v4/generated-protos"

func ReadXYZ(exprPB *protos.Experiment, indexes []uint32) []*protos.Coordinate3D {
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

	return beams
}
