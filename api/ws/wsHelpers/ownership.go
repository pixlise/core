package wsHelpers

import (
	"errors"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
)

// TODO: maybe we can pass in some generic thing that has an owner field?
/*

type HasOwnerField interface {
	Owner *protos.Ownership
}
func MakeOwnerForWrite(writable HasOwnerField, s *melody.Session, svcs *services.APIServices) (*protos.Ownership, error) {
	if writable.Owner != nil {
*/

func MakeOwnerForWrite(reqOwnerField *protos.Ownership, s *melody.Session, svcs *services.APIServices) (*protos.Ownership, error) {
	if reqOwnerField != nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Owner must be empty for write messages"))
	}

	sessUser, err := GetSessionUser(s)
	if err != nil {
		return nil, err
	}

	ts := uint64(svcs.TimeStamper.GetTimeNowSec())

	return &protos.Ownership{
		Creator: &protos.UserInfo{
			Id: sessUser.UserID,
		},
		CreatedUnixSec:  ts,
		ModifiedUnixSec: ts,
	}, nil
}
