package wsHandler

import (
	"errors"
	protos "github.com/pixlise/core/v4/generated-protos"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
)

func HandleBackupDBReq(req *protos.BackupDBReq, hctx wsHelpers.HandlerContext) (*protos.BackupDBResp, error) {
    return nil, errors.New("HandleBackupDBReq not implemented yet")
}
func HandleDBAdminConfigGetReq(req *protos.DBAdminConfigGetReq, hctx wsHelpers.HandlerContext) (*protos.DBAdminConfigGetResp, error) {
    return nil, errors.New("HandleDBAdminConfigGetReq not implemented yet")
}
func HandleRestoreDBReq(req *protos.RestoreDBReq, hctx wsHelpers.HandlerContext) (*protos.RestoreDBResp, error) {
    return nil, errors.New("HandleRestoreDBReq not implemented yet")
}
