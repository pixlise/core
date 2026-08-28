package wsHandler

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

func HandleSourceRepositoryListReq(req *protos.SourceRepositoryListReq, hctx wsHelpers.HandlerContext) (*protos.SourceRepositoryListResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.SourceRepositoriesName)

	cursor, err := coll.Find(ctx, bson.M{})
	if err != nil {
		return nil, fmt.Errorf("Failed to read source repositories: %v", err)
	}

	repos := []*protos.SourceRepository{}
	err = cursor.All(ctx, &repos)
	if err != nil {
		return nil, fmt.Errorf("Failed to decode source repositories: %v", err)
	}

	return &protos.SourceRepositoryListResp{
		Repositories: repos,
	}, nil
}

func validateRepo(repository *protos.SourceRepository) error {
	if len(repository.Name) <= 0 {
		return fmt.Errorf("Name must be set")
	}
	if len(repository.Url) <= 0 {
		return fmt.Errorf("URL must be set")
	}
	if len(repository.User) <= 0 {
		return fmt.Errorf("User must be set")
	}
	if len(repository.Secret) <= 0 {
		return fmt.Errorf("Secret must be set")
	}

	return nil
}

func HandleSourceRepositorySetReq(req *protos.SourceRepositorySetReq, hctx wsHelpers.HandlerContext) (*protos.SourceRepositorySetResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.SourceRepositoriesName)

	repo := req.Repository

	// Validate it first
	if err := validateRepo(repo); err != nil {
		return nil, fmt.Errorf("SourceRepositorySetReq given invalid repo: %v", err)
	}

	if len(repo.Id) <= 0 {
		// Insert with a new id
		repo.Id = hctx.Svcs.IDGen.GenObjectID()
		result, err := coll.InsertOne(ctx, repo)
		if err != nil {
			return nil, fmt.Errorf("Failed to insert new repository with id: %v. Error: %v", repo.Id, err)
		}

		if result.InsertedID != repo.Id {
			hctx.Svcs.Log.Errorf("Inserted repository id %v doesn't match specified one: %v", result.InsertedID, repo.Id)
		}
	} else {
		// Update
		result, err := coll.UpdateOne(ctx, bson.M{"_id": repo.Id}, bson.D{{Key: "$set", Value: repo}})

		if err != nil {
			return nil, fmt.Errorf("Failed to set source repository \"%v\": %v", repo.Id, err)
		}

		if result.MatchedCount == 0 && result.ModifiedCount == 0 && result.UpsertedCount == 0 {
			return nil, fmt.Errorf("Failed to update source repository \"%v\": Not found", repo.Id)
		}

		if result.ModifiedCount != 1 && result.UpsertedCount != 1 {
			hctx.Svcs.Log.Infof("HandleSourceRepositorySetReq: Unexpected result %+v", result)
		}
	}

	return &protos.SourceRepositorySetResp{Repository: repo}, nil
}

func HandleSourceRepositoryDeleteReq(req *protos.SourceRepositoryDeleteReq, hctx wsHelpers.HandlerContext) (*protos.SourceRepositoryDeleteResp, error) {
	ctx := context.TODO()
	coll := hctx.Svcs.MongoDB.Collection(dbCollections.SourceRepositoriesName)

	// Delete tag
	delResult, err := coll.DeleteOne(ctx, bson.M{"_id": req.Id})
	if err != nil {
		return nil, err
	}

	if delResult.DeletedCount != 1 {
		return nil, fmt.Errorf("Error deleting source repository %v returned delete count of %v", req.Id, delResult.DeletedCount)
	}

	return &protos.SourceRepositoryDeleteResp{}, nil
}
