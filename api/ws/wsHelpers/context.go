package wsHelpers

import (
	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/services"
)

type HandlerContext struct {
	Session  *melody.Session
	SessUser SessionUser
	Melody   *melody.Melody
	Svcs     *services.APIServices
}
