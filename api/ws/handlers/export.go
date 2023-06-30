package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
)

func HandleExportFilesReq(req *protos.ExportFilesReq, hctx wsHelpers.HandlerContext) (*protos.ExportFilesResp, error) {
    return nil, errors.New("HandleExportFilesReq not implemented yet")
}
