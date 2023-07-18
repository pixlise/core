package wsHandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleUserGroupListReq(req *protos.UserGroupListReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupListResp, error) {
	// Should only be called if we have admin rights, so other permission issues here
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	filter := bson.D{}
	opts := options.Find()
	cursor, err := coll.Find(ctx, filter, opts)

	groups := []*protos.UserGroupDB{}
	err = cursor.All(context.TODO(), &groups)
	if err != nil {
		return nil, err
	}

	groupsForSend := []*protos.UserGroup{}
	for _, group := range groups {
		groupSend, err := decorateUserGroup(group, hctx.Svcs.MongoDB, hctx.Svcs.Log)
		if err != nil {
			return nil, err
		}
		groupsForSend = append(groupsForSend, groupSend)
	}

	return &protos.UserGroupListResp{
		Groups: groupsForSend,
	}, nil
}

func HandleUserGroupCreateReq(req *protos.UserGroupCreateReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupCreateResp, error) {
	// Should only be called if we have admin rights, so other permission issues here
	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 50); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	exists, err, _ := checkUserGroupNameExists(req.Name, ctx, coll)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf(`Name: "%v" already exists`, req.Name))
	}

	// At this point we should know that the name is not taken
	groupId := hctx.Svcs.IDGen.GenObjectID()

	group := &protos.UserGroupDB{
		Id:             groupId,
		Name:           req.Name,
		CreatedUnixSec: uint64(hctx.Svcs.TimeStamper.GetTimeNowSec()),
		Members: &protos.UserGroupList{
			UserIds:  []string{},
			GroupIds: []string{},
		},
		Viewers: &protos.UserGroupList{
			UserIds:  []string{},
			GroupIds: []string{},
		},
		AdminUserIds: []string{
			// Creator is an admin user who can create items, but
			// this list is for non-admin users who can be given
			// admin rights over a group (just to add/remove members)
			// so we don't need to add the creating user here!
		},
	}

	_, _err := coll.InsertOne(ctx, group)
	if _err != nil {
		return nil, _err
	}

	groupSend, err := decorateUserGroup(group, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupCreateResp{Group: groupSend}, nil
}

func HandleUserGroupDeleteReq(req *protos.UserGroupDeleteReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupDeleteResp, error) {
	// Should only be called if we have admin rights, so other permission issues here
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	result, err := coll.DeleteOne(ctx, bson.M{"_id": req.GroupId})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.GroupId)
		}
		return nil, err
	}

	if result.DeletedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup Delete result had unexpected counts %+v id: %v", result, req.GroupId)
	}

	return &protos.UserGroupDeleteResp{}, nil
}

func HandleUserGroupSetNameReq(req *protos.UserGroupSetNameReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupSetNameResp, error) {
	// Should only be called if we have admin rights, so other permission issues here
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 50); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	groupResult := coll.FindOne(ctx, bson.M{"_id": req.GroupId})
	if groupResult.Err() != nil {
		if groupResult.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.GroupId)
		}
		return nil, groupResult.Err()
	}

	// Read existing group (so we can return it)
	group := protos.UserGroupDB{}
	err := groupResult.Decode(&group)
	if err != nil {
		return nil, err
	}

	// Update the name
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName).UpdateByID(ctx, req.GroupId, bson.D{{Key: "$set", Value: bson.D{{"name", req.Name}}}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup UpdateByID result had unexpected counts %+v id: %v", result, group.Id)
	}

	group.Name = req.Name
	groupSend, err := decorateUserGroup(&group, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupSetNameResp{Group: groupSend}, nil
}

func checkUserGroupNameExists(name string, ctx context.Context, coll *mongo.Collection) (bool, error, *mongo.SingleResult) {
	// Check if name exists already
	existing := coll.FindOne(ctx, bson.M{"name": name})

	// Should return ErrNoDocuments if name is not already taken... So lack of error means we have one, so this is an error!
	if existing.Err() == nil {
		return true, nil, existing
	} else {
		// Got an error, make sure it's the right one
		if existing.Err() != mongo.ErrNoDocuments {
			return false, errorwithstatus.MakeBadRequestError(fmt.Errorf(`Failed to check if name: "%v" is unique`, name)), existing
		}
	}

	return false, nil, existing
}

func HandleUserGroupAddAdminReq(req *protos.UserGroupAddAdminReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupAddAdminResp, error) {
	group, err := modifyGroupAdminList(req.GroupId, req.AdminUserId, true, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupAddAdminResp{
		Group: group,
	}, nil
}

func HandleUserGroupDeleteAdminReq(req *protos.UserGroupDeleteAdminReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupDeleteAdminResp, error) {
	group, err := modifyGroupAdminList(req.GroupId, req.AdminUserId, false, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupDeleteAdminResp{
		Group: group,
	}, nil
}

func modifyGroupAdminList(groupId string, adminUserId string, add bool, hctx wsHelpers.HandlerContext) (*protos.UserGroup, error) {
	if err := wsHelpers.CheckStringField(&groupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&adminUserId, "AdminUserId", 1, wsHelpers.Auth0UserIdFieldMaxLength); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	group, err := getGroupAndCheckPermission(groupId, hctx.SessUser.User.Id, hctx.SessUser.Permissions, ctx, coll)
	if err != nil {
		return nil, err
	}

	dbOp := "$pull"
	if add {
		// Check if already in there
		if utils.ItemInSlice(adminUserId, group.AdminUserIds) {
			return nil, errorwithstatus.MakeBadRequestError(errors.New(adminUserId + " is already an admin"))
		}
		dbOp = "$addToSet"
		// Add to result already, if we fail to write to db, result wont be sent
		group.AdminUserIds = append(group.AdminUserIds, adminUserId)
	} else {
		// Check that it's actually there
		idx := -1
		for c, id := range group.AdminUserIds {
			if adminUserId == id {
				// Found it!
				idx = c
				break
			}
		}
		if idx == -1 {
			return nil, errorwithstatus.MakeBadRequestError(errors.New(adminUserId + " is not an admin"))
		}

		// Delete from result already, if we fail to write to db, result wont be sent
		group.AdminUserIds = append(group.AdminUserIds[0:idx], group.AdminUserIds[idx+1:]...)
	}

	// We're allowed to edit, so do it
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName).UpdateByID(ctx, groupId, bson.M{dbOp: bson.M{"adminuserids": adminUserId}})
	if err != nil {
		return nil, err
	}

	if result.ModifiedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup Admin %v result had unexpected counts %+v id: %v", dbOp, result, group.Id)
	}

	groupSend, err := decorateUserGroup(group, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return groupSend, nil
}

func HandleUserGroupAddViewerReq(req *protos.UserGroupAddViewerReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupAddViewerResp, error) {
	group, err := modifyGroupMembershipList(req.GroupId, req.GetGroupViewerId(), req.GetUserViewerId(), true, true, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupAddViewerResp{
		Group: group,
	}, nil
}

func HandleUserGroupDeleteViewerReq(req *protos.UserGroupDeleteViewerReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupDeleteViewerResp, error) {
	group, err := modifyGroupMembershipList(req.GroupId, req.GetGroupViewerId(), req.GetUserViewerId(), true, false, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupDeleteViewerResp{
		Group: group,
	}, nil
}

func HandleUserGroupAddMemberReq(req *protos.UserGroupAddMemberReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupAddMemberResp, error) {
	group, err := modifyGroupMembershipList(req.GroupId, req.GetGroupMemberId(), req.GetUserMemberId(), false, true, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupAddMemberResp{
		Group: group,
	}, nil
}

func HandleUserGroupDeleteMemberReq(req *protos.UserGroupDeleteMemberReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupDeleteMemberResp, error) {
	group, err := modifyGroupMembershipList(req.GroupId, req.GetGroupMemberId(), req.GetUserMemberId(), false, false, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupDeleteMemberResp{
		Group: group,
	}, nil
}

// Does the job of the above...
// viewer=true means editing group viewers, viewer=false means editing group members
// add=true means adding, add=false means deleting
func modifyGroupMembershipList(groupId string, opGroupId string, opUserId string, viewer bool, add bool, hctx wsHelpers.HandlerContext) (*protos.UserGroup, error) {
	if err := wsHelpers.CheckStringField(&groupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	fieldStart := "members"
	if viewer {
		fieldStart = "viewers"
	}

	// Must have one of these...
	checkId := opGroupId
	idMaxLen := wsHelpers.IdFieldMaxLength
	idName := "GroupId"
	isGroup := true
	dbField := fieldStart + ".groupids"
	if len(checkId) <= 0 {
		checkId = opUserId
		idMaxLen = wsHelpers.Auth0UserIdFieldMaxLength
		idName = "UserId"
		isGroup = false
		dbField = fieldStart + ".userids"
	}
	idName = fieldStart + "." + idName

	if err := wsHelpers.CheckStringField(&checkId, idName, 1, idMaxLen); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	group, err := getGroupAndCheckPermission(groupId, hctx.SessUser.User.Id, hctx.SessUser.Permissions, ctx, coll)
	if err != nil {
		return nil, err
	}

	groupList := group.Members
	if viewer {
		groupList = group.Viewers
	}

	editIds := groupList.GroupIds
	if !isGroup {
		editIds = groupList.UserIds
	}

	dbOp := "$pull"
	if add {
		if utils.ItemInSlice(checkId, editIds) {
			return nil, errorwithstatus.MakeBadRequestError(errors.New(checkId + " is already a " + idName))
		}
		dbOp = "$addToSet"
		if isGroup {
			groupList.GroupIds = append(groupList.GroupIds, checkId)
		} else {
			groupList.UserIds = append(groupList.UserIds, checkId)
		}
	} else {
		// Find the index
		idx := -1
		for c, id := range editIds {
			if checkId == id {
				// Found it!
				idx = c
				break
			}
		}

		if idx == -1 {
			return nil, errorwithstatus.MakeBadRequestError(errors.New(checkId + " is not a " + idName))
		}

		// Delete from our group that we're returning too
		if isGroup {
			groupList.GroupIds = append(groupList.GroupIds[0:idx], groupList.GroupIds[idx+1:]...)
		} else {
			groupList.UserIds = append(groupList.UserIds[0:idx], groupList.UserIds[idx+1:]...)
		}
	}

	// We're allowed to edit, so do it
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName).UpdateByID(ctx, groupId, bson.D{{Key: dbOp, Value: bson.D{{dbField, checkId}}}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup %v %v result had unexpected counts %+v id: %v", dbOp, dbField, result, checkId)
	}

	groupSend, err := decorateUserGroup(group, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return groupSend, nil
}

func getGroupAndCheckPermission(groupId string, requestingUser string, requestingUserPermission map[string]bool, ctx context.Context, coll *mongo.Collection) (*protos.UserGroupDB, error) {
	// First read the group in question
	groupRes := coll.FindOne(ctx, bson.M{"_id": groupId})
	if groupRes.Err() != nil {
		if groupRes.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(groupId)
		}
		return nil, groupRes.Err()
	}

	group := protos.UserGroupDB{}
	err := groupRes.Decode(&group)
	if err != nil {
		return nil, err
	}

	// Can be called by admins or non-admins who are admins of this group, so we need to check permissions here
	isAllowed := wsHelpers.HasPermission(requestingUserPermission, protos.Permission_PERM_PIXLISE_ADMIN)
	if !isAllowed {
		// Check if it's a member of the group admins
		if utils.ItemInSlice(requestingUser, group.AdminUserIds) {
			isAllowed = true
		}
	}

	if !isAllowed {
		return nil, errorwithstatus.MakeUnauthorisedError(errors.New("Not allowed to edit user group"))
	}

	return &group, nil
}

// Take the DB-based usergroup passed in and form a user group that can be returned in responses
// These contain all the "aux" user/group fields like names, icons etc
func decorateUserGroup(dbGroup *protos.UserGroupDB, db *mongo.Database, logger logger.ILogger) (*protos.UserGroup, error) {
	result := &protos.UserGroup{
		Info: &protos.UserGroupInfo{
			Id:             dbGroup.Id,
			Name:           dbGroup.Name,
			CreatedUnixSec: dbGroup.CreatedUnixSec,
		},
		Viewers:    &protos.UserGroupInfoList{},
		Members:    &protos.UserGroupInfoList{},
		AdminUsers: []*protos.UserInfo{},
	}

	// All the user lists (GetDBUser has local caching so it's not all bad...)
	writeToLists := []*[]*protos.UserInfo{
		&result.AdminUsers,
		&result.Viewers.Users,
		&result.Members.Users,
	}

	readFromLists := [][]string{
		dbGroup.AdminUserIds,
		dbGroup.Viewers.UserIds,
		dbGroup.Members.GroupIds,
	}

	listName := []string{"Admin", "Viewers", "Members"}

	for c, writeToList := range writeToLists {
		readFromList := readFromLists[c]

		for _, userId := range readFromList {
			var user *protos.UserInfo

			if item, err := wsHelpers.GetDBUser(userId, db); err != nil {
				// Print an error but return an empty user struct
				logger.Errorf("Failed to find user info for user-group %v %v user ID %v", listName[c], dbGroup.Id, userId)
				user = &protos.UserInfo{
					Id: userId,
				}
			} else {
				user = item.Info
			}

			*writeToList = append(*writeToList, user)
		}
	}

	// Now read all the groups we're interested in into a map, form where we
	// can pick them out and form our return lists
	groupIds := append(dbGroup.Viewers.GroupIds, dbGroup.Members.GroupIds...)
	groupLookup, err := getUserGroupInfos(groupIds, db)
	if err != nil {
		logger.Errorf("Error reading groups: %v. Error was: %v", strings.Join(groupIds, ","), err)
	}

	readGroupFromLists := [][]string{
		dbGroup.Viewers.GroupIds, dbGroup.Members.GroupIds,
	}
	listName = []string{"Viewers", "Members"}
	writeToGroupLists := []*[]*protos.UserGroupInfo{
		&result.Viewers.Groups, &result.Members.Groups,
	}

	for c, writeToList := range writeToGroupLists {
		readFromList := readGroupFromLists[c]

		for _, groupId := range readFromList {
			var group *protos.UserGroupInfo
			if groupFound, ok := groupLookup[groupId]; ok {
				group = groupFound
			} else {
				logger.Errorf("Failed to find group info for user-group %v %v groupId %v", listName[c], dbGroup.Id, groupId)
				group = &protos.UserGroupInfo{
					Id: groupId,
				}
			}

			*writeToList = append(*writeToList, group)
		}
	}

	return result, nil
}

func getUserGroupInfos(userGroupIds []string, db *mongo.Database) (map[string]*protos.UserGroupInfo, error) {
	result := map[string]*protos.UserGroupInfo{}

	ctx := context.TODO()
	coll := db.Collection(dbCollections.UserGroupsName)

	// Read the requested items from DB, but only reading the fields we're interested in!
	filter := bson.M{"_id": bson.M{"$in": userGroupIds}}
	opts := options.Find().SetProjection(bson.D{{"info", true}})
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.UserGroup{}
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}

	// Form the map and we're done
	for _, item := range items {
		result[item.Info.Id] = &protos.UserGroupInfo{
			// NOTE: only item.Info is valid, the rest hasn't been read due to the DB Find() call
			Id:             item.Info.Id,
			Name:           item.Info.Name,
			CreatedUnixSec: item.Info.CreatedUnixSec,
		}
	}

	return result, nil
}
