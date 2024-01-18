package piquant

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetPiquantVersion(svcs *services.APIServices) (*protos.PiquantVersion, error) {
	// Look up the PIQUANT version currently set
	result := svcs.MongoDB.Collection(dbCollections.PiquantVersionName).FindOne(context.TODO(), bson.M{"_id": "current"})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			return nil, errorwithstatus.MakeNotFoundError("PIQUANT version")
		}
		return nil, result.Err()
	}

	ver := &protos.PiquantVersion{}
	err := result.Decode(ver)
	if err != nil {
		return nil, err
	}

	return ver, nil
}
