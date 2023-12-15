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
	testMultiQuant(apiHost)
	testQuantFit(apiHost)
	testQuantUpload(apiHost)
	testQuantGetListDelete(apiHost)
}

// Helper functions

func resetDBPiquantAndJobs() {
	db := wstestlib.GetDB()
	ctx := context.TODO()
	// Seed jobs
	coll := db.Collection(dbCollections.JobStatusName)
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = db.CreateCollection(ctx, dbCollections.JobStatusName)
	if err != nil {
		log.Fatal(err)
	}

	// Seed piquant versions
	coll = db.Collection(dbCollections.PiquantVersionName)
	err = coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = db.CreateCollection(ctx, dbCollections.PiquantVersionName)
	if err != nil {
		log.Fatal(err)
	}
	insertResult, err := coll.InsertOne(context.TODO(), &protos.PiquantVersion{
		Id:              "current",
		Version:         "registry.gitlab.com/pixlise/piquant/runner:3.2.16",
		ModifiedUnixSec: 1234567890,
		ModifierUserId:  "user-123",
	})
	if err != nil || insertResult.InsertedID != "current" {
		panic(err)
	}
}

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

func seedS3File(fileName string, s3Path string /*userId string, scanId string*/, bucket string) {
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

func seedDBROI(rois []*protos.ROIItem) {
	db := wstestlib.GetDB()
	coll := db.Collection(dbCollections.RegionsOfInterestName)
	ctx := context.TODO()
	err := coll.Drop(ctx)
	if err != nil {
		log.Fatal(err)
	}
	err = db.CreateCollection(ctx, dbCollections.RegionsOfInterestName)
	if err != nil {
		log.Fatal(err)
	}

	if len(rois) > 0 {
		items := []interface{}{}
		for _, r := range rois {
			items = append(items, r)
		}
		_, err = coll.InsertMany(ctx, items, nil)
		if err != nil {
			log.Fatalln(err)
		}
	}
}
