package ws

import (
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func makeRespStatus(err error) protos.ResponseStatus {
	switch e := err.(type) {
	case errorwithstatus.Error:
		// Here we're expecting to be given the protos.ResponseStatus values, but if it's not one of these
		// we say WS_SERVER_ERROR
		status := e.Status()
		if status <= int(protos.ResponseStatus_WS_UNDEFINED.Number()) || status > int(protos.ResponseStatus_WS_SERVER_ERROR.Number()) {
			return protos.ResponseStatus_WS_SERVER_ERROR
		}
		return protos.ResponseStatus(e.Status())
	default:
		return protos.ResponseStatus_WS_SERVER_ERROR
	}
}
