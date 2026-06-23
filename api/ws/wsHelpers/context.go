package wsHelpers

import (
	"github.com/olahol/melody"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/sessionuser"
)

type HandlerContext struct {
	Session  *melody.Session
	SessUser sessionuser.SessionUser
	Melody   *melody.Melody
	Svcs     *services.APIServices
}
