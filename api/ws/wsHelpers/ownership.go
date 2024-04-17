package wsHelpers

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MakeOwnerForWrite(objectId string, objectType protos.ObjectType, creatorUserId string, createTimeUnixSec int64) *protos.OwnershipItem {
	ownerItem := &protos.OwnershipItem{
		Id:             objectId,
		ObjectType:     objectType,
		CreatedUnixSec: uint32(createTimeUnixSec),
		CreatorUserId:  "",
		Editors: &protos.UserGroupList{
			UserIds:  []string{},
			GroupIds: []string{},
		},
		Viewers: &protos.UserGroupList{
			UserIds:  []string{},
			GroupIds: []string{},
		},
	}

	if len(creatorUserId) > 0 {
		ownerItem.CreatorUserId = creatorUserId
		//ownerItem.Viewers
		ownerItem.Editors = &protos.UserGroupList{
			UserIds: []string{creatorUserId},
		}
	}

	return ownerItem
}

// Checks object access - if requireEdit is true, it checks for edit access
// otherwise just checks for view access. Returns an error if it failed to determine
// or if access is not granted, returns error formed with MakeUnauthorisedError
func CheckObjectAccess(requireEdit bool, objectId string, objectType protos.ObjectType, hctx HandlerContext) (*protos.OwnershipItem, error) {
	return CheckObjectAccessForUser(requireEdit, objectId, objectType, hctx.SessUser.User.Id, hctx.SessUser.MemberOfGroupIds, hctx.Svcs.MongoDB)
}

func CheckObjectAccessForUser(requireEdit bool, objectId string, objectType protos.ObjectType, userId string, memberOfGroupIds []string, db *mongo.Database) (*protos.OwnershipItem, error) {
	ownerCollectionId := objectId

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
	return nil, errorwithstatus.MakeUnauthorisedError(fmt.Errorf("%v access denied for: %v (%v)", accessType, objectType.String(), objectId))
}

// Gets all object IDs which the user has access to - if requireEdit is true, it checks for edit access
// otherwise just checks for view access
// Returns a map of object id->creator user id
func ListAccessibleIDs(requireEdit bool, objectType protos.ObjectType, svcs *services.APIServices, requestorSession SessionUser) (map[string]*protos.OwnershipItem, error) {
	idLookups := []interface{}{
		bson.D{{Key: "editors.userids", Value: requestorSession.User.Id}},
	}
	if !requireEdit {
		idLookups = append(idLookups, bson.D{{Key: "viewers.userids", Value: requestorSession.User.Id}})
	}

	// Add the group IDs
	for _, groupId := range requestorSession.MemberOfGroupIds {
		idLookups = append(idLookups, bson.D{{Key: "editors.groupids", Value: groupId}})
		if !requireEdit {
			idLookups = append(idLookups, bson.D{{Key: "viewers.groupids", Value: groupId}})
		}
	}

	filter := bson.D{
		{
			Key: "$and", Value: []interface{}{
				bson.D{{Key: "objecttype", Value: objectType}},
				bson.M{"$or": idLookups},
			},
		},
	}

	result := map[string]*protos.OwnershipItem{}

	opts := options.Find() //.SetProjection(bson.D{{"_id", true}})
	cursor, err := svcs.MongoDB.Collection(dbCollections.OwnershipName).Find(context.TODO(), filter, opts)
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

func ListGroupAccessibleIDs(requireEdit bool, objectType protos.ObjectType, groupID string, mongoDB *mongo.Database) (map[string]*protos.OwnershipItem, error) {
	idLookups := []interface{}{
		bson.D{{Key: "editors.groupids", Value: groupID}},
	}

	if !requireEdit {
		idLookups = append(idLookups, bson.D{{Key: "viewers.groupids", Value: groupID}})
	}

	filter := bson.D{
		{
			Key: "$and", Value: []interface{}{
				bson.D{{Key: "objecttype", Value: objectType}},
				bson.M{"$or": idLookups},
			},
		},
	}

	result := map[string]*protos.OwnershipItem{}

	opts := options.Find() //.SetProjection(bson.D{{"_id", true}})
	cursor, err := mongoDB.Collection(dbCollections.OwnershipName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

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

func FetchOwnershipSummary(ownership *protos.OwnershipItem, sessionUser SessionUser, db *mongo.Database, ts timestamper.ITimeStamper, fullDetails bool) *protos.OwnershipSummary {
	user, err := getUserInfo(ownership.CreatorUserId, db, ts)
	result := &protos.OwnershipSummary{
		CreatedUnixSec: ownership.CreatedUnixSec,
		CanEdit:        false,
	}
	if err == nil {
		result.CreatorUser = user
	} else {
		result.CreatorUser = &protos.UserInfo{
			Id: ownership.CreatorUserId,
		}
	}

	// Still have to be an editor even if you're the creator
	result.CanEdit = false

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

		// NOTE: if we're an editor, subtract one!
		if utils.ItemInSlice(ownership.CreatorUserId, ownership.Editors.UserIds) && result.EditorUserCount > 0 {
			result.EditorUserCount--
		}

		result.EditorGroupCount = uint32(len(ownership.Editors.GroupIds))

		if !result.CanEdit && ownership.Editors.UserIds != nil {
			result.CanEdit = utils.ItemInSlice(sessionUser.User.Id, ownership.Editors.UserIds)
		}

		if !result.CanEdit && ownership.Editors.GroupIds != nil {
			for _, groupId := range sessionUser.MemberOfGroupIds {
				if utils.ItemInSlice(groupId, ownership.Editors.GroupIds) {
					result.CanEdit = true
					break
				}
			}
		}
	}

	result.SharedWithOthers = result.ViewerUserCount > 0 || result.ViewerGroupCount > 0 || result.EditorUserCount > 0 || result.EditorGroupCount > 0

	// Hide more data intensive fields if we don't care
	if !fullDetails {
		result.CreatorUser.IconURL = ""
	}

	return result
}

func MakeOwnerSummary(ownership *protos.OwnershipItem, sessionUser SessionUser, db *mongo.Database, ts timestamper.ITimeStamper) *protos.OwnershipSummary {
	return FetchOwnershipSummary(ownership, sessionUser, db, ts, false)
}

func MakeFullOwnerSummary(ownership *protos.OwnershipItem, sessionUser SessionUser, db *mongo.Database, ts timestamper.ITimeStamper) *protos.OwnershipSummary {
	return FetchOwnershipSummary(ownership, sessionUser, db, ts, true)
}

func FindUserIdsFor(objectId string, mongoDB *mongo.Database) ([]string, error) {
	filter := bson.M{"_id": objectId}
	opts := options.FindOne()
	ownership := mongoDB.Collection(dbCollections.OwnershipName).FindOne(context.TODO(), filter, opts)
	if ownership.Err() != nil {
		if ownership.Err() == mongo.ErrNoDocuments {
			return []string{}, errorwithstatus.MakeNotFoundError(objectId)
		}
		return []string{}, ownership.Err()
	}

	ownershipItem := protos.OwnershipItem{}
	err := ownership.Decode(&ownershipItem)
	if err != nil {
		return []string{}, err
	}

	userIds := []string{}
	groupIds := []string{}

	// Gather up all the user ids for the groups selected
	if ownershipItem.Viewers != nil {
		userIds = append(userIds, ownershipItem.Viewers.UserIds...)
		groupIds = append(groupIds, ownershipItem.Viewers.GroupIds...)
	}
	if ownershipItem.Editors != nil {
		userIds = append(userIds, ownershipItem.Editors.UserIds...)
		groupIds = append(groupIds, ownershipItem.Editors.GroupIds...)
	}

	usersForGroups, err := GetUserIdsForGroup(groupIds, mongoDB)
	if err != nil {
		return []string{}, err
	}

	return append(userIds, usersForGroups...), nil
}

func GetUserIdsForGroup(groupIds []string, mongoDB *mongo.Database) ([]string, error) {
	filter := bson.M{"_id": bson.M{"$in": groupIds}}
	opts := options.Find()
	cursor, err := mongoDB.Collection(dbCollections.UserGroupsName).Find(context.TODO(), filter, opts)

	if err != nil {
		return []string{}, err
	}

	groups := []*protos.UserGroupDB{}
	err = cursor.All(context.TODO(), &groups)
	if err != nil {
		return []string{}, err
	}

	userIds := []string{}
	for _, group := range groups {
		// Pull in the users
		for _, userId := range group.Viewers.UserIds {
			userIds = append(userIds, userId)
		}
		for _, userId := range group.Members.UserIds {
			userIds = append(userIds, userId)
		}

		// Recurse into groups
		groupIds := []string{}
		for _, groupId := range group.Viewers.GroupIds {
			groupIds = append(groupIds, groupId)
		}
		for _, groupId := range group.Members.GroupIds {
			groupIds = append(groupIds, groupId)
		}

		usersForGroup, err := GetUserIdsForGroup(groupIds, mongoDB)
		if err != nil {
			return []string{}, err
		}

		userIds = append(userIds, usersForGroup...)
	}

	return userIds, nil
}
