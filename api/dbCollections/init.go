package dbCollections

import (
	"context"
	"log"

	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func InitCollections(db *mongo.Database, iLog logger.ILogger) {
	// Ensure collections exist, required because some collections are "first" written to in a transaction which fails
	// if the collection doesn't already exist
	collectionsRequired := []string{
		QuantificationsName,
		OwnershipName,
		ElementSetsName,
		ExpressionGroupsName,
		ExpressionsName,
		ModulesName,
		ModuleVersionsName,
		RegionsOfInterestName,
		ScreenConfigurationName,
	}

	ctx := context.TODO()
	existingCollections, err := db.ListCollectionNames(ctx, bson.D{})
	if err != nil {
		log.Fatal(err)
	}

	for _, collName := range collectionsRequired {
		if !utils.ItemInSlice(collName, existingCollections) {
			// Doesn't exist, create it
			iLog.Infof("Mongo collection %v doesn't exist, pre-creating it...", collName)
			err = db.CreateCollection(ctx, collName)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
}