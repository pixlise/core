package wsHandler

import (
	"context"
	"errors"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/errorwithstatus"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const elementSetCollection = "elementSets"

func HandleElementSetDeleteReq(req *protos.ElementSetDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetDeleteResp, error) {
	result, err := svcs.MongoDB.Collection(elementSetCollection).DeleteOne(context.TODO(), bson.M{"_id": req.Id})
	if err != nil {
		return nil, errorwithstatus.MakeBadRequestError(err)
	}

	if result.DeletedCount != 1 {
		return nil, errorwithstatus.MakeNotFoundError(req.Id)
	}

	return &protos.ElementSetDeleteResp{}, nil
}

func HandleElementSetGetReq(req *protos.ElementSetGetReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetGetResp, error) {
	result := svcs.MongoDB.Collection(elementSetCollection).FindOne(context.TODO(), bson.M{"_id": req.Id})
	if result.Err() != nil {
		return nil, result.Err()
	}

	dbItem := protos.ElementSet{}
	err := result.Decode(&dbItem)
	if err != nil {
		return nil, err
	}

	return &protos.ElementSetGetResp{
		ElementSet: &dbItem,
	}, nil
}

func HandleElementSetListReq(req *protos.ElementSetListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetListResp, error) {
	filter := bson.D{}
	opts := options.Find()
	cursor, err := svcs.MongoDB.Collection(elementSetCollection).Find(context.TODO(), filter, opts)
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
			Id:            item.Id,
			Name:          item.Name,
			AtomicNumbers: z,
			Owner:         item.Owner,
		}
	}

	return &protos.ElementSetListResp{
		ElementSets: itemMap,
	}, nil
}

func HandleElementSetWriteReq(req *protos.ElementSetWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetWriteResp, error) {
	// Owner should never be accepted from API
	saveOwner, err := wsHelpers.MakeOwnerForWrite(req.ElementSet.Owner, s, svcs)
	if err != nil {
		return nil, err
	}
	req.ElementSet.Owner = saveOwner

	resp := &protos.ElementSetWriteResp{}

	if len(req.ElementSet.Id) > 0 {
		// It's an overwrite operation... check that fields are valid and build a list of what we're setting
		update := bson.D{}

		if len(req.ElementSet.Name) > 50 {
			return nil, errorwithstatus.MakeBadRequestError(errors.New("Name length is invalid"))
		} else if len(req.ElementSet.Name) > 0 {
			update = append(update, bson.E{Key: "name", Value: req.ElementSet.Name})
		}
		if len(req.ElementSet.Lines) > 118 /*Max Z*/ {
			return nil, errorwithstatus.MakeBadRequestError(errors.New("Lines length is invalid"))
		} else if len(req.ElementSet.Lines) > 0 {
			update = append(update, bson.E{Key: "lines", Value: req.ElementSet.Lines})
		}

		_, err := svcs.MongoDB.Collection(elementSetCollection).UpdateByID(context.TODO(), req.ElementSet.Id, bson.D{{Key: "$set", Value: update}})
		if err != nil {
			return nil, err
		}

		// TODO: is there a better way than another query?
		// Query the current edited object so we can return it
		result := svcs.MongoDB.Collection(elementSetCollection).FindOne(context.TODO(), bson.M{"_id": req.ElementSet.Id})
		if result.Err() != nil {
			return nil, result.Err()
		}

		dbItem := protos.ElementSet{}
		err = result.Decode(&dbItem)
		if err != nil {
			return nil, err
		}

		resp.ElementSet = &dbItem
	} else {
		// It's a new item, check these fields...
		if len(req.ElementSet.Name) <= 0 || len(req.ElementSet.Name) > 50 {
			return nil, errorwithstatus.MakeBadRequestError(errors.New("Name length is invalid"))
		}
		if len(req.ElementSet.Lines) <= 0 || len(req.ElementSet.Lines) > 118 /*Max Z*/ {
			return nil, errorwithstatus.MakeBadRequestError(errors.New("Lines length is invalid"))
		}

		// Generate a new id
		id := svcs.IDGen.GenObjectID()
		req.ElementSet.Id = id

		_, err := svcs.MongoDB.Collection(elementSetCollection).InsertOne(context.TODO(), req.ElementSet)
		if err != nil {
			return nil, err
		}

		resp.ElementSet = req.ElementSet
	}

	return resp, nil
}
