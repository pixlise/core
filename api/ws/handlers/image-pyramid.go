package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v4/generated-protos"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
)

func HandleImagePyramidGetReq(req *protos.ImagePyramidGetReq, hctx wsHelpers.HandlerContext) (*protos.ImagePyramidGetResp, error) {
    return nil, errors.New("HandleImagePyramidGetReq not implemented yet")
}
func HandleImageTileDataGetReq(req *protos.ImageTileDataGetReq, hctx wsHelpers.HandlerContext) (*protos.ImageTileDataGetResp, error) {
    return nil, errors.New("HandleImageTileDataGetReq not implemented yet")
}
func HandleImageTileStructureGetReq(req *protos.ImageTileStructureGetReq, hctx wsHelpers.HandlerContext) (*protos.ImageTileStructureGetResp, error) {
    return nil, errors.New("HandleImageTileStructureGetReq not implemented yet")
}
