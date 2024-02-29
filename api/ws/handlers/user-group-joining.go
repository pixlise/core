package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleUserGroupJoinReq(req *protos.UserGroupJoinReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupJoinResp, error) {
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	ctx := context.TODO()

	// Check if this group exists
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)
	group, err := getGroup(req.GroupId, ctx, coll)
	if err != nil {
		return nil, err
	}

	// Check that user is not already a viewer OR member of the group
	if utils.ItemInSlice(hctx.SessUser.User.Id, group.Members.UserIds) || utils.ItemInSlice(hctx.SessUser.User.Id, group.Viewers.UserIds) {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("User is already a member/viewer of group %v", req.GroupId))
	}

	// Add a record to DB for this join request
	reqId := hctx.Svcs.IDGen.GenObjectID()

	reqDB := &protos.UserGroupJoinRequestDB{
		Id:             reqId,
		UserId:         hctx.SessUser.User.Id,
		JoinGroupId:    req.GroupId,
		CreatedUnixSec: uint32(hctx.Svcs.TimeStamper.GetTimeNowSec()),
		AsMember:       req.AsMember,
	}

	coll = hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupJoinRequestsName)
	_, _err := coll.InsertOne(ctx, reqDB)
	if _err != nil {
		return nil, _err
	}

	// Create & send a notification
	intent := "viewer"
	if req.AsMember {
		intent = "member"
	}

	hctx.Svcs.Notifier.NotifyUserMessage(
		fmt.Sprintf("%v has requested to join group %v", hctx.SessUser.User.Name, group.Name),
		fmt.Sprintf(`You are being sent this because you are an administrator of PIXLISE user group %v.
A user named %v has just requested to join the group as a %v`, group.Name, hctx.SessUser.User.Name, intent),
		protos.NotificationType_NT_JOIN_GROUP_REQUEST,
		"/user-group/join-requests", // TODO clarify
		hctx.SessUser.User.Id,
		group.AdminUserIds,
		"PIXLISE API",
	)

	return &protos.UserGroupJoinResp{}, nil
}

func HandleUserGroupIgnoreJoinReq(req *protos.UserGroupIgnoreJoinReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupIgnoreJoinResp, error) {
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}
	if err := wsHelpers.CheckStringField(&req.RequestId, "RequestId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Delete the join request if exists
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	_, err := getGroupAndCheckPermission(req.GroupId, hctx.SessUser.User.Id, hctx.SessUser.Permissions, ctx, coll)
	if err != nil {
		return nil, err
	}

	coll = hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupJoinRequestsName)
	result, err := coll.DeleteOne(ctx, bson.M{"_id": req.RequestId})
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError(req.RequestId)
		}
		return nil, err
	}

	if result.DeletedCount != 1 {
		hctx.Svcs.Log.Errorf("UserGroup Join Request Delete result had unexpected counts %+v id: %v", result, req.RequestId)
	}

	return &protos.UserGroupIgnoreJoinResp{}, nil
}

func HandleUserGroupJoinListReq(req *protos.UserGroupJoinListReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupJoinListResp, error) {
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// If user has admin rights to this group, send back a list of all requests to join this group
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	_, err := getGroupAndCheckPermission(req.GroupId, hctx.SessUser.User.Id, hctx.SessUser.Permissions, ctx, coll)
	if err != nil {
		return nil, err
	}

	coll = hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupJoinRequestsName)

	filter := bson.M{"joingroupid": req.GroupId}
	opts := options.Find()
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.UserGroupJoinRequestDB{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Loop through join requests and add user details
	for _, item := range items {
		userDBItem, err := wsHelpers.GetDBUser(item.UserId, hctx.Svcs.MongoDB)
		if err != nil {
			return nil, err
		}

		item.Details = userDBItem.Info
	}

	return &protos.UserGroupJoinListResp{
		Requests: items,
	}, nil
}
