package wsHandler

import (
	"fmt"
	"strconv"

	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandlePseudoIntensityReq(req *protos.PseudoIntensityReq, hctx wsHelpers.HandlerContext) (*protos.PseudoIntensityResp, error) {
	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)

	if err != nil {
		return nil, err
	}

	// Read labels first
	labels := []string{}
	for _, item := range exprPB.PseudoIntensityRanges {
		labels = append(labels, item.Name)
	}

	// Send back pseudo-intensities for the indexes requested
	tooManyPseudosLocations := 0
	pseudoIntensities := []*protos.PseudoIntensityData{}
	for _, c := range indexes {
		loc := exprPB.Locations[c]
		pseudos := loc.PseudoIntensities

		pmc, err := strconv.Atoi(loc.Id)
		if err != nil {
			return nil, fmt.Errorf("Failed to convert PMC %v to int while reading scan location %v", loc.Id, c)
		}

		// There really should only be one item!
		if len(pseudos) > 1 {
			tooManyPseudosLocations++
		} else if len(pseudos) == 1 {
			// Just read the first one
			pseudoIntensities = append(pseudoIntensities, &protos.PseudoIntensityData{
				Id:          uint32(pmc),
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
