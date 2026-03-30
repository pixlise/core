package wsHandler

import (
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleScanBeamLocationsReq(req *protos.ScanBeamLocationsReq, hctx wsHelpers.HandlerContext) (*protos.ScanBeamLocationsResp, error) {
	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}

	beams := scan.ReadXYZ(exprPB, indexes)

	return &protos.ScanBeamLocationsResp{
		BeamLocations: beams,
	}, nil
}
