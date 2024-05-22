package wsHandler

import (
	"fmt"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/indexcompression"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleDetectedDiffractionPeaksReq(req *protos.DetectedDiffractionPeaksReq, hctx wsHelpers.HandlerContext) (*protos.DetectedDiffractionPeaksResp, error) {
	// Because we're dealing in entry indexes (relative to the scan), we download the dataset.bin file here too
	// to get the totals, and to look up PMCs from diffraction DB
	exprPB, err := beginDatasetFileReq(req.ScanId, hctx)
	if err != nil {
		return nil, err
	}

	if req.Entries == nil {
		return nil, fmt.Errorf("no entry range specified for scan %v", req.ScanId)
	}

	// Cache the file locally, like we do with datasets (aka Scans)
	diffRawData, err := wsHelpers.ReadDiffractionFile(req.ScanId, hctx.Svcs)
	if err != nil {
		return nil, err
	}

	// Form a PMC->diffraction peaks lookup
	diffLookup := map[string]*protos.Diffraction_Location{}
	for _, loc := range diffRawData.Locations {
		diffLookup[loc.Id] = loc
	}

	// Decode the range
	indexes, err := indexcompression.DecodeIndexList(req.Entries.Indexes, len(exprPB.Locations))
	if err != nil {
		return nil, err
	}

	diffPerLoc := []*protos.DetectedDiffractionPerLocation{}
	for _, c := range indexes {
		exprLoc := exprPB.Locations[c]

		if loc, ok := diffLookup[exprLoc.Id]; ok {
			peaks := []*protos.DetectedDiffractionPerLocation_DetectedDiffractionPeak{}

			for _, locPeak := range loc.Peaks {
				peaks = append(peaks, &protos.DetectedDiffractionPerLocation_DetectedDiffractionPeak{
					PeakChannel:       locPeak.PeakChannel,
					EffectSize:        locPeak.EffectSize,
					BaselineVariation: locPeak.BaselineVariation,
					GlobalDifference:  locPeak.GlobalDifference,
					DifferenceSigma:   locPeak.DifferenceSigma,
					PeakHeight:        locPeak.PeakHeight,
					Detector:          locPeak.Detector,
				})
			}

			diffPerLoc = append(diffPerLoc, &protos.DetectedDiffractionPerLocation{
				Id:    loc.Id,
				Peaks: peaks,
			})
		}
	}

	return &protos.DetectedDiffractionPeaksResp{
		PeaksPerLocation: diffPerLoc,
	}, nil
}
