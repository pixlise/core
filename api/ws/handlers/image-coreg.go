package wsHandler

import (
	"github.com/pixlise/core/v3/api/coreg"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleImportMarsViewerImageReq(req *protos.ImportMarsViewerImageReq, hctx wsHelpers.HandlerContext) (*protos.ImportMarsViewerImageResp, error) {
	jobId := ""
	var err error

	if len(req.TriggerUrl) > 0 {
		// We got triggered by URL, so start with that
	} else {
		jobId, err = coreg.StartCoregImport(req.MarsViewerExport, hctx)
		if err != nil {
			return nil, err
		}
	}

	return &protos.ImportMarsViewerImageResp{
		JobId: jobId,
	}, nil
}
