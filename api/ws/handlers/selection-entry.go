package wsHandler

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/api/ws/wsHelpers"
	"github.com/pixlise/core/v3/core/logger"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleSelectedScanEntriesReq(req *protos.SelectedScanEntriesReq, hctx wsHelpers.HandlerContext) (*protos.SelectedScanEntriesResp, error) {
	idxs, err := readSelection("entry_"+req.ScanId+"_"+hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
	if err != nil {
		return nil, err
	}

	return &protos.SelectedScanEntriesResp{
		EntryIndexes: idxs,
	}, nil
}

func HandleSelectedScanEntriesWriteReq(req *protos.SelectedScanEntriesWriteReq, hctx wsHelpers.HandlerContext) (*protos.SelectedScanEntriesWriteResp, error) {
	err := writeSelection("entry_"+req.ScanId+"_"+hctx.SessUser.User.Id, req.EntryIndexes, hctx.Svcs.MongoDB, hctx.Svcs.Log)
	if err != nil {
		return nil, err
	}

	return &protos.SelectedScanEntriesWriteResp{}, nil
}

func readSelection(id string, db *mongo.Database) (*protos.ScanEntryRange, error) {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.SelectionName)

	result := coll.FindOne(ctx, bson.M{"_id": id})
	if result.Err() != nil {
		if result.Err() == mongo.ErrNoDocuments {
			// No selection, return blank, not an error
			return &protos.ScanEntryRange{
				Indexes: []int32{},
			}, nil
		}
		return nil, result.Err()
	}

	idxRange := protos.ScanEntryRange{}
	err := result.Decode(&idxRange)
	if err != nil {
		return nil, err
	}

	return &idxRange, nil
}

func writeSelection(id string, idxs *protos.ScanEntryRange, db *mongo.Database, logger logger.ILogger) error {
	ctx := context.TODO()
	coll := db.Collection(dbCollections.SelectionName)
	opts := options.Update().SetUpsert(true)

	dbResult, err := coll.UpdateByID(ctx, id, bson.D{{Key: "$set", Value: idxs}}, opts)
	if err != nil {
		return err
	}

	if dbResult.UpsertedCount != 1 && dbResult.ModifiedCount != 1 {
		logger.Errorf("writeSelection UpdateByID result had unexpected counts %+v", dbResult)
	}

	return nil
}
