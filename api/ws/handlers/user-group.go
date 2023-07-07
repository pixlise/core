package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v3/OLDCODE/core/utils"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
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

	groups := []*protos.UserGroup{}
	err = cursor.All(context.TODO(), &groups)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupListResp{
		Groups: groups,
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

	group := &protos.UserGroup{
		Id:             groupId,
		Name:           req.Name,
		CreatedUnixSec: uint64(hctx.Svcs.TimeStamper.GetTimeNowSec()),
		Members:        &protos.UserGroupList{},
		AdminUserIds:   []string{
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

	return &protos.UserGroupCreateResp{Group: group}, nil
}

func HandleUserGroupDeleteReq(req *protos.UserGroupDeleteReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupDeleteResp, error) {
	// Should only be called if we have admin rights, so other permission issues here
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, 50); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	result, err := coll.DeleteOne(ctx, bson.M{"id": req.GroupId})
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
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, 50); err != nil {
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
	group := protos.UserGroup{}
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
	return &protos.UserGroupSetNameResp{Group: &group}, nil
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
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, 50); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.AdminUserId, "AdminUserId", 1, 50); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	group, err := getGroupAndCheckPermission(req.GroupId, hctx.SessUser.User.Id, hctx.SessUser.Permissions, ctx, coll)
	if err != nil {
		return nil, err
	}

	if utils.StringInSlice(req.AdminUserId, group.AdminUserIds) {
		return nil, errorwithstatus.MakeBadRequestError(errors.New(req.AdminUserId + " is already an admin"))
	}

	// We're allowed to edit, so do it
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.ElementSetsName).UpdateByID(ctx, req.GroupId, bson.D{{Key: "$add", Value: bson.D{{"adminuserids", req.AdminUserId}}}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup Add Admin result had unexpected counts %+v id: %v", result, group.Id)
	}

	return &protos.UserGroupAddAdminResp{
		Group: group,
	}, nil
}

func HandleUserGroupDeleteAdminReq(req *protos.UserGroupDeleteAdminReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupDeleteAdminResp, error) {
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, 50); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.AdminUserId, "AdminUserId", 1, 50); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	group, err := getGroupAndCheckPermission(req.GroupId, hctx.SessUser.User.Id, hctx.SessUser.Permissions, ctx, coll)
	if err != nil {
		return nil, err
	}

	if !utils.StringInSlice(req.AdminUserId, group.AdminUserIds) {
		return nil, errorwithstatus.MakeBadRequestError(errors.New(req.AdminUserId + " is not an admin"))
	}

	// We're allowed to edit, so do it
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.ElementSetsName).UpdateByID(ctx, req.GroupId, bson.D{{Key: "$delete", Value: bson.D{{"adminuserids", req.AdminUserId}}}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup Delete Admin result had unexpected counts %+v id: %v", result, group.Id)
	}

	return &protos.UserGroupDeleteAdminResp{
		Group: group,
	}, nil
}

func HandleUserGroupAddMemberReq(req *protos.UserGroupAddMemberReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupAddMemberResp, error) {
	return nil, errors.New("HandleUserGroupAddMemberReq not implemented yet")
}

func HandleUserGroupDeleteMemberReq(req *protos.UserGroupDeleteMemberReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupDeleteMemberResp, error) {
	return nil, errors.New("HandleUserGroupDeleteMemberReq not implemented yet")
}

func getGroupAndCheckPermission(groupId string, requestingUser string, requestingUserPermission map[string]bool, ctx context.Context, coll *mongo.Collection) (*protos.UserGroup, error) {
	// First read the group in question
	groupRes := coll.FindOne(ctx, bson.M{"id": groupId})
	if groupRes.Err() != nil {
		if groupRes.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(groupId)
		}
		return nil, groupRes.Err()
	}

	group := protos.UserGroup{}
	err := groupRes.Decode(&group)
	if err != nil {
		return nil, err
	}

	// Can be called by admins or non-admins who are admins of this group, so we need to check permissions here
	isAllowed := wsHelpers.HasPermission(requestingUserPermission, protos.Permission_PERM_PIXLISE_ADMIN)
	if !isAllowed {
		// Check if it's a member of the group admins
		if utils.StringInSlice(requestingUser, group.AdminUserIds) {
			isAllowed = true
		}
	}

	if !isAllowed {
		return nil, errorwithstatus.MakeUnauthorisedError(errors.New("Not allowed to edit user group"))
	}

	return &group, nil
}
