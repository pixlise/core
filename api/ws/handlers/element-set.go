package wsHandler

import (
	"context"
	"errors"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const elementSetCollection = "elementSets"

func HandleElementSetDeleteReq(req *protos.ElementSetDeleteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetDeleteResp, error) {
	return nil, errors.New("HandleElementSetDeleteReq not implemented yet")
}

func HandleElementSetGetReq(req *protos.ElementSetGetReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ElementSetGetResp, error) {
	// Read from DB too
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
	if len(req.ElementSet.Id) > 0 {
		// It's an overwrite operation...
		_, err := svcs.MongoDB.Collection(elementSetCollection).UpdateByID(context.TODO(), req.Id, req.ElementSet)
		if err != nil {
			return nil, err
		}
	} else {
		// Generate a new id
		id := svcs.IDGen.GenObjectID()
		req.ElementSet.Id = id

		_, err := svcs.MongoDB.Collection(elementSetCollection).InsertOne(context.TODO(), req.ElementSet)
		if err != nil {
			return nil, err
		}
	}

	return &protos.ElementSetWriteResp{}, nil
}
