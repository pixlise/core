package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func checkTagExists(name string, ctx context.Context, coll *mongo.Collection) (bool, error, *mongo.SingleResult) {
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

func decorateTag(tag *protos.TagDB, db *mongo.Database, log logger.ILogger) (*protos.Tag, error) {
	decoratedTag := &protos.Tag{
		Id:     tag.Id,
		Name:   tag.Name,
		Type:   tag.Type,
		ScanId: tag.ScanId,
	}

	var user *protos.UserInfo

	// Look up user by ownerId
	if owner, err := wsHelpers.GetDBUser(tag.OwnerId, db); err != nil {
		// Print an error but return an empty user struct
		// log.Errorf("Failed to find user info for owner %v of tag %v (%v)", tag.OwnerId, tag.Id, tag.Name)
		user = &protos.UserInfo{
			Id: tag.OwnerId,
		}
	} else {
		user = &protos.UserInfo{
			Id:      owner.Id,
			Name:    owner.Info.Name,
			Email:   owner.Info.Email,
			IconURL: owner.Info.IconURL,
		}
	}

	decoratedTag.Owner = user

	return decoratedTag, nil
}

func HandleTagCreateReq(req *protos.TagCreateReq, hctx wsHelpers.HandlerContext) ([]*protos.TagCreateResp, error) {
	// Limit tags to 50 characters
	if err := wsHelpers.CheckStringField(&req.Name, "Name", 1, 50); err != nil {
		return nil, err
	}

	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.TagsName)

	exists, err, _ := checkTagExists(req.Name, ctx, coll)
	if err != nil {
		return nil, err
	} else if exists {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf(`Tag: "%v" already exists`, req.Name))
	}

	// At this point we should know that the name is not taken
	tagId := hctx.Svcs.IDGen.GenObjectID()

	tag := &protos.TagDB{
		Id:      tagId,
		Name:    req.Name,
		Type:    req.Type,
		ScanId:  req.ScanId,
		OwnerId: hctx.SessUser.User.Id,
	}

	_, _err := coll.InsertOne(ctx, tag)
	if _err != nil {
		return nil, _err
	}

	resolvedTag, err := decorateTag(tag, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return []*protos.TagCreateResp{&protos.TagCreateResp{Tag: resolvedTag}}, nil
}

func HandleTagDeleteReq(req *protos.TagDeleteReq, hctx wsHelpers.HandlerContext) ([]*protos.TagDeleteResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.TagsName)

	// Check if tag exists and is owned by user
	filter := bson.M{"$and": []interface{}{
		bson.M{"_id": req.TagId},
		bson.M{"ownerid": hctx.SessUser.User.Id},
	}}
	cursor, err := coll.Find(ctx, filter)
	if err != nil {
		return nil, err
	}

	// If user doesn't own tag, return error
	if !cursor.Next(ctx) {
		return nil, errorwithstatus.MakeUnauthorisedError(fmt.Errorf("User does not own tag: %v", req.TagId))
	}

	// Delete tag
	_, err = coll.DeleteOne(ctx, bson.M{"_id": req.TagId})
	if err != nil {
		return nil, err
	}

	return []*protos.TagDeleteResp{&protos.TagDeleteResp{}}, nil
}

func HandleTagListReq(req *protos.TagListReq, hctx wsHelpers.HandlerContext) ([]*protos.TagListResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.TagsName)

	filter := bson.D{}
	opts := options.Find()
	cursor, err := coll.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}

	tags := []*protos.TagDB{}
	err = cursor.All(context.TODO(), &tags)
	if err != nil {
		return nil, err
	}

	decoratedTags := []*protos.Tag{}
	for _, tag := range tags {
		decoratedTag, _ := decorateTag(tag, hctx.Svcs.MongoDB, hctx.Svcs.Log)
		decoratedTags = append(decoratedTags, decoratedTag)
	}

	return []*protos.TagListResp{&protos.TagListResp{Tags: decoratedTags}}, nil
}
