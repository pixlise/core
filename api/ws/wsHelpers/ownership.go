package wsHelpers

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/timestamper"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

// TODO: maybe we can pass in some generic thing that has an owner field?
// NO: we cannot. https://github.com/golang/go/issues/51259
/*

type HasOwnerField interface {
	Owner *protos.Ownership
}
func MakeOwnerForWrite(writable HasOwnerField, s *melody.Session, svcs *services.APIServices) (*protos.Ownership, error) {
	if writable.Owner != nil {
*/

func MakeOwnerForWrite(objectId string, objectType protos.ObjectType, hctx HandlerContext) (*protos.OwnershipItem, error) {
	ts := uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())

	ownerItem := &protos.OwnershipItem{
		Id:             objectId,
		ObjectType:     objectType,
		CreatorUserId:  hctx.SessUser.User.Id,
		CreatedUnixSec: ts,
		//Viewers: ,
		Editors: &protos.UserGroupList{
			UserIds: []string{hctx.SessUser.User.Id},
		},
	}

	return ownerItem, nil
}

// Checks object access - if requireEdit is true, it checks for edit access
// otherwise just checks for view access. Returns an error if it failed to determine
// or if access is not granted, returns error formed with MakeUnauthorisedError
func CheckObjectAccess(requireEdit bool, objectId string, objectType protos.ObjectType, hctx HandlerContext) (*protos.OwnershipItem, error) {
	return CheckObjectAccessForUser(requireEdit, objectId, objectType, hctx.SessUser.User.Id, hctx.SessUser.MemberOfGroupIds, hctx.Svcs.MongoDB)
}

func CheckObjectAccessForUser(requireEdit bool, objectId string, objectType protos.ObjectType, userId string, memberOfGroupIds []string, db *mongo.Database) (*protos.OwnershipItem, error) {
	ownerCollectionId := objectId
	if objectType == protos.ObjectType_OT_SCAN {
		ownerCollectionId = "scan_" + objectId
	}

	result := db.Collection(dbCollections.OwnershipName).FindOne(context.TODO(), bson.M{"_id": ownerCollectionId})
	if result.Err() != nil {
		// If the error is due to the item not existing, this isn't a permissions error, but likely the object doesn't
		// exist at all. No point going further, report this as a bad request right here
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(objectId)
		}

		return nil, fmt.Errorf("Failed to determine object access permissions for id: %v, type: %v. Error was: %v", objectId, objectType.String(), result.Err())
	}

	// Read it in & check
	ownership := &protos.OwnershipItem{}
	err := result.Decode(ownership)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode object access data for id: %v, type: %v. Error was: %v", objectId, objectType.String(), result.Err())
	}

	// Now check permissions
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
		if toCheckItem == nil {
			continue
		}

		// First check user id
		if toCheckItem.UserIds != nil && utils.ItemInSlice(userId, toCheckItem.UserIds) {
			return ownership, nil // User has access
		} else {
			// Check groups
			if toCheckItem.GroupIds != nil {
				for _, groupId := range memberOfGroupIds {
					if utils.ItemInSlice(groupId, toCheckItem.GroupIds) {
						return ownership, nil // User has access via group it belongs to
					}
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
func ListAccessibleIDs(requireEdit bool, objectType protos.ObjectType, hctx HandlerContext) (map[string]*protos.OwnershipItem, error) {
	idLookups := []interface{}{
		bson.D{{"editors.userids", hctx.SessUser.User.Id}},
	}
	if !requireEdit {
		idLookups = append(idLookups, bson.D{{"viewers.userids", hctx.SessUser.User.Id}})
	}

	// Add the group IDs
	for _, groupId := range hctx.SessUser.MemberOfGroupIds {
		idLookups = append(idLookups, bson.D{{"editors.groupids", groupId}})
		if !requireEdit {
			idLookups = append(idLookups, bson.D{{"viewers.groupids", groupId}})
		}
	}

	filter := bson.D{
		{
			"$and", []interface{}{
				bson.D{{"objecttype", objectType}},
				bson.M{"$or": idLookups},
			},
		},
	}

	result := map[string]*protos.OwnershipItem{}

	opts := options.Find() //.SetProjection(bson.D{{"_id", true}})
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).Find(context.TODO(), filter, opts)
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
		result[item.Id] = item
	}

	return result, nil
}

func MakeOwnerSummary(ownership *protos.OwnershipItem, db *mongo.Database, ts timestamper.ITimeStamper) *protos.OwnershipSummary {
	user, err := getUserInfo(ownership.CreatorUserId, db, ts)
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

	if ownership.Viewers != nil {
		result.ViewerUserCount = uint32(len(ownership.Viewers.UserIds))

		// NOTE: if we're a viewer, subtract one!
		if utils.ItemInSlice(ownership.CreatorUserId, ownership.Viewers.UserIds) && result.ViewerUserCount > 0 {
			result.ViewerUserCount--
		}

		result.ViewerGroupCount = uint32(len(ownership.Viewers.GroupIds))
	}
	if ownership.Editors != nil {
		result.EditorUserCount = uint32(len(ownership.Editors.UserIds))

		// NOTE: if we're a viewer, subtract one!
		if utils.ItemInSlice(ownership.CreatorUserId, ownership.Editors.UserIds) && result.EditorUserCount > 0 {
			result.EditorUserCount--
		}

		result.EditorGroupCount = uint32(len(ownership.Editors.GroupIds))
	}

	result.SharedWithOthers = result.ViewerUserCount > 0 || result.ViewerGroupCount > 0 || result.EditorUserCount > 0 || result.EditorGroupCount > 0
	return result
}
