// Copyright (c) 2018-2022 California Institute of Technology (“Caltech”). U.S.
// Government sponsorship acknowledged.
// All rights reserved.
// Redistribution and use in source and binary forms, with or without
// modification, are permitted provided that the following conditions are
// met:
//
// * Redistributions of source code must retain the above copyright notice, this
//   list of conditions and the following disclaimer.
// * Redistributions in binary form must reproduce the above copyright notice,
//   this list of conditions and the following disclaimer in the documentation
//   and/or other materials provided with the distribution.
// * Neither the name of Caltech nor its operating division, the Jet Propulsion
//   Laboratory, nor the names of its contributors may be used to endorse or
//   promote products derived from this software without specific prior written
//   permission.
//
// THIS SOFTWARE IS PROVIDED BY THE COPYRIGHT HOLDERS AND CONTRIBUTORS "AS IS"
// AND ANY EXPRESS OR IMPLIED WARRANTIES, INCLUDING, BUT NOT LIMITED TO, THE
// IMPLIED WARRANTIES OF MERCHANTABILITY AND FITNESS FOR A PARTICULAR PURPOSE
// ARE DISCLAIMED. IN NO EVENT SHALL THE COPYRIGHT OWNER OR CONTRIBUTORS BE
// LIABLE FOR ANY DIRECT, INDIRECT, INCIDENTAL, SPECIAL, EXEMPLARY, OR
// CONSEQUENTIAL DAMAGES (INCLUDING, BUT NOT LIMITED TO, PROCUREMENT OF
// SUBSTITUTE GOODS OR SERVICES; LOSS OF USE, DATA, OR PROFITS; OR BUSINESS
// INTERRUPTION) HOWEVER CAUSED AND ON ANY THEORY OF LIABILITY, WHETHER IN
// CONTRACT, STRICT LIABILITY, OR TORT (INCLUDING NEGLIGENCE OR OTHERWISE)
// ARISING IN ANY WAY OUT OF THE USE OF THIS SOFTWARE, EVEN IF ADVISED OF THE
// POSSIBILITY OF SUCH DAMAGE.

package notifications

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/pixlise/core/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"log"
	"time"
)

type mongoutils struct {
	client                 *mongo.Client
	userDatabase           *mongo.Database
	userCollection         *mongo.Collection
	notificationCollection *mongo.Collection
}

func (m *mongoutils) Connect() error {
	cmdMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			log.Print(evt.Command)
		},
	}
	//ctx := context.Background()
	var err error
	m.client, err = mongo.NewClient(options.Client().ApplyURI("mongodb://localhost").SetMonitor(cmdMonitor))
	//client, err := mongo.NewClient(options.Client().ApplyURI("mongodb://localhost"))
	/*if err != nil {
		return nil, err
	}*/
	ctx, _ := context.WithTimeout(context.Background(), 10*time.Second)
	err = m.client.Connect(ctx)
	if err != nil {
		return err
	}
	//defer client.Disconnect(ctx)

	m.userDatabase = m.client.Database("userdatabase")
	m.userCollection = m.userDatabase.Collection("users")
	return nil
}

func (m *mongoutils) GetAllMongoUsers(log logger.ILogger) ([]UserStruct, error) {

	filter := bson.D{}
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := m.userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var notifications []UserStruct

	for cursor.Next(context.Background()) {
		l := UserStruct{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, l)
	}

	return notifications, nil
}

func (m *mongoutils) GetMongoSubscribersByTopicID(override []string, searchtopic string, logger logger.ILogger) ([]UserStruct, error) {
	var filter bson.M
	if override != nil && len(override) > 0 {
		filter = bson.M{
			"$and": []bson.D{
				{
					{"userid", bson.D{{"$in", override}}},
				},
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	} else {
		filter = bson.M{
			"$and": []bson.D{
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	}
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	jsonString, _ := json.Marshal(filter)
	fmt.Printf("\n%v\n", string(jsonString))
	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := m.userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var notifications []UserStruct

	for cursor.Next(context.Background()) {
		l := UserStruct{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, l)
	}

	return notifications, nil
}

func (m *mongoutils) GetMongoSubscribersByEmailTopicID(override []string, searchtopic string, logger logger.ILogger) ([]UserStruct, error) {
	var filter bson.M
	if override != nil && len(override) > 0 {
		filter = bson.M{
			"$and": []bson.D{
				{
					{"userconfig.email", bson.D{{"$in", override}}},
				},
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	} else {
		filter = bson.M{
			"$and": []bson.D{
				{
					{
						Key:   "notifications.topics.name",
						Value: searchtopic,
					},
				},
			},
		}
	}
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	jsonString, _ := json.Marshal(filter)
	fmt.Printf("\n%v\n", string(jsonString))
	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := m.userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var notifications []UserStruct

	for cursor.Next(context.Background()) {
		l := UserStruct{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, l)
	}

	return notifications, nil
}

func (m *mongoutils) GetMongoSubscribersByTopic(searchtopic string, logger logger.ILogger) ([]UserStruct, error) {
	var filter bson.M

	filter = bson.M{
		"$and": []bson.D{
			{
				{
					Key:   "notifications.topics.name",
					Value: searchtopic,
				},
			},
		},
	}

	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	jsonString, _ := json.Marshal(filter)
	fmt.Printf("\n%v\n", string(jsonString))
	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := m.userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var notifications []UserStruct

	for cursor.Next(context.Background()) {
		l := UserStruct{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, l)
	}

	return notifications, nil
}

func (m *mongoutils) InsertUINotification(newNotification UINotificationObj) error {
	_, err := m.userCollection.InsertOne(context.TODO(), newNotification)
	if err != nil {
		return err
	}
	return nil
}

func (m *mongoutils) GetUINotifications(user string) ([]UINotificationObj, error) {
	filter := bson.D{{"userid", user}}
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}

	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := m.userCollection.Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	var notifications []UINotificationObj

	for cursor.Next(context.Background()) {
		l := UINotificationObj{}
		err := cursor.Decode(&l)
		if err != nil {
			return nil, err
		}
		notifications = append(notifications, l)
	}

	return notifications, nil
}

func (m *mongoutils) DeleteUINotifications(user string) error {

	filter := bson.D{{"userid", user}}

	_, err := m.userCollection.DeleteMany(context.TODO(), filter)
	return err
}

func (m *mongoutils) CreateMongoUserObject(user UserStruct) error {

	_, err := m.userCollection.InsertOne(context.TODO(), user)
	return err

}

func (m *mongoutils) FetchMongoUserObject(userid string, exist bool, name string, email string) (UserStruct, error) {

	filter := bson.D{{"userid", userid}}
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.FindOne().SetSort(sort) //.SetProjection(projection)
	cursor := m.userCollection.FindOne(context.TODO(), filter, opts)

	var notifications UserStruct

	err := cursor.Decode(&notifications)
	if err != nil {
		return UserStruct{}, err
	}

	return notifications, nil
}

func (m *mongoutils) UpdateMongoUserConfig(userid string, data UserStruct) error {
	filter := bson.D{{"userid", userid}}
	update := bson.D{{"$set", data}}
	opts := options.Update().SetUpsert(true)

	_, err := m.userCollection.UpdateOne(context.TODO(), filter, update, opts)
	return err
}
