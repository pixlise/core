package wsHelpers

import (
	"context"
	"errors"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/utils"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type ConnectToken struct {
	Id            string `bson:"_id"`
	ExpiryUnixSec int64
	User          jwtparser.JWTUserInfo
	Permissions   []string
}

func CreateConnectToken(svcs *services.APIServices, user jwtparser.JWTUserInfo) string {
	// Generate a new token
	perms := []string{}
	for k := range user.Permissions {
		perms = append(perms, k)
	}

	token := ConnectToken{
		Id:            utils.RandStringBytesMaskImpr(32), // The actual token
		ExpiryUnixSec: svcs.TimeStamper.GetTimeNowSec() + 10,
		User:          user,
		Permissions:   perms,
	}

	// Save to DB
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.ConnectTempTokensName)

	_, err := coll.InsertOne(ctx, token, options.InsertOne())
	if err != nil {
		svcs.Log.Errorf("Failed to save new connect token: %v. Error: %v", token.Id, err)
		return ""
	}

	return token.Id
}

func CheckConnectToken(token string, svcs *services.APIServices) (jwtparser.JWTUserInfo, error) {
	// Check to see if token exists in DB
	ctx := context.TODO()
	coll := svcs.MongoDB.Collection(dbCollections.ConnectTempTokensName)

	result := coll.FindOne(ctx, bson.M{"_id": token}, options.FindOne())
	if result.Err() != nil {
		// If not found, return the right error
		if result.Err() == mongo.ErrNoDocuments {
			return jwtparser.JWTUserInfo{}, errors.New("Provided token is unknown")
		}

		// Other error
		svcs.Log.Errorf("Failed to read connect token: %v. Error: %v", token, result.Err())
		return jwtparser.JWTUserInfo{}, result.Err()
	}

	// Read the item
	readToken := ConnectToken{}
	if err := result.Decode(&readToken); err != nil {
		svcs.Log.Errorf("Failed to decode connect token: %v. Error: %v", token, err)
		return jwtparser.JWTUserInfo{}, result.Err()
	}

	// Check expiry
	nowUnixSec := svcs.TimeStamper.GetTimeNowSec()

	if readToken.ExpiryUnixSec < nowUnixSec {
		svcs.Log.Errorf("WS connect failed for EXPIRED token: %v. User: %v (%v)\n", token, readToken.User.UserID, readToken.User.Name)
		return jwtparser.JWTUserInfo{}, errors.New("Expired token")
	}

	// Delete expired tokens from DB
	filter := bson.M{
		"$or": []interface{}{
			bson.M{"_id": readToken.Id},
			bson.M{"expiryunixsec": bson.M{"$lt": nowUnixSec}},
		},
	}

	delResult, err := coll.DeleteMany(ctx, filter, options.Delete())
	if err != nil {
		svcs.Log.Errorf("Failed to delete expired/used connect tokens: %v\n", err)
	} else {
		if delResult.DeletedCount > 0 {
			svcs.Log.Infof("Deleted %v expired/used connect tokens", delResult.DeletedCount)
		}
	}

	// Set the permissions field!
	readToken.User.Permissions = map[string]bool{}
	for _, perm := range readToken.Permissions {
		readToken.User.Permissions[perm] = true
	}

	return readToken.User, nil
}
