package wsHandler

import (
	"context"
	"errors"
	"sort"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func HandleUserDetailsReq(req *protos.UserDetailsReq, hctx wsHelpers.HandlerContext) ([]*protos.UserDetailsResp, error) {
	userDBItem, err := wsHelpers.GetDBUser(hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	perms := utils.GetMapKeys(hctx.SessUser.Permissions)

	// Sort for consistant ordering, we don't know what order will prevail coming from JWT and our
	// map doesn't help either!
	sort.Strings(perms)

	return []*protos.UserDetailsResp{&protos.UserDetailsResp{
		Details: &protos.UserDetails{
			Info:                  userDBItem.Info,
			DataCollectionVersion: userDBItem.DataCollectionVersion,
			Permissions:           perms,
		},
	}}, nil
}

func HandleUserDetailsWriteReq(req *protos.UserDetailsWriteReq, hctx wsHelpers.HandlerContext) ([]*protos.UserDetailsWriteResp, error) {
	if &req.Name != nil && req.Name != "" {
		if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 50); err != nil {
			return nil, err
		}
	}

	if &req.Email != nil && req.Email != "" {
		if err := wsHelpers.CheckStringField(&req.Email, "Email", 1, 320); err != nil {
			return nil, err
		}
	}

	// Limit to 30kb because it can be a URL or a data://<base64 image>
	if err := wsHelpers.CheckStringField(&req.IconURL, "IconURL", 0, 30000); err != nil {
		return nil, err
	}

	if &req.DataCollectionVersion != nil && req.DataCollectionVersion != "" {
		if err := wsHelpers.CheckStringField(&req.DataCollectionVersion, "DataCollectionVersion", 1, 20); err != nil {
			return nil, err
		}
	}

	update := bson.D{}
	if &req.Name != nil && req.Name != "" {
		update = append(update, bson.E{Key: "info.name", Value: req.Name})
	}
	if &req.Email != nil && req.Email != "" {
		update = append(update, bson.E{Key: "info.email", Value: req.Email})
	}
	if &req.IconURL != nil {
		update = append(update, bson.E{Key: "info.iconurl", Value: req.IconURL})
	}
	if &req.DataCollectionVersion != nil && req.DataCollectionVersion != "" {
		update = append(update, bson.E{Key: "datacollectionversion", Value: req.DataCollectionVersion})
	}

	if len(update) <= 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("No fields to change"))
	}

	// It's valid, update the DB
	ctx := context.TODO()
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.UsersName).UpdateByID(ctx, hctx.SessUser.User.Id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("UserDetails UpdateByID result had unexpected counts %+v id: %v", result, hctx.SessUser.User.Id)
	}

	// Notify our cache that this user changed, so we ensure things sent out will have the right
	// user info on them
	wsHelpers.NotifyUserInfoChange(hctx.SessUser.User.Id)

	// TODO: Trigger user details update (?)

	return []*protos.UserDetailsWriteResp{&protos.UserDetailsWriteResp{}}, nil
}

func HandleUserSearchReq(req *protos.UserSearchReq, hctx wsHelpers.HandlerContext) ([]*protos.UserSearchResp, error) {
	if err := wsHelpers.CheckStringField(&req.SearchString, "SearchString", 0, 100); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.UsersName)

	filter := bson.M{"$or": []interface{}{
		bson.M{"info.name": bson.M{"$regex": req.SearchString}},
		bson.M{"info.email": bson.M{"$regex": req.SearchString}},
	}}

	//opts := options.Find()
	cursor, err := coll.Find(ctx, filter) //, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			// Silent error, just return empty
			return []*protos.UserSearchResp{&protos.UserSearchResp{
				Users: []*protos.UserInfo{},
			}}, nil
		}

		return nil, err
	}

	result := []*protos.UserDetails{}
	err = cursor.All(ctx, &result)
	if err != nil {
		return nil, err
	}

	users := []*protos.UserInfo{}
	for _, user := range result {
		users = append(users, user.Info)
	}

	return []*protos.UserSearchResp{&protos.UserSearchResp{
		Users: users,
	}}, nil
}
