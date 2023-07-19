package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

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
