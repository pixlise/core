package wsHandler

import (
	"errors"
	"fmt"
	"path"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/filepaths"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleQuantBlessReq(req *protos.QuantBlessReq, hctx wsHelpers.HandlerContext) (*protos.QuantBlessResp, error) {
	return nil, errors.New("HandleQuantBlessReq not implemented yet")
}

func HandleQuantDeleteReq(req *protos.QuantDeleteReq, hctx wsHelpers.HandlerContext) (*protos.QuantDeleteResp, error) {
	// Can't use the helper: DeleteUserObject because we also have to delete stuff from S3 and we need the scanId associated to find
	// the path. So here we do a get first, with edit priviledges required
	dbItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](true, req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
	if err != nil {
		return nil, err
	}

	// We have the item, form the paths that need to be deleted
	toDelete := []string{
		filepaths.GetQuantPath(hctx.SessUser.User.Id, dbItem.Params.Params.DatasetID, dbItem.Id+".bin"),
		filepaths.GetQuantPath(hctx.SessUser.User.Id, dbItem.Params.Params.DatasetID, dbItem.Id+".csv"),
	}

	// Add all the known log files too
	for _, logFile := range dbItem.Status.PiquantLogs {
		toDelete = append(toDelete, filepaths.GetQuantPath(hctx.SessUser.User.Id, dbItem.Params.Params.DatasetID, path.Join(filepaths.MakeQuantLogDirName(dbItem.Id), logFile)))
	}

	// Now delete from DB
	if _, err := wsHelpers.DeleteUserObject[protos.QuantDeleteResp](req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx); err != nil {
		return nil, err
	}

	// Delete all the files
	errors := []string{}
	for _, delFile := range toDelete {
		err := hctx.Svcs.FS.DeleteObject(hctx.Svcs.Config.UsersBucket, delFile)
		if err != nil {
			errors = append(errors, fmt.Sprintf("s3://%v/%v: %v", hctx.Svcs.Config.UsersBucket, delFile, err))
		}
	}

	if len(errors) > 0 {
		// Print out all errors, but don't error out if something remains in S3
		hctx.Svcs.Log.Errorf("Failed to delete files from S3 when deleting quant: %v. File errors:\n%v", req.QuantId, strings.Join(errors, "\n"))
	}

	return &protos.QuantDeleteResp{}, nil
}

func HandleQuantPublishReq(req *protos.QuantPublishReq, hctx wsHelpers.HandlerContext) (*protos.QuantPublishResp, error) {
	return nil, errors.New("HandleQuantPublishReq not implemented yet")
}
