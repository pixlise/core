package wsHandler

import (
	"context"

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

	groups := []*protos.UserGroupDB{}
	err = cursor.All(context.TODO(), &groups)
	if err != nil {
		return nil, err
	}

	// Just sending back the "info" part
	groupInfos := []*protos.UserGroupInfo{}
	for _, group := range groups {
		groupInfos = append(groupInfos, &protos.UserGroupInfo{
			Id:             group.Id,
			Name:           group.Name,
			CreatedUnixSec: group.CreatedUnixSec,
		})
	}

	return &protos.UserGroupListResp{
		GroupInfos: groupInfos,
	}, nil
}

// Getting an individual user group - this should only be allowed for PIXLISE_ADMIN permissioned users, or group admins
func HandleUserGroupReq(req *protos.UserGroupReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupResp, error) {
	if err := wsHelpers.CheckStringField(&req.GroupId, "GroupId", 1, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	// Read this one from DB
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

	decGroup, err := decorateUserGroup(&group, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return &protos.UserGroupResp{
		Group: decGroup,
	}, nil
}
