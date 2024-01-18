// Licensed to NASA JPL under one or more contributor
// license agreements. See the NOTICE file distributed with
// this work for additional information regarding copyright
// ownership. NASA JPL licenses this file to you under
// the Apache License, Version 2.0 (the "License"); you may
// not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing,
// software distributed under the License is distributed on an
// "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
// KIND, either express or implied.  See the License for the
// specific language governing permissions and limitations
// under the License.

package importerutils

import (
	"context"

	"github.com/pixlise/core/v4/api/dbCollections"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ReadScanItem(scanId string, db *mongo.Database) (*protos.ScanItem, error) {
	filter := bson.M{"_id": scanId}
	opts := options.FindOne()
	result := db.Collection(dbCollections.ScansName).FindOne(context.TODO(), filter, opts)
	if result.Err() != nil {
		return nil, result.Err()
	}

	item := &protos.ScanItem{}
	err := result.Decode(item)

	if err != nil {
		return nil, err
	}

	return item, nil
}
