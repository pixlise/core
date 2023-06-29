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
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
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
func CheckObjectAccess(requireEdit bool, objectId string, objectType protos.ObjectType, s *melody.Session, db *mongo.Database) (*protos.OwnershipItem, error) {
	result := db.Collection(dbCollections.OwnershipName).FindOne(context.TODO(), bson.M{"_id": objectId})
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

	// For editing, we only look at the editor list
	accessType := "Edit"
	toCheck := []*protos.UserGroupList{ownership.Editors}
	if !requireEdit {
		// If we're interested in view permissions, editors have implicit view permissions
		// so we just add the viewer list here
		toCheck = append(toCheck, ownership.Viewers)
		accessType = "View"
	}

	for _, toCheckItem := range toCheck {
		// First check user id
		if utils.StringInSlice(connectingUser.User.Id, toCheckItem.UserIds) {
			return ownership, nil // User has access
		} else {
			// Check groups
			for _, groupId := range connectingUser.MemberOfGroupIds {
				if utils.StringInSlice(groupId, toCheckItem.GroupIds) {
					return ownership, nil // User has access via group it belongs to
				}
			}
		}
	}

	// Access denied
	return nil, errorwithstatus.MakeUnauthorisedError(fmt.Errorf("%v access denied for: %v", accessType, objectId))
}

// Gets all object IDs which the user has access to - if requireEdit is true, it checks for edit access
// otherwise just checks for view access
// Returns a map of object id->creator user id
func ListAccessibleIDs(requireEdit bool, objectType protos.ObjectType, s *melody.Session, db *mongo.Database) (map[string]string, error) {
	filter := bson.D{
		{"$and", []interface{}{
			bson.D{{"objecttype", 2}},
			bson.D{{"editors.userids", "auth0|5de45d85ca40070f421a3a34"}},
		}},
	}

	result := map[string]string{}

	opts := options.Find() //.SetProjection(bson.D{{"_id", true}})
	cursor, err := db.Collection(dbCollections.OwnershipName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	//err = cursor.All(context.TODO(), &resultIds)
	items := []*protos.OwnershipItem{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		result[item.Id] = item.CreatorUserId
	}

	return result, nil
}

func MakeOwnerSummary(ownership *protos.OwnershipItem, db *mongo.Database) *protos.OwnershipSummary {
	user, err := getUser(ownership.CreatorUserId, db)
	result := &protos.OwnershipSummary{
		CreatedUnixSec: ownership.CreatedUnixSec,
	}
	if err == nil {
		result.CreatorUser = user
	} else {
		result.CreatorUser = &protos.UserInfo{
			Id: ownership.CreatorUserId,
		}
	}
	return result
}

func getUser(userId string, db *mongo.Database) (*protos.UserInfo, error) {
	filter := bson.M{"_id": userId}
	opts := options.FindOne().SetProjection(bson.D{{"info", true}})
	result := db.Collection(dbCollections.UsersName).FindOne(context.TODO(), filter, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	userDBItem := protos.UserDBItem{}
	err := result.Decode(&userDBItem)
	if err != nil {
		return nil, err
	}

	return userDBItem.Info, nil
}
