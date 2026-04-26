package wsHandler

import (
	"fmt"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleImportMarsViewerImageReq(req *protos.ImportMarsViewerImageReq, hctx wsHelpers.HandlerContext) (*protos.ImportMarsViewerImageResp, error) {
	return nil, fmt.Errorf("No longer implemented, coreg service was discontinued on MarsViewer side")
}
