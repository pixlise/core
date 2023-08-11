package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleQuantListReq(req *protos.QuantListReq, hctx wsHelpers.HandlerContext) (*protos.QuantListResp, error) {
	filter, _, err := wsHelpers.MakeFilter(req.SearchParams, false, protos.ObjectType_OT_QUANTIFICATION, hctx)
	if err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.QuantificationsName)

	opts := options.Find()

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.QuantificationSummary{}
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}

	quants := []*protos.QuantificationSummary{}
	for _, item := range items {
		quants = append(quants, item)
	}

	return &protos.QuantListResp{
		Quants: quants,
	}, nil
}

func HandleQuantGetReq(req *protos.QuantGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantGetResp, error) {
	dbItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
	if err != nil {
		return nil, err
	}

	// TODO: something with owner? Should we add it to the outgoing item?

	// If they want data too, retrieve it
	var quant *protos.Quantification
	if !req.SummaryOnly {
		quantPath := filepaths.GetQuantPath(hctx.SessUser.User.Id, dbItem.Params.Params.DatasetID, req.QuantId+".bin")
		quant, err = wsHelpers.ReadQuantificationFile(req.QuantId, quantPath, hctx.Svcs)
		if err != nil {
			return nil, err
		}
	}

	return &protos.QuantGetResp{
		Summary: dbItem,
		Data:    quant,
	}, nil
}

func HandleQuantLastOutputGetReq(req *protos.QuantLastOutputGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantLastOutputGetResp, error) {
	return nil, errors.New("HandleQuantLastOutputGetReq not implemented yet")
}
