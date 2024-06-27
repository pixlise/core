package wsHandler

import (
	"fmt"
	"path"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/quantification"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
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
	if err := wsHelpers.CheckStringField(&req.QuantId, "QuantId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

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

	// We seem to have some old quants where the status struct says start time was 0, but there is another start time in quant params, so
	// substitute a non-zero value in this case
	if dbItem.Status.StartUnixTimeSec == 0 && dbItem.Params.StartUnixTimeSec > 0 {
		dbItem.Status.StartUnixTimeSec = dbItem.Params.StartUnixTimeSec
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
		if hctx.Svcs.FS.IsNotFoundError(err) {
			// Just return empty
			result = []byte{}
			err = nil
		} else {
			return nil, err
		}
	}

	return &protos.QuantLastOutputGetResp{Output: string(result)}, nil
}

func HandleQuantLogListReq(req *protos.QuantLogListReq, hctx wsHelpers.HandlerContext) (*protos.QuantLogListResp, error) {
	if err := wsHelpers.CheckStringField(&req.QuantId, "QuantId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Check that user has access to this quant
	dbItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
	if err != nil {
		return nil, err
	}

	logFilePaths, err := hctx.Svcs.FS.ListObjects(hctx.Svcs.Config.UsersBucket, path.Join(dbItem.Status.OutputFilePath, req.QuantId+"-logs")+"/")
	if err != nil {
		return nil, err
	}

	logFileNames := []string{}
	for _, logpath := range logFilePaths {
		logFileNames = append(logFileNames, path.Base(logpath))
	}

	return &protos.QuantLogListResp{
		FileNames: logFileNames,
	}, nil
}

func HandleQuantLogGetReq(req *protos.QuantLogGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantLogGetResp, error) {
	if err := wsHelpers.CheckStringField(&req.QuantId, "QuantId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.LogName, "LogName", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Check that user has access to this quant
	dbItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
	if err != nil {
		return nil, err
	}

	logPath := path.Join(dbItem.Status.OutputFilePath, req.QuantId+"-logs", req.LogName)
	logData, err := hctx.Svcs.FS.ReadObject(hctx.Svcs.Config.UsersBucket, logPath)
	if err != nil {
		if hctx.Svcs.FS.IsNotFoundError(err) {
			return nil, errorwithstatus.MakeNotFoundError(req.LogName)
		}
		return nil, err
	}

	return &protos.QuantLogGetResp{
		LogData: string(logData),
	}, nil
}

func HandleQuantRawDataGetReq(req *protos.QuantRawDataGetReq, hctx wsHelpers.HandlerContext) (*protos.QuantRawDataGetResp, error) {
	if err := wsHelpers.CheckStringField(&req.QuantId, "QuantId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Check that user has access to this quant
	dbItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
	if err != nil {
		return nil, err
	}
	//UserContent/5df311ed8a0b5d0ebf5fb476/089063943/Quantifications/
	// Read the CSV file from S3

	csvPath := path.Join(dbItem.Status.OutputFilePath, req.QuantId+".csv")
	csvData, err := hctx.Svcs.FS.ReadObject(hctx.Svcs.Config.UsersBucket, csvPath)
	if err != nil {
		if hctx.Svcs.FS.IsNotFoundError(err) {
			return nil, errorwithstatus.MakeNotFoundError(req.QuantId + ".csv")
		}
		return nil, err
	}

	return &protos.QuantRawDataGetResp{
		Data: string(csvData),
	}, nil
}
