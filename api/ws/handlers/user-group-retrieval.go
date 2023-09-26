package wsHandler

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
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
	if err != nil {
		return nil, err
	}

	groups := []*protos.UserGroupDB{}
	err = cursor.All(context.TODO(), &groups)
	if err != nil {
		return nil, err
	}

	// Just sending back the "info" part
	groupInfos := []*protos.UserGroupInfo{}
	for _, group := range groups {
		userRelationship := protos.UserGroupRelationship_UGR_UNKNOWN
		if utils.ItemInSlice(hctx.SessUser.User.Id, group.AdminUserIds) {
			userRelationship = protos.UserGroupRelationship_UGR_ADMIN
		} else if utils.ItemInSlice(hctx.SessUser.User.Id, group.Members.UserIds) {
			userRelationship = protos.UserGroupRelationship_UGR_MEMBER
		} else if utils.ItemInSlice(hctx.SessUser.User.Id, group.Viewers.UserIds) {
			userRelationship = protos.UserGroupRelationship_UGR_VIEWER
		}

		groupInfos = append(groupInfos, &protos.UserGroupInfo{
			Id:                    group.Id,
			Name:                  group.Name,
			Description:           group.Description,
			CreatedUnixSec:        group.CreatedUnixSec,
			LastUserJoinedUnixSec: group.LastUserJoinedUnixSec,
			RelationshipToUser:    userRelationship,
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

func HandleUserGroupListJoinableReq(req *protos.UserGroupListJoinableReq, hctx wsHelpers.HandlerContext) (*protos.UserGroupListJoinableResp, error) {
	// Should only be called if we have admin rights, so other permission issues here
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UserGroupsName)

	filter := bson.D{}
	// opts := options.Find()
	// Since we want only summary data, specify less fields to retrieve
	opts := options.Find().SetProjection(bson.D{
		{"_id", true},
		{"name", true},
		{"description", true},
		{"adminuserids", true},
		{"lastuserjoinedunixsec", true},
		{"uniqueusercount", true},
	})
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	groups := []*protos.UserGroupDB{}
	err = cursor.All(context.TODO(), &groups)
	if err != nil {
		return nil, err
	}

	groupSummaries := []*protos.UserGroupJoinSummaryInfo{}
	for _, group := range groups {
		// TODO: This seems to just be listing to count how many there are. Do we need to read the scans? Presumably a length of ids
		// returned is enough. If we DO need to read the scans, we should project them to be as small as possible (or maybe we can ask
		// mongo to count them and return a single number!)
		idToOwner, err := wsHelpers.ListGroupAccessibleIDs(false, protos.ObjectType_OT_SCAN, group.Id, hctx.Svcs.MongoDB)
		if err != nil {
			return nil, err
		}

		ids := utils.GetMapKeys(idToOwner)

		filter := bson.M{"_id": bson.M{"$in": ids}}

		opts = options.Find()
		cursor, err = hctx.Svcs.MongoDB.Collection(dbCollections.ScansName).Find(context.TODO(), filter, opts)
		if err != nil {
			return nil, err
		}

		scans := []*protos.ScanItem{}
		err = cursor.All(context.TODO(), &scans)
		if err != nil {
			return nil, err
		}

		admins := []*protos.UserInfo{}
		for _, adminId := range group.AdminUserIds {
			user := &protos.UserInfo{}
			if dbUser, err := wsHelpers.GetDBUser(adminId, hctx.Svcs.MongoDB); err != nil {
				// Print an error but return an empty user struct
				hctx.Svcs.Log.Errorf("Failed to find user info for user-group %v %v user ID %v", group.Name, group.Id, adminId)
				user = &protos.UserInfo{
					Id: adminId,
				}
			} else {
				user = dbUser.Info
			}

			admins = append(admins, user)
		}

		// uniqueusercount

		groupSummaries = append(groupSummaries, &protos.UserGroupJoinSummaryInfo{
			Id:                    group.Id,
			Name:                  group.Name,
			Description:           group.Description,
			Administrators:        admins,
			Datasets:              uint32(len(scans)),
			LastUserJoinedUnixSec: group.LastUserJoinedUnixSec,
		})
	}

	return &protos.UserGroupListJoinableResp{
		Groups: groupSummaries,
	}, nil
}
