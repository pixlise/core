package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

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
