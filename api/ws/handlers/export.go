package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleExportFilesReq(req *protos.ExportFilesReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExportFilesResp, error) {
    return nil, errors.New("HandleExportFilesReq not implemented yet")
}
