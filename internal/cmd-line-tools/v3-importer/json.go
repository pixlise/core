package main

import (
	"context"
	"fmt"
	"log"
	"path/filepath"
	"strings"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func importJSONFiles(jsonImportDir string, destDB *mongo.Database) error {
	if len(jsonImportDir) <= 0 {
		fmt.Println("SKIPPED - jsonImportDir is empty")
		return nil
	}

	fs := fileaccess.FSAccess{}
	files, err := fs.ListObjects(jsonImportDir, "")
	if err != nil {
		return err
	}

	// Run through and make sure each is a collection
	collNames := dbCollections.GetAllCollections()

	for _, f := range files {
		if !strings.HasSuffix(f, ".json") {
			return fmt.Errorf("File type not valid: %v", f)
		}

		collName := f[0 : len(f)-5]
		if !utils.ItemInSlice(collName, collNames) {
			return fmt.Errorf("File name is not a collection name: %v", f)
		}

		// Read it into the specified collection
		v := []interface{}{}
		err = fs.ReadJSON(filepath.Join(jsonImportDir, f), "", &v, false)
		if err != nil {
			return fmt.Errorf("Failed to read json file %v. Error: %v", f, err)
		}

		// We have it as an object, should be able to add to DB
		coll := destDB.Collection(collName)
		result, err := coll.InsertMany(context.TODO(), v, options.InsertMany())

		if err != nil || len(result.InsertedIDs) != len(v) {
			if mongo.IsDuplicateKeyError(err) {
				log.Printf("WARNING: Importing %v caused duplicate key error: %v. Continuing...", f, err)
			} else {
				return fmt.Errorf("Failed to insert into DB from json file %v. Error: %v", f, err)
			}
		} else {
			fmt.Printf("Imported: %v\n", f)
		}
	}

	return nil
}
