package wsHelpers

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func DeleteUserObject[T any](objectId string, objectType protos.ObjectType, collectionName string, hctx HandlerContext) (*T, error) {
	ctx := context.TODO()

	_, err := CheckObjectAccess(true, objectId, objectType, hctx)
	if err != nil {
		return nil, err
	}

	// Delete element set AND corresponding ownership item
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
		result, err := hctx.Svcs.MongoDB.Collection(collectionName).DeleteOne(context.TODO(), bson.M{"_id": objectId})
		if err != nil {
			return nil, errorwithstatus.MakeBadRequestError(err)
		}

		if result.DeletedCount != 1 {
			return nil, errorwithstatus.MakeNotFoundError(objectId)
		}

		result, err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).DeleteOne(context.TODO(), bson.M{"_id": objectId})
		if err != nil {
			return nil, errorwithstatus.MakeBadRequestError(err)
		}

		if result.DeletedCount != 1 {
			return nil, errorwithstatus.MakeNotFoundError(objectId)
		}

		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)
	if err != nil {
		return nil, err
	}

	// Delete responses are just empty msgs
	var resp T
	return &resp, nil
}

func GetUserObjectById[T any](forEditing bool, objectId string, objectType protos.ObjectType, collectionName string, hctx HandlerContext) (*T, *protos.OwnershipItem, error) {
	owner, err := CheckObjectAccess(forEditing, objectId, objectType, hctx)
	if err != nil {
		return nil, nil, err
	}

	result := hctx.Svcs.MongoDB.Collection(collectionName).FindOne(context.TODO(), bson.M{"_id": objectId})
	if result.Err() != nil {
		return nil, nil, result.Err()
	}

	var dbItem T
	err = result.Decode(&dbItem)
	return &dbItem, owner, err
}
