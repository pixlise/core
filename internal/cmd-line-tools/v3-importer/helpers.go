package main

import (
	"strings"

	protos "github.com/pixlise/core/v3/generated-protos"
)

func convertOwnership(origin SrcAPIObjectItem) *protos.Ownership {
	userId := origin.Creator.UserID
	if !strings.HasPrefix(userId, "auth0|") {
		userId = "auth0|" + userId
	}
	return &protos.Ownership{
		Creator: &protos.UserInfo{
			Id: userId,
			// Name - Not sent to DB!
			// Email - Not sent to DB!
			// IconURL - Not sent to DB!
		},
		CreatedUnixSec:  uint64(origin.CreatedUnixTimeSec),
		ModifiedUnixSec: uint64(origin.ModifiedUnixTimeSec),
	}
}
