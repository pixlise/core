package wsHandler

import (
	"fmt"
	"path"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
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
	if err := wsHelpers.CheckStringField(&req.ScanId, "ScanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	if req.PiquantCommand != "quant" {
		return nil, fmt.Errorf("PiquantCommand must be quant") // for now!
	}

	if req.OutputType != protos.QuantOutputType_QO_DATA && req.OutputType != protos.QuantOutputType_QO_LOG {
		return nil, fmt.Errorf("Invalid OutputType")
	}

	// Get the file name
	fileName := ""
	if req.OutputType == protos.QuantOutputType_QO_DATA {
		fileName = filepaths.QuantLastOutputFileName + ".csv" // quant only supplies this
	} else {
		fileName = filepaths.QuantLastOutputLogName
	}

	// Get the path to stream
	s3Path := filepaths.GetUserLastPiquantOutputPath(hctx.SessUser.User.Id, req.ScanId, req.PiquantCommand, fileName)

	result, err := hctx.Svcs.FS.ReadObject(hctx.Svcs.Config.UsersBucket, s3Path)
	if err != nil {
		return nil, err
	}

	return &protos.QuantLastOutputGetResp{Output: string(result)}, nil

}
