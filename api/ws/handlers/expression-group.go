package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func HandleExpressionGroupDeleteReq(req *protos.ExpressionGroupDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.ExpressionGroupDeleteResp](req.Id, protos.ObjectType_OT_EXPRESSION_GROUP, dbCollections.ExpressionGroupsName, hctx)
}

func HandleExpressionGroupListReq(req *protos.ExpressionGroupListReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupListResp, error) {
	filter, idToOwner, err := wsHelpers.MakeFilter(req.SearchParams, false, protos.ObjectType_OT_EXPRESSION_GROUP, hctx)
	if err != nil {
		return nil, err
	}

	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionGroupsName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ExpressionGroup{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Transform to map of output values
	itemMap := map[string]*protos.ExpressionGroup{}
	for _, item := range items {
		if owner, ok := idToOwner[item.Id]; ok {
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}
		itemMap[item.Id] = item
	}

	return &protos.ExpressionGroupListResp{
		Groups: itemMap,
	}, nil
}

func HandleExpressionGroupGetReq(req *protos.ExpressionGroupGetReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ExpressionGroup](false, req.Id, protos.ObjectType_OT_EXPRESSION_GROUP, dbCollections.ExpressionGroupsName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return &protos.ExpressionGroupGetResp{
		Group: dbItem,
	}, nil
}

func validateExpressionGroup(egroup *protos.ExpressionGroup) error {
	if err := wsHelpers.CheckStringField(&egroup.Name, "Name", 1, 50); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&egroup.Description, "Description", 0, wsHelpers.DescriptionFieldMaxLength); err != nil {
		return err
	}
	if err := wsHelpers.CheckFieldLength(egroup.Tags, "Tags", 0, wsHelpers.TagListMaxLength); err != nil {
		return err
	}
	if err := wsHelpers.CheckFieldLength(egroup.GroupItems, "GroupItems", 2, 5); err != nil {
		return err
	}

	for c, g := range egroup.GroupItems {
		if err := wsHelpers.CheckStringField(&g.ExpressionId, fmt.Sprintf("[%v].ExpressionId", c), 1, wsHelpers.IdFieldMaxLength); err != nil {
			return err
		}
	}

	return nil
}

func createExpressionGroup(egroup *protos.ExpressionGroup, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroup, error) {
	ctx := context.TODO()

	// It's a new item, check these fields...
	err := validateExpressionGroup(egroup)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a new id
	// NOTE: Previously we had a thing called an RGB mix, which was a group of 3 expressions. We have now transitioned to having
	//       the more generic mechanism of an expression group. RGB mixes were all saved prefixed with "rgbmix-", which we didn't
	//       change in the migration tool because there are many places they could be stored, but now we call prefix them with
	//       "grp-". This should be fine, except anywhere where we're checking if the ID is for a group or an expression we need
	//       to check for both!
	id := "grp-" + hctx.Svcs.IDGen.GenObjectID()
	egroup.Id = id

	// We need to create an ownership item along with it
	ownerItem, err := wsHelpers.MakeOwnerForWrite(id, protos.ObjectType_OT_EXPRESSION_GROUP, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())
	if err != nil {
		return nil, err
	}

	egroup.ModifiedUnixSec = ownerItem.CreatedUnixSec

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	// Write the 2 items in a single transaction
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, _err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionGroupsName).InsertOne(sessCtx, egroup)
		if _err != nil {
			return nil, _err
		}
		_, _err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, ownerItem)
		if _err != nil {
			return nil, _err
		}
		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)

	if err != nil {
		return nil, err
	}

	egroup.Owner = wsHelpers.MakeOwnerSummary(ownerItem, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return egroup, nil
}

func updateExpressionGroup(egroup *protos.ExpressionGroup, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroup, error) {
	ctx := context.TODO()

	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ExpressionGroup](true, egroup.Id, protos.ObjectType_OT_EXPRESSION_GROUP, dbCollections.ExpressionGroupsName, hctx)
	if err != nil {
		return nil, err
	}

	// Update fields
	update := bson.D{}
	if len(egroup.Name) > 0 {
		dbItem.Name = egroup.Name
		update = append(update, bson.E{Key: "name", Value: egroup.Name})
	}

	if len(egroup.Description) > 0 {
		dbItem.Description = egroup.Description
		update = append(update, bson.E{Key: "description", Value: egroup.Description})
	}

	if len(egroup.Tags) > 0 {
		dbItem.Tags = egroup.Tags
		update = append(update, bson.E{Key: "tags", Value: egroup.Tags})
	}

	if len(egroup.GroupItems) > 0 {
		dbItem.GroupItems = egroup.GroupItems
		update = append(update, bson.E{Key: "groupitems", Value: egroup.GroupItems})
	}

	// Validate it
	err = validateExpressionGroup(dbItem)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Update modified time
	dbItem.ModifiedUnixSec = uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	update = append(update, bson.E{Key: "modifiedunixsec", Value: dbItem.ModifiedUnixSec})

	// It's valid, update the DB
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionGroupsName).UpdateByID(ctx, egroup.Id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("ExpresssionGroup UpdateByID result had unexpected counts %+v id: %v", result, egroup.Id)
	}

	// Return the merged item we validated, which in theory is in the DB now
	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return dbItem, nil
}

func HandleExpressionGroupWriteReq(req *protos.ExpressionGroupWriteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGroupWriteResp, error) {
	// Owner should never be accepted from API
	if req.Group.Owner != nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Owner must be empty for write messages"))
	}

	var item *protos.ExpressionGroup
	var err error

	if len(req.Group.Id) <= 0 {
		item, err = createExpressionGroup(req.Group, hctx)
	} else {
		item, err = updateExpressionGroup(req.Group, hctx)
	}
	if err != nil {
		return nil, err
	}

	return &protos.ExpressionGroupWriteResp{
		Group: item,
	}, nil
}
