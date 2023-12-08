package quantification

import (
	"context"

	"github.com/pixlise/core/v3/api/dbCollections"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readconcern"
	"go.mongodb.org/mongo-driver/mongo/writeconcern"
)

func writeQuantAndOwnershipToDB(summary *protos.QuantificationSummary, owner *protos.OwnershipItem, db *mongo.Database) error {
	ctx := context.TODO()

	wc := writeconcern.New(writeconcern.WMajority())
	rc := readconcern.Snapshot()
	txnOpts := options.Transaction().SetWriteConcern(wc).SetReadConcern(rc)

	sess, err := db.Client().StartSession()
	if err != nil {
		return err
	}
	defer sess.EndSession(ctx)

	// Write the 2 items in a single transaction
	callback := func(sessCtx mongo.SessionContext) (interface{}, error) {
		_, _err := db.Collection(dbCollections.QuantificationsName).InsertOne(sessCtx, summary)
		if _err != nil {
			return nil, _err
		}
		_, _err = db.Collection(dbCollections.OwnershipName).InsertOne(sessCtx, owner)
		if _err != nil {
			return nil, _err
		}
		return nil, nil
	}

	_, err = sess.WithTransaction(ctx, callback, txnOpts)

	return err
}
