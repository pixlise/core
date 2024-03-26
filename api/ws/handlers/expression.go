package wsHandler

import (
	"context"
	"errors"

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

func HandleExpressionGetReq(req *protos.ExpressionGetReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.DataExpression](false, req.Id, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return &protos.ExpressionGetResp{
		Expression: dbItem,
	}, nil
}

func HandleExpressionDeleteReq(req *protos.ExpressionDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.ExpressionDeleteResp](req.Id, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
}

func HandleExpressionListReq(req *protos.ExpressionListReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionListResp, error) {
	filter, idToOwner, err := wsHelpers.MakeFilter(req.SearchParams, false, protos.ObjectType_OT_EXPRESSION, hctx)
	if err != nil {
		return nil, err
	}

	// Since we want only summary data, specify less fields to retrieve
	opts := options.Find().SetProjection(bson.D{
		{Key: "_id", Value: true},
		{Key: "name", Value: true},
		{Key: "sourcelanguage", Value: true},
		{Key: "comments", Value: true},
		{Key: "tags", Value: true},
		{Key: "modulereferences", Value: true},
		{Key: "modifiedunixsec", Value: true},
	})

	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionsName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.DataExpression{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Transform to map of output values
	itemMap := map[string]*protos.DataExpression{}
	for _, item := range items {
		if owner, ok := idToOwner[item.Id]; ok {
			item.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}
		itemMap[item.Id] = item
	}

	return &protos.ExpressionListResp{
		Expressions: itemMap,
	}, nil
}

func validateExpression(expr *protos.DataExpression) error {
	if err := wsHelpers.CheckStringField(&expr.Name, "Name", 1, 50); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&expr.Comments, "Comments", 0, wsHelpers.DescriptionFieldMaxLength); err != nil {
		return err
	}
	if err := wsHelpers.CheckStringField(&expr.SourceCode, "SourceCode", 1, wsHelpers.SourceCodeMaxLength); err != nil {
		return err
	}
	if expr.SourceLanguage != "LUA" && expr.SourceLanguage != "PIXLANG" {
		return errors.New("Invalid source language: " + expr.SourceLanguage)
	}
	if err := wsHelpers.CheckFieldLength(expr.Tags, "Tags", 0, wsHelpers.TagListMaxLength); err != nil {
		return err
	}
	if err := wsHelpers.CheckFieldLength(expr.ModuleReferences, "ModuleReferences", 0, 10); err != nil {
		return err
	}

	return nil
}

func createExpression(expr *protos.DataExpression, hctx wsHelpers.HandlerContext) (*protos.DataExpression, error) {
	ctx := context.TODO()

	// It's a new item, check these fields...
	err := validateExpression(expr)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a new id
	id := hctx.Svcs.IDGen.GenObjectID()
	expr.Id = id

	// We need to create an ownership item along with it
	ownerItem := wsHelpers.MakeOwnerForWrite(id, protos.ObjectType_OT_EXPRESSION, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())

	expr.ModifiedUnixSec = ownerItem.CreatedUnixSec

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
		_, _err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionsName).InsertOne(sessCtx, expr)
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
	expr.Owner = wsHelpers.MakeOwnerSummary(ownerItem, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return expr, nil
}

func updateExpression(expr *protos.DataExpression, hctx wsHelpers.HandlerContext) (*protos.DataExpression, error) {
	ctx := context.TODO()

	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.DataExpression](true, expr.Id, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
	if err != nil {
		return nil, err
	}

	// Update fields
	update := bson.D{}
	if len(expr.Name) > 0 {
		dbItem.Name = expr.Name
		update = append(update, bson.E{Key: "name", Value: expr.Name})
	}

	if len(expr.Comments) > 0 {
		dbItem.Comments = expr.Comments
		update = append(update, bson.E{Key: "comments", Value: expr.Comments})
	}

	if len(expr.Tags) > 0 {
		dbItem.Tags = expr.Tags
		update = append(update, bson.E{Key: "tags", Value: expr.Tags})
	}

	if len(expr.ModuleReferences) > 0 {
		dbItem.ModuleReferences = expr.ModuleReferences
		update = append(update, bson.E{Key: "modulereferences", Value: expr.ModuleReferences})
	}

	if len(expr.SourceCode) > 0 {
		dbItem.SourceCode = expr.SourceCode
		update = append(update, bson.E{Key: "sourcecode", Value: expr.SourceCode})
	}

	if len(expr.SourceLanguage) > 0 {
		dbItem.SourceLanguage = expr.SourceLanguage
		update = append(update, bson.E{Key: "sourcelanguage", Value: expr.SourceLanguage})
	}

	// Validate it
	err = validateExpression(dbItem)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Update modified time
	dbItem.ModifiedUnixSec = uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	update = append(update, bson.E{Key: "modifiedunixsec", Value: dbItem.ModifiedUnixSec})

	// It's valid, update the DB
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionsName).UpdateByID(ctx, expr.Id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("DataExpression UpdateByID result had unexpected counts %+v id: %v", result, expr.Id)
	}

	// Return the merged item we validated, which in theory is in the DB now
	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return dbItem, nil
}

func HandleExpressionWriteReq(req *protos.ExpressionWriteReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionWriteResp, error) {
	// Owner should never be accepted from API
	if req.Expression.Owner != nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Owner must be empty for write messages"))
	}

	var item *protos.DataExpression
	var err error

	if len(req.Expression.Id) <= 0 {
		item, err = createExpression(req.Expression, hctx)
	} else {
		item, err = updateExpression(req.Expression, hctx)
	}
	if err != nil {
		return nil, err
	}

	return &protos.ExpressionWriteResp{
		Expression: item,
	}, nil
}

func HandleExpressionWriteExecStatReq(req *protos.ExpressionWriteExecStatReq, hctx wsHelpers.HandlerContext) (*protos.ExpressionWriteExecStatResp, error) {
	// Validate request
	if err := wsHelpers.CheckStringField(&req.Id, "Id", 0, wsHelpers.IdFieldMaxLength); err != nil {
		return nil, err
	}

	if req.Stats == nil || len(req.Stats.DataRequired) <= 0 || req.Stats.RuntimeMsPer1000Pts < 0 {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Invalid stats in request"))
	}

	ctx := context.TODO()

	// NOTE: This is special!
	// We almost always have to make sure the user is an "editor" of the object in question, but here we're harvesting usage stats
	// from anyones machine who can "view" the expression, so we only check view level permissions, and yet allow them to save
	// runtime stats for it!
	_, _, err := wsHelpers.GetUserObjectById[protos.DataExpression](false, req.Id, protos.ObjectType_OT_EXPRESSION, dbCollections.ExpressionsName, hctx)
	if err != nil {
		return nil, err
	}

	// Replace its recent exec stats with the one given to us
	req.Stats.TimeStampUnixSec = uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	update := bson.D{{Key: "$set", Value: bson.D{{Key: "recentexecstats", Value: req.Stats}}}}

	// It's valid, update the DB
	filter := bson.D{{Key: "_id", Value: req.Id}}
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.ExpressionsName).UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("DataExpression ExecStatWrite UpdateByID result had unexpected counts %+v id: %v", result, req.Id)
	}

	return &protos.ExpressionWriteExecStatResp{}, nil
}
