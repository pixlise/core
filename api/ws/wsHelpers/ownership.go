package wsHelpers

import (
	"context"
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
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
		ObjectType:     objectType,
		CreatorUserId:  sessUser.User.Id,
		CreatedUnixSec: ts,
		//Viewers: ,
		Editors: &protos.UserGroupList{
			UserIds: []string{sessUser.User.Id},
		},
	}

	return ownerItem, nil
}

// Checks object access - if requireEdit is true, it checks for edit access
// otherwise just checks for view access. Returns an error if it failed to determine
// or if access is not granted, returns error formed with MakeUnauthorisedError
func CheckObjectAccess(requireEdit bool, objectId string, objectType protos.ObjectType, s *melody.Session, svcs *services.APIServices) (*protos.OwnershipItem, error) {
	result := svcs.MongoDB.Collection(dbCollections.OwnershipName).FindOne(context.TODO(), bson.M{"_id": objectId})
	if result.Err() != nil {
		return nil, fmt.Errorf("Failed to determine object access permissions for id: %v, type: %v. Error was: %v", objectId, objectType.String(), result.Err())
	}

	// Read it in & check
	ownership := &protos.OwnershipItem{}
	err := result.Decode(ownership)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode object access data for id: %v, type: %v. Error was: %v", objectId, objectType.String(), result.Err())
	}

	// Now check permissions
	connectingUser, err := GetSessionUser(s)
	if err != nil {
		return nil, fmt.Errorf("Failed to get session user permissions for id: %v, type: %v. Error was: %v", objectId, objectType.String(), err)
	}

	accessType := "View"
	toCheck := ownership.Viewers
	if requireEdit {
		toCheck = ownership.Editors
		accessType = "Edit"
	}

	// First check user id
	if utils.StringInSlice(connectingUser.User.Id, toCheck.UserIds) {
		return ownership, nil // User has access
	} else {
		// Check groups
		for _, groupId := range connectingUser.MemberOfGroupIds {
			if utils.StringInSlice(groupId, toCheck.GroupIds) {
				return ownership, nil // User has access via group it belongs to
			}
		}
	}

	// Access denied
	return nil, errorwithstatus.MakeUnauthorisedError(fmt.Errorf("%v access denied for: %v", accessType, objectId))
}
