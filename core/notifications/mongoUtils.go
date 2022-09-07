package notifications

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/url"
	"strings"
	"time"

	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/pixlise/core/core/logger"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/event"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoUtils struct {
	client                 *mongo.Client
	userDatabase           *mongo.Database
	userCollection         *mongo.Collection
	notificationCollection *mongo.Collection
	SecretsCache           *secretcache.Cache
	ConnectionSecret       string
	MongoUsername          string
	MongoPassword          string
	MongoEndpoint          string
	Log                    logger.ILogger
}

func getCustomTLSConfig(caFile string) (*tls.Config, error) {
	tlsConfig := new(tls.Config)
	certs, err := ioutil.ReadFile(caFile)

	if err != nil {
		return tlsConfig, err
	}

	tlsConfig.RootCAs = x509.NewCertPool()
	ok := tlsConfig.RootCAs.AppendCertsFromPEM(certs)

	if !ok {
		return tlsConfig, errors.New("Failed parsing pem file")
	}

	return tlsConfig, nil
}

func (m *MongoUtils) Connect() error {
	if m.MongoEndpoint == "" {
		m.Log.Infof("Mongo DB endpoint not configured, all calls will be ignored...")
		return nil
	}

	cmdMonitor := &event.CommandMonitor{
		Started: func(_ context.Context, evt *event.CommandStartedEvent) {
			m.Log.Infof("%v", evt.Command)
		},
	}
	//ctx := context.Background()
	var err error
	m.Log.Infof("Connecting to mongo db: %v", m.MongoEndpoint)
	if m.ConnectionSecret != "" && m.SecretsCache != nil {
		connectionStringTemplate := "mongodb://%s:%s@%s/userdatabase?tls=true&replicaSet=rs0&readpreference=%s"
		pw, err := m.SecretsCache.GetSecretString(m.ConnectionSecret)
		var result map[string]interface{}
		json.Unmarshal([]byte(pw), &result)
		if err != nil {
			m.Log.Errorf("failed to fetch secret: %v", m.ConnectionSecret)
		}
		str := fmt.Sprintf("%v", result["password"])
		connectionURI := fmt.Sprintf(connectionStringTemplate, "pixlise", url.QueryEscape(str), m.MongoEndpoint, "secondaryPreferred")
		tlsConfig, err := getCustomTLSConfig("./rds-combined-ca-bundle.pem")
		if err != nil {
			m.Log.Errorf("Failed getting TLS configuration: %v", err)
		}
		m.client, err = mongo.NewClient(options.Client().ApplyURI(connectionURI).SetMonitor(cmdMonitor).SetTLSConfig(tlsConfig).SetRetryWrites(false))
		if err != nil {
			m.Log.Errorf("Failed connection: %v", err)
		}
		m.Log.Infof("Connection Successful")
	} else {
		m.client, err = mongo.NewClient(options.Client().ApplyURI("mongodb://localhost").SetMonitor(cmdMonitor))
	}

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

	m.Log.Infof("Switching Databases")
	m.userDatabase = m.client.Database("userdatabase")
	m.userCollection = m.userDatabase.Collection("users")
	m.notificationCollection = m.userDatabase.Collection("notifications")
	m.Log.Infof("Mongo Setup Complete")
	return nil
}

func (m *MongoUtils) GetAllMongoUsers(log logger.ILogger) ([]UserStruct, error) {
	m.Log.Infof("Fetching All Subscribers Mongo Object")

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
	m.Log.Infof("Fetched All Subscribers Mongo Object")
	return notifications, nil
}

func (m *MongoUtils) GetMongoSubscribersByTopicID(override []string, searchtopic string, logger logger.ILogger) ([]UserStruct, error) {
	m.Log.Infof("Fetching Subscriber Mongo Object for topic: %v", searchtopic)

	var filter bson.M
	if override != nil && len(override) > 0 {
		var v []string
		for _, f := range override {
			s := strings.TrimPrefix(f, "auth0|")
			v = append(v, s)
		}
		filter = bson.M{
			"$and": []bson.D{
				{
					{"userid", bson.D{{"$in", v}}},
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
	m.Log.Infof("Fetched Subscriber Mongo Object for topic: %v", searchtopic)

	return notifications, nil
}

func (m *MongoUtils) GetMongoSubscribersByEmailTopicID(override []string, searchtopic string, logger logger.ILogger) ([]UserStruct, error) {
	m.Log.Infof("Fetching Subscriber Mongo Object for topic: %v", searchtopic)

	var filter bson.M
	if override != nil && len(override) > 0 {
		var v []string
		for _, f := range override {
			s := strings.TrimPrefix(f, "auth0|")
			v = append(v, s)
		}
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
	m.Log.Infof("Fetched Subscriber Mongo Object for topic: %v", searchtopic)

	return notifications, nil
}

func (m *MongoUtils) GetMongoSubscribersByTopic(searchtopic string, logger logger.ILogger) ([]UserStruct, error) {
	m.Log.Infof("Fetching Subscribers Mongo Object for topic: %v", searchtopic)

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
	m.Log.Infof("Fetched Subscribers Mongo Object for topic: %v", searchtopic)

	return notifications, nil
}

func (m *MongoUtils) InsertUINotification(newNotification UINotificationObj) error {
	if m.MongoEndpoint == "" {
		return errors.New("Mongo not connected")
	}

	m.Log.Infof("Inserting UI Notification Mongo Object for user: %v", newNotification.UserID)

	_, err := m.notificationCollection.InsertOne(context.TODO(), newNotification)
	if err != nil {
		return err
	}
	m.Log.Infof("Inserted UI Notification Mongo Object for user: %v", newNotification.UserID)

	return nil
}

func (m *MongoUtils) GetUINotifications(user string) ([]UINotificationObj, error) {
	m.Log.Infof("Fetching Mongo Notifications for user: %v", user)

	filter := bson.D{{"userid", user}}
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}

	opts := options.Find().SetSort(sort) //.SetProjection(projection)
	cursor, err := m.notificationCollection.Find(context.TODO(), filter, opts)
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
	m.Log.Infof("Fetched Mongo Notifications for user: %v", user)

	return notifications, nil
}

func (m *MongoUtils) DeleteUINotifications(user string) error {
	filter := bson.D{{"userid", user}}

	_, err := m.notificationCollection.DeleteMany(context.TODO(), filter)
	return err
}

func (m *MongoUtils) CreateMongoUserObject(user UserStruct) error {
	if m.MongoEndpoint == "" {
		return errors.New("Mongo not connected")
	}

	m.Log.Infof("Creating Mongo Object for user: %v", user.Userid)

	_, err := m.userCollection.InsertOne(context.TODO(), user)
	m.Log.Infof("Created Mongo Object for user: %v", user.Userid)

	return err
}

func (m *MongoUtils) FetchMongoUserObject(userid string, exist bool, name string, email string) (UserStruct, error) {
	if m.MongoEndpoint == "" {
		return UserStruct{}, errors.New("Mongo not connected")
	}

	m.Log.Infof("Fetching Mongo Object for user: %v", userid)
	filter := bson.D{{"userid", userid}}
	sort := bson.D{{"timestamp", -1}}
	//projection := bson.D{{"type", 1}, {"rating", 1}, {"_id", 0}}
	opts := options.FindOne().SetSort(sort) //.SetProjection(projection)
	cursor := m.userCollection.FindOne(context.TODO(), filter, opts)

	var notifications UserStruct
	m.Log.Infof("Decoding Mongo Object for user: %v", userid)
	err := cursor.Decode(&notifications)
	if err != nil {
		return UserStruct{}, err
	}
	m.Log.Infof("Fetched Mongo Object for user: %v", userid)

	return notifications, nil
}

func (m *MongoUtils) UpdateMongoUserConfig(userid string, data UserStruct) error {
	if m.MongoEndpoint == "" {
		return errors.New("Mongo not connected")
	}

	m.Log.Infof("Updating Mongo Object for user: %v", userid)

	filter := bson.D{{"userid", userid}}
	update := bson.D{{"$set", data}}
	opts := options.Update().SetUpsert(true)

	_, err := m.userCollection.UpdateOne(context.TODO(), filter, update, opts)
	m.Log.Infof("Updated Mongo Object for user: %v", userid)

	return err
}
