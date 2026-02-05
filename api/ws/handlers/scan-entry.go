package wsHandler

import (
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/scan"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleScanEntryReq(req *protos.ScanEntryReq, hctx wsHelpers.HandlerContext) (*protos.ScanEntryResp, error) {
	exprPB, indexes, err := beginDatasetFileReqForRange(req.ScanId, req.Entries, hctx)
	if err != nil {
		return nil, err
	}

	entries, err := scan.ReadScanEntries(exprPB, indexes)
	if err != nil {
		return nil, err
	}

	return &protos.ScanEntryResp{
		Entries: entries,
	}, nil
}
