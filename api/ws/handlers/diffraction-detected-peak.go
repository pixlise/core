package wsHandler

import (
	"fmt"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
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

	diffPerLoc, err := wsHelpers.GetDetectedDiffractionPeaks(req.Entries.Indexes, exprPB, diffRawData)
	if err != nil {
		return nil, err
	}

	return &protos.DetectedDiffractionPeaksResp{
		PeaksPerLocation: diffPerLoc,
	}, nil
}
