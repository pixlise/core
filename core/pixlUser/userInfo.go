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

package pixlUser

import (
	"context"
	"errors"

	mongoDBConnection "github.com/pixlise/core/v2/core/mongo"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type UserDetailsLookup struct {
	mongo *mongo.Client

	userDatabase   *mongo.Database
	userCollection *mongo.Collection

	cache map[string]UserDetails
}

func MakeUserDetailsLookup( /*timeStamper timestamper.ITimeStamper,*/ mongoClient *mongo.Client, envName string) UserDetailsLookup {
	userDatabaseName := mongoDBConnection.GetUserDatabaseName(envName)

	userDatabase := mongoClient.Database(userDatabaseName)
	userCollection := userDatabase.Collection("users")

	return UserDetailsLookup{
		//timeStamper: timeStamper,
		mongo: mongoClient,

		userDatabase:   userDatabase,
		userCollection: userCollection,

		cache: map[string]UserDetails{},
	}
}

func (u *UserDetailsLookup) createUser(userid string, name string, email string) (UserStruct, error) {
	us := UserStruct{
		Userid: userid,
		Notifications: Notifications{
			Topics:          []Topics{},
			Hints:           []string{},
			UINotifications: []UINotificationItem{},
		},
		Config: UserDetails{
			DataCollection: "unknown",
			Name:           name,
			Email:          email,
			Cell:           "", // TODO: delete this? It went unused
		},
	}

	if u.userCollection == nil {
		return us, errors.New("createUser: Mongo not connected")
	}

	//n.Logger.Debugf("Creating Mongo Object for user: %v", user.Userid)

	_, err := u.userCollection.InsertOne(context.TODO(), us)

	//n.Logger.Debugf("Created Mongo Object for user: %v", user.Userid)

	if err == nil {
		// Add entry into cache
		u.cache[userid] = us.Config
	}

	return us, err
}

func (u *UserDetailsLookup) GetUser(userid string) (UserStruct, error) {
	if u.userCollection == nil {
		return UserStruct{}, errors.New("GetUser: Mongo not connected")
	}

	//n.Logger.Debugf("Fetching Mongo Object for user: %v", userid)

	filter := bson.D{{"userid", userid}}

	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.FindOne().SetSort(sort) //.SetProjection(projection)
	cursor := u.userCollection.FindOne(context.TODO(), filter, opts)

	var user UserStruct

	//n.Logger.Debugf("Decoding Mongo Object for user: %v", userid)
	err := cursor.Decode(&user)
	//n.Logger.Debugf("Fetched Mongo Object for user: %v", userid)

	// Update cache if it worked
	if err == nil {
		u.cache[userid] = user.Config
	}

	return user, err
}

func (u *UserDetailsLookup) GetUserEnsureExists(userid string, name string, email string) (UserStruct, error) {
	// Try to read it, if it doesn't exist, create it
	user, err := u.GetUser(userid)

	// If user doesn't exist, we create it
	if err == mongo.ErrNoDocuments {
		return u.createUser(userid, name, email)
	}

	// Otherwise return whatever we got
	return user, err
}

// Getting JUST UserDetails (so it goes through our in-memory cache). This is useful for the many places in the code that only
// require user name+email to ensure we're sending out up-to-date "creator" aka "APIObjectItem" structures
func (u *UserDetailsLookup) GetCurrentCreatorDetails(userID string) (UserInfo, error) {
	details, ok := u.cache[userID]

	if !ok {
		// We don't have it! Read from user DB & return that
		readUser, err := u.GetUser(userID)

		if err != nil {
			return UserInfo{}, err
		}

		details = readUser.Config
	}

	// Return as a UserInfo
	result := UserInfo{
		UserID:      userID,
		Name:        details.Name,
		Email:       details.Email,
		Permissions: map[string]bool{},
	}

	return result, nil
}

func (u *UserDetailsLookup) WriteUser(user UserStruct) error {
	if u.userCollection == nil {
		return errors.New("WriteUser: Mongo not connected")
	}

	//n.Logger.Debugf("Updating Mongo Object for user: %v", userid)

	filter := bson.D{{"userid", user.Userid}}
	update := bson.D{{"$set", user}}
	opts := options.Update().SetUpsert(true)

	_, err := u.userCollection.UpdateOne(context.TODO(), filter, update, opts)
	//n.Logger.Debugf("Updated Mongo Object for user: %v", userid)

	if err == nil {
		// Update entry in cache
		u.cache[user.Userid] = user.Config
	}

	return err
}
