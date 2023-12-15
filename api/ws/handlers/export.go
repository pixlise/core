package wsHandler

import (
	"errors"
	"fmt"
	"os"
	"path"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func HandleExportFilesReq(req *protos.ExportFilesReq, hctx wsHelpers.HandlerContext) (*protos.ExportFilesResp, error) {
	if len(req.ExportTypes) <= 0 {
		return nil, errors.New("No export types specified")
	}

	// For now we only allow exporting one thing...
	if len(req.ExportTypes) != 1 || req.ExportTypes[0] != protos.ExportDataType_EDT_QUANT_CSV {
		return nil, errors.New("Only one export type allowed: QUANT_CSV")
	}

	zipRoot := path.Join(os.TempDir(), "export-"+utils.RandStringBytesMaskImpr(8))
	err := os.MkdirAll(zipRoot, os.ModePerm)

	if err != nil {
		return nil, err
	}

	for _, expType := range req.ExportTypes {
		if expType == protos.ExportDataType_EDT_QUANT_CSV {
			// Read from DB
			dbItem, _, err := wsHelpers.GetUserObjectById[protos.QuantificationSummary](false, req.QuantId, protos.ObjectType_OT_QUANTIFICATION, dbCollections.QuantificationsName, hctx)
			if err != nil {
				return nil, err
			}

			// Read the quant CSV that should already be there
			quantFileName := req.QuantId + ".csv"
			quantPath := path.Join(dbItem.Status.OutputFilePath, quantFileName)
			fileBytes, err := hctx.Svcs.FS.ReadObject(hctx.Svcs.Config.UsersBucket, quantPath)
			if err != nil {
				// Doesn't seem to exist?
				if hctx.Svcs.FS.IsNotFoundError(err) {
					return nil, errorwithstatus.MakeNotFoundError(req.QuantId)
				}

				hctx.Svcs.Log.Errorf("Failed to load quant data for %v, from: s3://%v/%v, error was: %v.", req.QuantId, hctx.Svcs.Config.UsersBucket, quantPath, err)
				return nil, err
			}

			// Return this in a zip
			quantWritePath := path.Join(zipRoot, quantFileName)
			err = os.WriteFile(quantWritePath, fileBytes, os.ModePerm)

			if err != nil {
				return nil, fmt.Errorf("Failed to write %v to export. Error: %v", quantFileName, err)
			}
		}
	}

	zipBytes, err := utils.ZipDirectory(zipRoot)
	if err != nil {
		return nil, fmt.Errorf("Failed to create zip of export data. Error: %v", err)
	}

	return &protos.ExportFilesResp{ZipData: zipBytes}, nil
}
