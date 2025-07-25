package wsHandler

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/auth0login"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"gopkg.in/auth0.v4/management"
)

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

	// Now check if the group has any default roles and assign them if so
	// Only do this for group members, not viewers
	err = assignDefaultRoles(req.GetUserMemberId(), group, hctx)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupAddMemberResp{
		Group: group,
	}, nil
}

func assignDefaultRoles(targetUserId string, group *protos.UserGroup, hctx wsHelpers.HandlerContext) error {
	if len(group.Info.DefaultRoles) > 0 {
		targetUser, err := wsHelpers.GetDBUser(targetUserId, hctx.Svcs.MongoDB)
		if err != nil {
			return err
		}

		auth0API, err := auth0login.InitAuth0ManagementAPI(hctx.Svcs.Config)
		if err != nil {
			return errors.New("failed to initialize Auth0 API: " + err.Error())
		}

		targetUserRoles, err := auth0API.User.Roles(targetUser.Id)
		if err != nil {
			return errors.New("failed to fetch roles for user: " + err.Error())
		}

		allRoles, err := auth0API.Role.List()
		if err != nil {
			return errors.New("failed to list roles:" + err.Error())
		}
		if len(allRoles.Roles) == 0 {
			return errors.New("no roles found")
		}

		userRoleMap := make(map[string]bool)
		for _, role := range targetUserRoles.Roles {
			userRoleMap[*role.ID] = true
		}

		roleMap := make(map[string]*management.Role)
		for _, role := range allRoles.Roles {
			roleMap[*role.ID] = role
		}

		// Only assign missing default roles
		for _, roleID := range group.Info.DefaultRoles {
			if !userRoleMap[roleID] {
				if role, exists := roleMap[roleID]; exists {
					fmt.Printf("Assigning role: %v to user: %v", roleID, targetUser.Id)
					auth0API.User.AssignRoles(targetUser.Id, role)
				}
			}
		}
	}

	return nil
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
	otherFieldStart := "viewers"
	if viewer {
		fieldStart = "viewers"
		otherFieldStart = "members"
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
	otherIdName := otherFieldStart + "." + idName
	idName = fieldStart + "." + idName

	if err := wsHelpers.CheckStringField(&checkId, idName, 1, idMaxLen); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	// NOTE: If member/viewer delete message has userid==sessionid, allow the person to leave a group
	var group *protos.UserGroupDB
	var err error

	if !add && !isGroup && checkId == hctx.SessUser.User.Id {
		// User is trying to leave a group they're in, allow this!
		group, err = getGroup(groupId, ctx, coll)
	} else {
		group, err = getGroupAndCheckPermission(groupId, hctx.SessUser.User.Id, hctx.SessUser.Permissions, ctx, coll)
	}

	if err != nil {
		return nil, err
	}

	groupList := group.Members
	otherGroupList := group.Viewers
	if viewer {
		groupList = group.Viewers
		otherGroupList = group.Members
	}

	editIds := groupList.GroupIds
	otherEditIds := otherGroupList.GroupIds
	if !isGroup {
		editIds = groupList.UserIds
		otherEditIds = otherGroupList.UserIds
	}

	dbOp := "$pull"
	if add {
		if utils.ItemInSlice(checkId, editIds) {
			return nil, errorwithstatus.MakeBadRequestError(errors.New(checkId + " is already a " + idName))
		}
		// Check the "other" list too (eg if we're adding to Viewer, check that not already Member, vice versa)
		if utils.ItemInSlice(checkId, otherEditIds) {
			return nil, errorwithstatus.MakeBadRequestError(errors.New(checkId + " is already a " + otherIdName))
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
	result, err := coll.UpdateByID(ctx, groupId, bson.D{{Key: dbOp, Value: bson.D{{Key: dbField, Value: checkId}}}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup %v %v result had unexpected counts %+v id: %v", dbOp, dbField, result, checkId)
	}

	_, err = coll.UpdateByID(ctx, groupId, bson.D{{Key: "$set", Value: bson.D{{Key: "lastuserjoinedunixsec", Value: uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())}}}})
	if err != nil {
		return nil, err
	}

	// Dismiss any outstanding request for this user to join the group
	if !isGroup && add {
		coll = hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupJoinRequestsName)
		filter := bson.M{"joingroupid": groupId, "userid": opUserId}
		dismissResult, err := coll.DeleteMany(ctx, filter)
		if err != nil {
			hctx.Svcs.Log.Errorf("UserGroup %v add viewer/member user %v failed to dismiss requests: %v", groupId, opUserId, err)
		}
		if dismissResult.DeletedCount <= 0 {
			hctx.Svcs.Log.Errorf("UserGroup %v add viewer/member user %v had no join requests to dismiss", groupId, opUserId)
		}
	}

	groupSend, err := decorateUserGroup(group, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return groupSend, nil
}

func getGroup(groupId string, ctx context.Context, coll *mongo.Collection) (*protos.UserGroupDB, error) {
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

	return &group, nil
}

func getGroupAndCheckPermission(groupId string, requestingUser string, requestingUserPermission map[string]bool, ctx context.Context, coll *mongo.Collection) (*protos.UserGroupDB, error) {
	// First read the group in question
	group, err := getGroup(groupId, ctx, coll)
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

	return group, nil
}

// Take the DB-based usergroup passed in and form a user group that can be returned in responses
// These contain all the "aux" user/group fields like names, icons etc
func decorateUserGroup(dbGroup *protos.UserGroupDB, db *mongo.Database, logger logger.ILogger) (*protos.UserGroup, error) {
	result := &protos.UserGroup{
		Info: &protos.UserGroupInfo{
			Id:             dbGroup.Id,
			Name:           dbGroup.Name,
			Description:    dbGroup.Description,
			CreatedUnixSec: dbGroup.CreatedUnixSec,
			Joinable:       dbGroup.Joinable,
			DefaultRoles:   dbGroup.DefaultRoles,
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
		dbGroup.Members.UserIds,
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
	opts := options.Find().SetProjection(bson.M{"id": true, "name": true, "createdunixsec": true, "joinable": true})

	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.UserGroupDB{}
	err = cursor.All(ctx, &items)
	if err != nil {
		return nil, err
	}

	// Form the map and we're done
	for _, item := range items {
		result[item.Id] = &protos.UserGroupInfo{
			// NOTE: only item.Info is valid, the rest hasn't been read due to the DB Find() call
			Id:             item.Id,
			Name:           item.Name,
			CreatedUnixSec: item.CreatedUnixSec,
			Joinable:       item.Joinable,
			DefaultRoles:   item.DefaultRoles,
		}
	}

	return result, nil
}
