package wsHelpers

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

type ItemWithId struct {
	Id string `bson:"_id,omitempty"`
}

func DeleteUserObject[T any](objectId string, objectType protos.ObjectType, collectionName string, hctx HandlerContext) (*T, error) {
	_, err := DeleteUserObjectByIdField("_id", objectId, objectType, true, collectionName, hctx)
	if err != nil {
		return nil, err
	}

	// Delete responses are just empty msgs
	var resp T
	return &resp, nil
}

func DeleteUserObjectByIdField(idField string, objectId string, objectType protos.ObjectType, deleteOneOnly bool, collectionName string, hctx HandlerContext) (int64, error) {
	ctx := context.TODO()

	_, err := CheckObjectAccess(true, objectId, objectType, hctx)
	if err != nil {
		return 0, err
	}

	// Delete element set AND corresponding ownership item
	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := hctx.Svcs.MongoDB.Client().StartSession()
	if err != nil {
		return 0, err
	}
	defer sess.EndSession(ctx)

	// Deleting in a single transaction - we have 2 copies of this, one calls DeleteOne, the other DeleteMany
	callbackDeleteOne := func(sessCtx mongo.SessionContext) (interface{}, error) {
		result, err := hctx.Svcs.MongoDB.Collection(collectionName).DeleteOne(context.TODO(), bson.M{idField: objectId})
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

		return 1, nil
	}

	callbackDeleteMany := func(sessCtx mongo.SessionContext) (interface{}, error) {
		// First, lets get the ids of the individual ROI items here
		filter := bson.M{idField: objectId}
		cursor, err := hctx.Svcs.MongoDB.Collection(collectionName).Find(context.TODO(), filter, options.Find().SetProjection(bson.D{{Key: "_id", Value: true}}))
		ids := []*ItemWithId{}
		err = cursor.All(context.TODO(), &ids)
		if err != nil {
			return nil, errorwithstatus.MakeBadRequestError(err)
		}

		delResult, err := hctx.Svcs.MongoDB.Collection(collectionName).DeleteMany(context.TODO(), bson.M{idField: objectId})
		if err != nil {
			return nil, errorwithstatus.MakeBadRequestError(err)
		}

		idList := []string{}
		for _, id := range ids {
			idList = append(idList, id.Id)
		}

		_, err = hctx.Svcs.MongoDB.Collection(dbCollections.OwnershipName).DeleteMany(context.TODO(), bson.M{"_id": bson.M{"$in": idList}})
		if err != nil {
			return nil, errorwithstatus.MakeBadRequestError(err)
		}

		return delResult.DeletedCount, nil
	}

	callback := callbackDeleteMany
	if deleteOneOnly {
		callback = callbackDeleteOne
	}

	result, err := sess.WithTransaction(ctx, callback, txnOpts)
	if err != nil {
		return 0, err
	}

	delCount, ok := result.(int64)
	if !ok {
		hctx.Svcs.Log.Errorf("Expected transaction to return delete count, but got: %v", result)
		return 0, nil
	}

	return delCount, nil
}

func GetUserObjectById[T any](forEditing bool, objectId string, objectType protos.ObjectType, collectionName string, hctx HandlerContext) (*T, *protos.OwnershipItem, error) {
	owner, err := CheckObjectAccess(forEditing, objectId, objectType, hctx)
	if err != nil {
		return nil, nil, err
	}

	result := hctx.Svcs.MongoDB.Collection(collectionName).FindOne(context.TODO(), bson.M{"_id": objectId})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, nil, errorwithstatus.MakeNotFoundError(objectId)
		}

		return nil, nil, result.Err()
	}

	var dbItem T
	err = result.Decode(&dbItem)
	return &dbItem, owner, err
}

func MakeFilter(
	searchParams *protos.SearchParams,
	requireEdit bool,
	objectType protos.ObjectType,
	hctx HandlerContext) (bson.M, map[string]*protos.OwnershipItem, error) {

	// Firstly, get the list of ids that are accessible to this user, based on ownership
	idToOwner, err := ListAccessibleIDs(false, objectType, hctx.Svcs, hctx.SessUser)
	if err != nil {
		return nil, idToOwner, err
	}

	if searchParams != nil && len(searchParams.CreatorUserId) > 0 {
		// Filter any out which are not by the requested creator
		for id, owner := range idToOwner {
			if owner.CreatorUserId != searchParams.CreatorUserId {
				delete(idToOwner, id)
			}
		}
	}

	ids := utils.GetMapKeys(idToOwner)

	filter := bson.M{"_id": bson.M{"$in": ids}}

	// Now apply any search params to it
	if searchParams != nil {
		if len(searchParams.ScanId) > 0 {
			filter["scanid"] = searchParams.ScanId
		}
		if len(searchParams.NameSearch) > 0 {
			filter["name"] = bson.M{"$regex": searchParams.NameSearch}
		}
		if len(searchParams.TagId) > 0 {
			filter["tags"] = searchParams.TagId
		}
	}

	return filter, idToOwner, nil
}
