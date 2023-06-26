package wsHandler

import (
	protos "github.com/pixlise/core/v3/generated-protos"
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"errors"
)

func HandleExpressionGroupDeleteReq(req *protos.ExpressionGroupDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionGroupDeleteResp, error) {
    return nil, errors.New("HandleExpressionGroupDeleteReq not implemented yet")
}
func HandleExpressionGroupListReq(req *protos.ExpressionGroupListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionGroupListResp, error) {
    return nil, errors.New("HandleExpressionGroupListReq not implemented yet")
}
func HandleExpressionGroupSetReq(req *protos.ExpressionGroupSetReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ExpressionGroupSetResp, error) {
    return nil, errors.New("HandleExpressionGroupSetReq not implemented yet")
}
