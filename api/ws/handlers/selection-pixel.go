package wsHandler

import (
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleSelectedImagePixelsReq(req *protos.SelectedImagePixelsReq, hctx wsHelpers.HandlerContext) (*protos.SelectedImagePixelsResp, error) {
	idxs, err := readSelection("pix_"+req.Image+"_"+hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return &protos.SelectedImagePixelsResp{
		PixelIndexes: idxs,
	}, nil
}

func HandleSelectedImagePixelsWriteReq(req *protos.SelectedImagePixelsWriteReq, hctx wsHelpers.HandlerContext) (*protos.SelectedImagePixelsWriteResp, error) {
	err := writeSelection("pix_"+req.Image+"_"+hctx.SessUser.User.Id, req.PixelIndexes, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return &protos.SelectedImagePixelsWriteResp{}, nil
}
