package wsHandler

import (
	"github.com/pixlise/core/v3/api/coreg"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleImportMarsViewerImageReq(req *protos.ImportMarsViewerImageReq, hctx wsHelpers.HandlerContext) (*protos.ImportMarsViewerImageResp, error) {
	jobId := ""
	var err error

	jobId, err = coreg.StartCoregImport(req.TriggerUrl, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.ImportMarsViewerImageResp{
		JobId: jobId,
	}, nil
}
