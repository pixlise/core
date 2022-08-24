package main

import (
	"github.com/pixlise/core/api/services"
	"github.com/pixlise/core/core/notifications"
	"log"
)

func connectToMongo() {
	mongo := notifications.MongoUtils{
		SecretsCache:     seccache,
		ConnectionSecret: cfg.MongoSecret,
		MongoUsername:    cfg.MongoUsername,
		MongoEndpoint:    cfg.MongoEndpoint,
		Log:              svcs.Log,
	}
	err = mongo.Connect()
}
func readJsonFromS3(svcs *services.APIServices, s3Path string, ) notifications.UserStruct {
	resp := notifications.UserStruct{}
	err := svcs.FS.ReadJSON(svcs.Config.ConfigBucket, s3Path, &resp, false)
	if err != nil {
		log.Fatalf("Error looking up file: %v", err)
	}

	return resp

}

func parseJsonToMongoObj() {

}

func writeToMongo() {

}

func main() {

	readJsonFromS3()
	parseJsonToMongoObj()
	writeToMongo()
}
