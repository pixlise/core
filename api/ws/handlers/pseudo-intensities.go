package wsHandler

import (
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandlePseudoIntensityReq(req *protos.PseudoIntensityReq, hctx wsHelpers.HandlerContext) (*protos.PseudoIntensityResp, error) {
	exprPB, startLocIdx, endLocIdx, err := beginDatasetFileReq(req.ScanId, req.StartingLocation, req.LocationCount, hctx)
	if err != nil {
		return nil, err
	}

	// Read labels first
	labels := []string{}
	for _, item := range exprPB.PseudoIntensityRanges {
		labels = append(labels, item.Name)
	}

	tooManyPseudosLocations := 0
	pseudoIntensities := []*protos.PseudoIntensityData{}
	for c := startLocIdx; c < endLocIdx; c++ {
		pseudos := exprPB.Locations[c].PseudoIntensities

		// There really should only be one item!
		if len(pseudos) > 1 {
			tooManyPseudosLocations++
		} else if len(pseudos) == 1 {
			// Just read the first one
			pseudoIntensities = append(pseudoIntensities, &protos.PseudoIntensityData{
				Intensities: pseudos[0].ElementIntensities,
			})
		}
	}

	if tooManyPseudosLocations > 0 {
		hctx.Svcs.Log.Errorf("Reading pseudointensities for scan %v: found more than 1 set of pseudos in %v PMCs", req.ScanId, tooManyPseudosLocations)
	}

	return &protos.PseudoIntensityResp{
		IntensityLabels: labels,
		Data:            pseudoIntensities,
	}, nil
}
