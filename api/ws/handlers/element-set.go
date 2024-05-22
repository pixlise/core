package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func HandleElementSetDeleteReq(req *protos.ElementSetDeleteReq, hctx wsHelpers.HandlerContext) (*protos.ElementSetDeleteResp, error) {
	return wsHelpers.DeleteUserObject[protos.ElementSetDeleteResp](req.Id, protos.ObjectType_OT_ELEMENT_SET, dbCollections.ElementSetsName, hctx)
}

func HandleElementSetGetReq(req *protos.ElementSetGetReq, hctx wsHelpers.HandlerContext) (*protos.ElementSetGetResp, error) {
	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ElementSet](false, req.Id, protos.ObjectType_OT_ELEMENT_SET, dbCollections.ElementSetsName, hctx)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return &protos.ElementSetGetResp{
		ElementSet: dbItem,
	}, nil
}

func HandleElementSetListReq(req *protos.ElementSetListReq, hctx wsHelpers.HandlerContext) (*protos.ElementSetListResp, error) {
	idToOwner, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_ELEMENT_SET, hctx.Svcs, hctx.SessUser)
	if err != nil {
		return nil, err
	}

	ids := utils.GetMapKeys(idToOwner)

	filter := bson.M{"_id": bson.M{"$in": ids}}
	opts := options.Find()
	cursor, err := hctx.Svcs.MongoDB.Collection(dbCollections.ElementSetsName).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	items := []*protos.ElementSet{}
	err = cursor.All(context.TODO(), &items)
	if err != nil {
		return nil, err
	}

	// Transform to map of output values
	itemMap := map[string]*protos.ElementSetSummary{}
	for _, item := range items {
		z := []int32{}
		for _, l := range item.Lines {
			z = append(z, l.Z)
		}

		summary := &protos.ElementSetSummary{
			Id:              item.Id,
			Name:            item.Name,
			AtomicNumbers:   z,
			ModifiedUnixSec: item.ModifiedUnixSec,
		}

		if owner, ok := idToOwner[item.Id]; ok {
			summary.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
		}

		itemMap[item.Id] = summary
	}

	return &protos.ElementSetListResp{
		ElementSets: itemMap,
	}, nil
}

func validateElementSet(elementSet *protos.ElementSet) error {
	if err := wsHelpers.CheckStringField(&elementSet.Name, "Name", 1, 50); err != nil {
		return err
	}
	return wsHelpers.CheckFieldLength(elementSet.Lines, "Lines", 1, 118 /*Max Z*/)
}

func createElementSet(elementSet *protos.ElementSet, hctx wsHelpers.HandlerContext) (*protos.ElementSet, error) {
	ctx := context.TODO()

	// It's a new item, check these fields...
	err := validateElementSet(elementSet)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a new id
	id := hctx.Svcs.IDGen.GenObjectID()
	elementSet.Id = id

	// We need to create an ownership item along with it
	ownerItem := wsHelpers.MakeOwnerForWrite(id, protos.ObjectType_OT_ELEMENT_SET, hctx.SessUser.User.Id, hctx.Svcs.TimeStamper.GetTimeNowSec())

	elementSet.ModifiedUnixSec = ownerItem.CreatedUnixSec

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
		_, _err := hctx.Svcs.MongoDB.Collection(dbCollections.ElementSetsName).InsertOne(sessCtx, elementSet)
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

	elementSet.Owner = wsHelpers.MakeOwnerSummary(ownerItem, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return elementSet, nil
}

func updateElementSet(elementSet *protos.ElementSet, hctx wsHelpers.HandlerContext) (*protos.ElementSet, error) {
	ctx := context.TODO()

	dbItem, owner, err := wsHelpers.GetUserObjectById[protos.ElementSet](true, elementSet.Id, protos.ObjectType_OT_ELEMENT_SET, dbCollections.ElementSetsName, hctx)
	if err != nil {
		return nil, err
	}

	// Update fields
	update := bson.D{}
	if len(elementSet.Name) > 0 {
		dbItem.Name = elementSet.Name
		update = append(update, bson.E{Key: "name", Value: elementSet.Name})
	}

	if len(elementSet.Lines) > 0 {
		dbItem.Lines = elementSet.Lines
		update = append(update, bson.E{Key: "lines", Value: elementSet.Lines})
	}

	// Validate it
	err = validateElementSet(dbItem)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Update modified time
	dbItem.ModifiedUnixSec = uint32(hctx.Svcs.TimeStamper.GetTimeNowSec())
	update = append(update, bson.E{Key: "modifiedunixsec", Value: dbItem.ModifiedUnixSec})

	// It's valid, update the DB
	result, err := hctx.Svcs.MongoDB.Collection(dbCollections.ElementSetsName).UpdateByID(ctx, elementSet.Id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		hctx.Svcs.Log.Errorf("Element Set UpdateByID result had unexpected counts %+v id: %v", result, elementSet.Id)
	}

	// Return the merged item we validated, which in theory is in the DB now
	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, hctx.SessUser, hctx.Svcs.MongoDB, hctx.Svcs.TimeStamper)
	return dbItem, nil
}

func HandleElementSetWriteReq(req *protos.ElementSetWriteReq, hctx wsHelpers.HandlerContext) (*protos.ElementSetWriteResp, error) {
	// Owner should never be accepted from API
	if req.ElementSet.Owner != nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Owner must be empty for write messages"))
	}

	var item *protos.ElementSet
	var err error

	if len(req.ElementSet.Id) <= 0 {
		item, err = createElementSet(req.ElementSet, hctx)
	} else {
		item, err = updateElementSet(req.ElementSet, hctx)
	}
	if err != nil {
		return nil, err
	}

	return &protos.ElementSetWriteResp{
		ElementSet: item,
	}, nil
}
