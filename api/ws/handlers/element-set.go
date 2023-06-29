package wsHandler

import (
	"context"
	"errors"
	"fmt"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	"github.com/pixlise/core/v3/core/utils"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleElementSetDeleteReq(req *protos.ElementSetDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetDeleteResp, error) {
	_, err := wsHelpers.CheckObjectAccess(true, req.Id, protos.ObjectType_OT_ELEMENT_SET, s, svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	result, err := svcs.MongoDB.Collection(dbCollections.ElementSetsName).DeleteOne(context.TODO(), bson.M{"_id": req.Id})
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	if result.DeletedCount != 1 {
		return nil, errorwithstatus.MakeNotFoundError(req.Id)
	}

	return &protos.ElementSetDeleteResp{}, nil
}

func HandleElementSetGetReq(req *protos.ElementSetGetReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetGetResp, error) {
	owner, err := wsHelpers.CheckObjectAccess(false, req.Id, protos.ObjectType_OT_ELEMENT_SET, s, svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	dbItem, err := getElementSet(req.Id, svcs)
	if err != nil {
		return nil, err
	}

	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, svcs.MongoDB)
	return &protos.ElementSetGetResp{
		ElementSet: dbItem,
	}, nil
}

func HandleElementSetListReq(req *protos.ElementSetListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetListResp, error) {
	objAndUsers, err := wsHelpers.ListAccessibleIDs(false, protos.ObjectType_OT_ELEMENT_SET, s, svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	filter := bson.M{"_id": bson.M{"$in": utils.GetStringMapStringKeys(objAndUsers)}}
	opts := options.Find()
	cursor, err := svcs.MongoDB.Collection(dbCollections.ElementSetsName).Find(context.TODO(), filter, opts)
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
		itemMap[item.Id] = &protos.ElementSetSummary{
			Id:             item.Id,
			Name:           item.Name,
			AtomicNumbers:  z,
			ModifedUnixSec: item.ModifedUnixSec,
			Owner:          item.Owner, // TODO: set this, we have user IDs above but really need ownership struct so we can call MakeOwnerSummary()
		}
	}

	return &protos.ElementSetListResp{
		ElementSets: itemMap,
	}, nil
}

func validateElementSet(elementSet *protos.ElementSet) error {
	if len(elementSet.Name) <= 0 || len(elementSet.Name) > 50 {
		return errors.New("Name length is invalid")
	}
	if len(elementSet.Lines) <= 0 || len(elementSet.Lines) > 118 /*Max Z*/ {
		return errors.New("Lines length is invalid")
	}
	return nil
}

func getElementSet(id string, svcs *services.APIServices) (*protos.ElementSet, error) {
	result := svcs.MongoDB.Collection(dbCollections.ElementSetsName).FindOne(context.TODO(), bson.M{"_id": id})
	if result.Err() != nil {
		return nil, result.Err()
	}

	dbItem := &protos.ElementSet{}
	err := result.Decode(dbItem)
	return dbItem, err
}

func createElementSet(elementSet *protos.ElementSet, s *melody.Session, svcs *services.APIServices) (*protos.ElementSet, error) {
	ctx := context.TODO()

	// It's a new item, check these fields...
	err := validateElementSet(elementSet)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	// Generate a new id
	id := svcs.IDGen.GenObjectID()
	elementSet.Id = id

	// We need to create an ownership item along with it
	ownerItem, err := wsHelpers.MakeOwnerForWrite(id, protos.ObjectType_OT_ELEMENT_SET, s, svcs)
	if err != nil {
		return nil, err
	}

	sess, err := svcs.MongoDB.Client().StartSession()
	if err != nil {
		return nil, err
	}
	defer sess.EndSession(ctx)

	// Write the 2 items in a single transaction
	result, err := sess.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		if _, err := svcs.MongoDB.Collection(dbCollections.ElementSetsName).InsertOne(sessCtx, elementSet); err != nil {
			return nil, err
		}
		if _, err := svcs.MongoDB.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, ownerItem); err != nil {
			return nil, err
		}
		return nil, nil
	})

	if err != nil {
		return nil, err
	}

	fmt.Printf("WithTransaction result: %+v", result)

	return elementSet, nil
}

func updateElementSet(elementSet *protos.ElementSet, s *melody.Session, svcs *services.APIServices) (*protos.ElementSet, error) {
	ctx := context.TODO()

	owner, err := wsHelpers.CheckObjectAccess(true, elementSet.Id, protos.ObjectType_OT_ELEMENT_SET, s, svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	// First, we read the existing object, so we can validate it together
	dbItem, err := getElementSet(elementSet.Id, svcs)
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to find element set to update: %v. Error: %v", elementSet.Id, err))
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
	update = append(update, bson.E{Key: "modifedUnixSec", Value: svcs.TimeStamper.GetTimeNowSec()})

	// It's valid, update the DB
	result, err := svcs.MongoDB.Collection(dbCollections.ElementSetsName).UpdateByID(ctx, elementSet.Id, bson.D{{Key: "$set", Value: update}})
	if err != nil {
		return nil, err
	}

	if result.MatchedCount != 1 {
		svcs.Log.Errorf("Element Set UpdateByID result had unexpected counts %+v id: %v", result, elementSet.Id)
	}

	// Return the merged item we validated, which in theory is in the DB now
	dbItem.Owner = wsHelpers.MakeOwnerSummary(owner, svcs.MongoDB)
	return dbItem, nil
}

func HandleElementSetWriteReq(req *protos.ElementSetWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetWriteResp, error) {
	// Owner should never be accepted from API
	if req.ElementSet.Owner != nil {
		return nil, errorwithstatus.MakeBadRequestError(errors.New("Owner must be empty for write messages"))
	}

	var item *protos.ElementSet
	var err error

	if len(req.ElementSet.Id) <= 0 {
		item, err = createElementSet(req.ElementSet, s, svcs)
	} else {
		item, err = updateElementSet(req.ElementSet, s, svcs)
	}
	if err != nil {
		return nil, err
	}

	resp := &protos.ElementSetWriteResp{}
	resp.ElementSet = item

	return resp, nil
}
