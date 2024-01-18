package wsHandler

import (
	"errors"
	"path"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func HandleQuantListReq(req *protos.QuantListReq, hctx wsHelpers.HandlerContext) (*protos.QuantListResp, error) {
	items, idToOwner, err := quantification.ListUserQuants(req.SearchParams, hctx)
	if err != nil {
		return nil, err
	}

	quants := []*protos.QuantificationSummary{}
	for _, item := range items {
		if owner, ok := idToOwner[item.Id]; ok {
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}
		quants = append(quants, item)
	}

	return &protos.QuantListResp{
		Quants: quants,
	}, nil
}

func HandleQuantGetReq(req *protos.QuantGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantGetResp, error) {
	dbItem, ownerItem, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
	if err != nil {
		return nil, err
	}

	// TODO: something with owner? Should we add it to the outgoing item?

	// If they want data too, retrieve it
	var quant *protos.Quantification
	if !req.SummaryOnly {
		//quantPath := filepaths.GetQuantPath(hctx.SessUser.User.Id, dbItem.Params.Params.DatasetID, req.QuantId+".bin")
		quantPath := path.Join(dbItem.Status.OutputFilePath, req.QuantId+".bin")
		quant, err = wsHelpers.ReadQuantificationFile(req.QuantId, quantPath, hctx.Svcs)
		if err != nil {
			return nil, err
		}
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(ownerItem, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)

	return &protos.QuantGetResp{
		Summary: dbItem,
		Data:    quant,
	}, nil
}

func HandleQuantLastOutputGetReq(req *protos.QuantLastOutputGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantLastOutputGetResp, error) {
	return nil, errors.New("HandleQuantLastOutputGetReq not implemented yet")
}
