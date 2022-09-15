package main

import (
	"encoding/json"
	"fmt"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/pixlise/core/v2/core/logger"
	"github.com/pixlise/core/v2/core/notifications"
	"io/ioutil"
	"os"
)

func connectToMongo() notifications.MongoUtils {
	var ourLogger logger.ILogger = &logger.StdOutLogger{}
	secretscache, err := secretcache.New()
	if err != nil {
		fmt.Printf("Broke Mongo Lads: %v", err)
		os.Exit(1)
	}
	mongo := notifications.MongoUtils{
		SecretsCache:     secretscache,
		ConnectionSecret: os.Getenv("MONGO_SECRET"),
		MongoUsername:    os.Getenv("MONGO_USER"),
		MongoEndpoint:    os.Getenv("MONGO_ENDPOINT"),
		Log:              ourLogger,
	}
	err = mongo.Connect()
	if err != nil {
		fmt.Printf("Broke Mongo Lads: %v", err)
		os.Exit(1)
	}
	return mongo
}

func parseJsonToMongoObj(j string) notifications.UserStruct {

	var s notifications.UserStruct

	err := json.Unmarshal([]byte(j), &s)
	if err != nil {
		fmt.Printf("Broke it lads: %v", err)
		os.Exit(1)
	}

	return s
}

func writeToMongo(mongo notifications.MongoUtils, user notifications.UserStruct) {
	err := mongo.CreateMongoUserObject(user)
	fmt.Printf("Failed to create object: %v", err)
}

func main() {
	items, _ := ioutil.ReadDir(".")
	for _, item := range items {
		if item.IsDir() {
			subitems, _ := ioutil.ReadDir(item.Name())
			for _, subitem := range subitems {
				if !subitem.IsDir() {
					// handle file there
					fmt.Println(item.Name() + "/" + subitem.Name())
					b, err := os.ReadFile(item.Name() + "/" + subitem.Name())
					if err != nil {
						fmt.Print(err)
					}

					fmt.Println(b) // print the content as 'bytes'

					j := string(b)
					user := parseJsonToMongoObj(j)
					mongo := connectToMongo()
					writeToMongo(mongo, user)
				}
			}
		} else {
			// handle file there
			fmt.Println(item.Name())
			fmt.Println(item.Name())
			b, err := os.ReadFile(item.Name())
			if err != nil {
				fmt.Print(err)
			}
			j := string(b)

			user := parseJsonToMongoObj(j)
			mongo := connectToMongo()
			writeToMongo(mongo, user)
		}
	}

}
