package wsHelpers

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/semanticversion"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func GetModuleVersion(moduleID string, version *protos.SemanticVersion, db *mongo.Database) (*protos.DataModuleVersion, error) {
	// NOTE: This was initially built with a query:
	// filter := bson.D{primitive.E{Key: "moduleid", Value: moduleID}, primitive.E{Key: "version", Value: version}}
	// But now ID is composed of these fields so it's more direct to query by ID
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ModuleVersionsName)

	result := &protos.DataModuleVersion{}
	id := moduleID + "-v" + semanticversion.SemanticVersionToString(version)
	verResult := coll.FindOne(ctx, bson.M{"_id": id})

	if verResult.Err() != nil {
		return nil, verResult.Err()
	}

	// Read the module item
	err := verResult.Decode(&result)
	return result, err
}
