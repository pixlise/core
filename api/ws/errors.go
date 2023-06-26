package ws

import (
	"net/http"

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
			// Try to map errors we'd send back via HTTP
			if status == http.StatusNotFound {
				return protos.ResponseStatus_WS_NOT_FOUND
			} else if status == http.StatusBadRequest {
				return protos.ResponseStatus_WS_BAD_REQUEST
			} else if status == http.StatusUnauthorized {
				return protos.ResponseStatus_WS_NO_PERMISSION
			} else {
				return protos.ResponseStatus_WS_SERVER_ERROR
			}
		}
		return protos.ResponseStatus(status)
	default:
		return protos.ResponseStatus_WS_SERVER_ERROR
	}
}
