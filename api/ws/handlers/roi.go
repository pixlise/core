package wsHandler

import (
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleRegionOfInterestGetReq(req *protos.RegionOfInterestGetReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ROIItem](false, req.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return &protos.RegionOfInterestGetResp{
		RegionOfInterest: dbItem,
	}, nil
}

func HandleRegionOfInterestDeleteReq(req *protos.RegionOfInterestDeleteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.RegionOfInterestDeleteResp](req.Id, protos.ObjectType_OT_ROI, dbCollections.RegionsOfInterestName, hctx)
}

func HandleRegionOfInterestListReq(req *protos.RegionOfInterestListReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestListResp, error) {
	return nil, errors.New("HandleRegionOfInterestListReq not implemented yet")
}
func HandleRegionOfInterestWriteReq(req *protos.RegionOfInterestWriteReq, hctx wsHelpers.HandlerContext) (*protos.RegionOfInterestWriteResp, error) {
	return nil, errors.New("HandleRegionOfInterestWriteReq not implemented yet")
}
