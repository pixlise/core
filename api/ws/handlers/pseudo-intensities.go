package wsHandler

import (
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandlePseudoIntensityReq(req *protos.PseudoIntensityReq, hctx wsHelpers.HandlerContext) (*protos.PseudoIntensityResp, error) {
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, 50); err != nil {
		return nil, err
	}

	_, _, err := wsHelpers.GetUserObjectById[protos.ScanItem](false, req.ScanId, protos.ObjectType_OT_SCAN, dbCollections.ScansName, hctx)
	if err != nil {
		return nil, err
	}

	// We've come this far, we have access to the scan, so read it
	exprPB, err := wsHelpers.ReadDatasetFile(req.ScanId, hctx.Svcs)
	if err != nil {
		return nil, err
	}

	// Read labels first
	labels := []string{}
	for _, item := range exprPB.PseudoIntensityRanges {
		labels = append(labels, item.Name)
	}

	// Read the pseudo-intensities for the requested PMCs
	// NOTE: req.LocationCount == 0 is interpreted as ALL
	locLast := uint32(len(exprPB.Locations))
	if req.LocationCount > 0 {
		locLast = req.StartingLocation + req.LocationCount
	}

	tooManyPseudosLocations := 0
	pseudoIntensities := []*protos.PseudoIntensityData{}
	for c := req.StartingLocation; c < locLast; c++ {
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
