package wsHandler

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/logger"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func HandleSelectedScanEntriesReq(req *protos.SelectedScanEntriesReq, hctx wsHelpers.HandlerContext) ([]*protos.SelectedScanEntriesResp, error) {
	// We read multiple scans and assemble a single response
	// If any have an error, we error the whole thing out

	// Cap it though for performance...
	if len(req.ScanIds) > 10 {
		return nil, errors.New("Too many ScanIds requested")
	}

	result := map[string]*protos.ScanEntryRange{}

	for _, scanId := range req.ScanIds {
		if err := wsHelpers.CheckStringField(&scanId, "scanId", 1, wsHelpers.IdFieldMaxLength); err != nil {
			return nil, err
		}

		idxs, err := readSelection("entry_"+scanId+"_"+hctx.SessUser.User.Id, hctx.Svcs.MongoDB)
		if err != nil {
			return nil, err
		}

		result[scanId] = idxs
	}

	return []*protos.SelectedScanEntriesResp{&protos.SelectedScanEntriesResp{
		ScanIdEntryIndexes: result,
	}}, nil
}

// Allowing user to save multiple scans worth of entry indexes in one message
func HandleSelectedScanEntriesWriteReq(req *protos.SelectedScanEntriesWriteReq, hctx wsHelpers.HandlerContext) ([]*protos.SelectedScanEntriesWriteResp, error) {
	// Cap it though for performance...
	if len(req.ScanIdEntryIndexes) > 10 {
		return nil, errors.New("Too many ScanIds written")
	}

	for scanId, idxs := range req.ScanIdEntryIndexes {
		err := writeSelection("entry_"+scanId+"_"+hctx.SessUser.User.Id, idxs, hctx.Svcs.MongoDB, hctx.Svcs.Log)
		if err != nil {
			return nil, err
		}
	}

	return []*protos.SelectedScanEntriesWriteResp{&protos.SelectedScanEntriesWriteResp{}}, nil
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

	// Modified and Upsert counts will be 0 if the selection hasn't changed, so we just check matched
	if dbResult.MatchedCount != 1 {
		logger.Errorf("writeSelection (%v) UpdateByID result had unexpected counts %+v", id, dbResult)
	}

	return nil
}
