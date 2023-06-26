package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleExpressionReq(req *protos.ExpressionReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionResp, error) {
    return nil, errors.New("HandleExpressionReq not implemented yet")
}
func HandleExpressionDeleteReq(req *protos.ExpressionDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionDeleteResp, error) {
    return nil, errors.New("HandleExpressionDeleteReq not implemented yet")
}
func HandleExpressionListReq(req *protos.ExpressionListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionListResp, error) {
    return nil, errors.New("HandleExpressionListReq not implemented yet")
}
func HandleExpressionWriteReq(req *protos.ExpressionWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionWriteResp, error) {
    return nil, errors.New("HandleExpressionWriteReq not implemented yet")
}
func HandleExpressionWriteExecStatReq(req *protos.ExpressionWriteExecStatReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionWriteExecStatResp, error) {
    return nil, errors.New("HandleExpressionWriteExecStatReq not implemented yet")
}
func HandleExpressionWriteResultReq(req *protos.ExpressionWriteResultReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionWriteResultResp, error) {
    return nil, errors.New("HandleExpressionWriteResultReq not implemented yet")
}
