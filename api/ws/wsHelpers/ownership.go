package wsHelpers

import (
	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
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

func MakeOwnerForWrite(objectId string, objectType protos.ObjectType, s *melody.Session, svcs *services.APIServices) (*protos.OwnershipItem, error) {
	sessUser, err := GetSessionUser(s)
	if err != nil {
		return nil, err
	}

	ts := uint64(svcs.TimeStamper.GetTimeNowSec())

	ownerId := svcs.IDGen.GenObjectID()
	ownerItem := &protos.OwnershipItem{
		Id:             ownerId,
		ObjectId:       objectId,
		ObjectType:     objectType,
		CreatorUserId:  sessUser.UserID,
		CreatedUnixSec: ts,
		//Viewers: ,
		Editors: &protos.UserGroupList{
			UserIds: []string{sessUser.UserID},
		},
	}

	return ownerItem, nil
}
