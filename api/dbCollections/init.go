package dbCollections

import (
	"context"
	"log"

	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
)

func InitCollections(db *mongo.Database, iLog logger.ILogger, environment string) {
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
		UserROIDisplaySettings,
		WidgetDataName,
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

	// We want to be able to watch change streams on some of our collections... DocumentDB seems to require this to be enabled
	// separately, so do that here
	// if environment != "unittest" && environment != "prodMigrated" {
	// Tried with an env, got AWS DocumentDB error: The modifyChangeStreams command can only be run against the admin database
	// Had to run this manually on DB with mongosh (connected to the DocumentDB cluster): db.adminCommand({modifyChangeStreams: 1, database: "pixlise-feature-v4", collection: "jobStatuses", enable: true})
	// 	result := db.RunCommand(ctx, bson.D{
	// 		{"modifyChangeStreams", 1},
	// 		{"database", db.Name()},
	// 		{"collection", JobStatusName},
	// 		{"enable", true},
	// 	})
	// 	if result != nil {
	// 		log.Fatal(result.Err())
	// 	}
	// }
}
