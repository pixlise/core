package wsHelpers

import (
	"context"
	"fmt"

	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func Example_wsHelpers_ConnectTokens() {
	db := wstestlib.GetDB()
	ctx := context.TODO()
	coll := db.Collection(dbCollections.ConnectTempTokensName)

	fmt.Printf("drop: %v\n", coll.Drop(ctx))

	token1 := ConnectToken{
		Id:            "abc123",
		ExpiryUnixSec: 1234567890,
		User: jwtparser.JWTUserInfo{
			Name:   "TheUser's Name",
			UserID: "u123",
			Email:  "user@mail.com",
		},
	}

	_, err := coll.InsertOne(ctx, token1, options.InsertOne())
	fmt.Printf("insert1: %v\n", err)

	// Insert again
	token1.Id = "abc999"
	_, err = coll.InsertOne(ctx, token1, options.InsertOne())
	fmt.Printf("insert2: %v\n", err)

	_, err = CheckConnectToken("abcd", &services.APIServices{
		Log:     &logger.StdOutLoggerForTest{},
		MongoDB: db,
		TimeStamper: &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1234567890},
		},
	})

	fmt.Printf("notfound: %v\n", err)

	usr, err := CheckConnectToken("abc123", &services.APIServices{
		Log:     &logger.StdOutLoggerForTest{},
		MongoDB: db,
		TimeStamper: &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1234567887},
		},
	})

	fmt.Printf("token ok %v, user: %v\n", err, usr.Name)

	_, err = CheckConnectToken("abc999", &services.APIServices{
		Log:     &logger.StdOutLoggerForTest{},
		MongoDB: db,
		TimeStamper: &timestamper.MockTimeNowStamper{
			QueuedTimeStamps: []int64{1234567892},
		},
	})

	fmt.Printf("token expired, err: %v\n", err)

	// Output:
	// drop: <nil>
	// insert1: <nil>
	// insert2: <nil>
	// notfound: Provided token is unknown
	// token ok <nil>, user: TheUser's Name
	// token expired, err: Expired token
}
