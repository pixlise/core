package main

import (
	"context"
	"fmt"
	"time"

	"github.com/pixlise/core/v3/api/dbCollections"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type SrcUINotificationItem struct {
	Topic     string    `json:"topic"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	UserID    string    `json:"userid"`
}

type SrcMethod struct {
	UI    bool `json:"ui"`
	Sms   bool `json:"sms"`
	Email bool `json:"email"`
}

type SrcNotificationConfig struct {
	SrcMethod `json:"method"`
}

type SrcTopics struct {
	Name   string                `json:"name"`
	Config SrcNotificationConfig `json:"config"`
}

type SrcNotifications struct {
	Topics          []SrcTopics             `json:"topics"`
	Hints           []string                `json:"hints"`
	UINotifications []SrcUINotificationItem `json:"uinotifications"`
}

type SrcUserDetails struct {
	Name           string `json:"name"`
	Email          string `json:"email"`
	Cell           string `json:"cell"`
	DataCollection string `json:"data_collection"`
}

type SrcUserStruct struct {
	ObjectId      string
	Userid        string           `json:"userid"`
	Notifications SrcNotifications `json:"notifications"`
	Config        SrcUserDetails   `json:"userconfig"`
}

func migrateUsersDB(src *mongo.Database, dest *mongo.Database) error {
	err := migrateUsersDBUsers(src, dest)
	if err != nil {
		return err
	}
	return migrateUsersDBNotifications(src, dest)
}

func migrateUsersDBUsers(src *mongo.Database, dest *mongo.Database) error {
	destColl := dest.Collection(dbCollections.UsersName)
	err := destColl.Drop(context.TODO())
	if err != nil {
		return err
	}

	filter := bson.D{}
	opts := options.Find()
	cursor, err := src.Collection("users").Find(context.TODO(), filter, opts)
	if err != nil {
		return err
	}

	srcUsers := []SrcUserStruct{}
	err = cursor.All(context.TODO(), &srcUsers)
	if err != nil {
		return err
	}

	destUsers := []interface{}{}
	readIds := map[string]bool{}
	for _, usr := range srcUsers {
		// Found duplicates...
		if readIds[usr.Userid] {
			fmt.Printf("Duplicate: %v - %v\n", usr.Userid, usr.Config.Name)
			continue
		}
		readIds[usr.Userid] = true

		// We add back the auth0| to the start of user id, which we stripped in past. This should allow us flexibility
		// in future to introduce auth0 users via google or other single sign-on services
		saveUserId := fixUserId(usr.Userid)
		destUser := protos.UserDBItem{
			Id:                    saveUserId,
			DataCollectionVersion: usr.Config.DataCollection,
			Info: &protos.UserInfo{
				Id:    saveUserId,
				Name:  usr.Config.Name,
				Email: usr.Config.Email,
			},
			Hints: &protos.UserHints{
				DismissedHints: usr.Notifications.Hints,
			},
			NotificationSettings: &protos.UserNotificationSettings{
				TopicSettings: map[string]protos.NotificationMethod{},
			},
		}

		if len(usr.Notifications.Topics) > 0 {
			topics := map[string]protos.NotificationMethod{}
			for _, topic := range usr.Notifications.Topics {
				m := protos.NotificationMethod_NOTIF_NONE
				if topic.Config.SrcMethod.Email && topic.Config.SrcMethod.UI {
					m = protos.NotificationMethod_NOTIF_BOTH
				} else if topic.Config.SrcMethod.Email {
					m = protos.NotificationMethod_NOTIF_EMAIL
				} else if topic.Config.SrcMethod.UI {
					m = protos.NotificationMethod_NOTIF_UI
				}
				topics[topic.Name] = m
			}

			destUser.NotificationSettings.TopicSettings = topics
		}

		destUsers = append(destUsers, destUser)
	}

	result, err := destColl.InsertMany(context.TODO(), destUsers)
	if err != nil {
		return err
	}

	fmt.Printf("Users inserted: %v\n", len(result.InsertedIDs))

	return err
}

func migrateUsersDBNotifications(src *mongo.Database, dest *mongo.Database) error {
	// We won't bring these across, the only ones stuck in prod are for a user who hasn't logged in since July 2022
	// because normally these should clear quickly
	return nil
}
