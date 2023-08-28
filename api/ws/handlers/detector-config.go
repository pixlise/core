package wsHandler

import (
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/piquant"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleDetectorConfigReq(req *protos.DetectorConfigReq, hctx wsHelpers.HandlerContext) (*protos.DetectorConfigResp, error) {
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	cfg, err := piquant.GetDetectorConfig(req.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return &protos.DetectorConfigResp{
		Config: cfg,
	}, nil
}
