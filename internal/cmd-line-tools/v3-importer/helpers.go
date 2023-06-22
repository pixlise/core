package main

import protos "github.com/pixlise/core/v3/generated-protos"

func convertOwnership(origin SrcAPIObjectItem) *protos.Ownership {
	return &protos.Ownership{
		Creator: &protos.UserInfo{
			Id: origin.Creator.Email,
			// Name - Not sent to DB!
			// Email - Not sent to DB!
			// IconURL - Not sent to DB!
		},
		CreatedUnixSec:  uint64(origin.CreatedUnixTimeSec),
		ModifiedUnixSec: uint64(origin.ModifiedUnixTimeSec),
	}
}
