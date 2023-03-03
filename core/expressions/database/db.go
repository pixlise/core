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

package expressionDB

import (
	"github.com/pixlise/core/v2/api/services"
	mongoDBConnection "github.com/pixlise/core/v2/core/mongo"
	"go.mongodb.org/mongo-driver/mongo"
)

func MakeExpressionDB(
	envName string,
	svcs *services.APIServices,
) *ExpressionDB {
	exprDB := mongoDBConnection.GetDatabaseName("expressions", envName)

	db := svcs.Mongo.Database(exprDB)

	expressions := db.Collection("expressions")
	modules := db.Collection("modules")
	moduleVersions := db.Collection("moduleVersions")

	return &ExpressionDB{
		Svcs: svcs,

		Database:       db,
		Expressions:    expressions,
		Modules:        modules,
		ModuleVersions: moduleVersions,
	}
}

type ExpressionDB struct {
	Svcs *services.APIServices

	Database       *mongo.Database
	Expressions    *mongo.Collection
	Modules        *mongo.Collection
	ModuleVersions *mongo.Collection
}

func (e *ExpressionDB) IsNotFoundError(err error) bool {
	errStr := err.Error()
	return errStr == "mongo: no documents in result"
}
