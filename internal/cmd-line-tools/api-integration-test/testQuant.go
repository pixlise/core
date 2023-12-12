package main

import (
	"context"
	"log"
	"os"

	"github.com/pixlise/core/v3/api/dbCollections"
	"github.com/pixlise/core/v3/core/wstestlib"
	protos "github.com/pixlise/core/v3/generated-protos"
)

func testQuants(apiHost string) {
	testQuantCreate(apiHost)
	return
	testQuantFit(apiHost)
	testQuantUpload(apiHost)
	testQuantGetListDelete(apiHost)
	//testMultiQuant(apiHost)
}

// Helper functions

func seedDBQuants(quants []*protos.QuantificationSummary) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.QuantificationsName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = db.CreateCollection(ctx, dbCollections.QuantificationsName)
	if err != nil {
		log.Fatal(err)
	}

	if len(quants) > 0 {
		items := []interface{}{}
		for _, q := range quants {
			items = append(items, q)
		}
		_, err = coll.InsertMany(ctx, items, nil)
		if err != nil {
			log.Fatalln(err)
		}
	}
}

func seedQuantFile(fileName string, s3Path string /*userId string, scanId string*/, bucket string) {
	data, err := os.ReadFile("./test-files/" + fileName)
	if err != nil {
		log.Fatalln(err)
	}

	// Upload it where we need it for the test
	//s3Path := filepaths.GetQuantPath(userId, scanId, fileName)
	err = apiStorageFileAccess.WriteObject(bucket, s3Path, data)
	if err != nil {
		log.Fatalln(err)
	}
}
