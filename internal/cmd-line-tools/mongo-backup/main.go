package main

import (
	"flag"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/mongobackup"
	"github.com/pixlise/core/v4/core/timestamper"
)

// Run this to do periodic mongo backups. Dumps and zips the DB and uploads it to an S3 location.
// Provide command line arguments to control operation

func main() {
	var startupSecStr string
	var intervalSecStr string
	var mongoHost string
	var mongoUsername string
	var mongoPassword string
	var backupBucket string
	var backupS3Path string
	var dbName string

	flag.StringVar(&startupSecStr, "startup_sec", "60", "First backup will run this many seconds after starting this process")
	flag.StringVar(&intervalSecStr, "interval_sec", "21600", "Backups will be run on this interval of seconds")
	flag.StringVar(&mongoHost, "db_host", "localhost:27017", "Mongo DB host. Can specify multiple and require connection to a secondary")
	flag.StringVar(&mongoUsername, "db_user", "", "Mongo DB Username")
	flag.StringVar(&mongoPassword, "db_password", "", "Mongo DB Password")
	flag.StringVar(&backupBucket, "backup_bucket", "", "S3 bucket to write backup to")
	flag.StringVar(&backupS3Path, "backup_path", "", "S3 bucket path to write backup to")
	flag.StringVar(&dbName, "backup_db", "", "Name of database to back up")

	flag.Parse()

	// Validate everything
	startupSec, err := strconv.Atoi(startupSecStr)
	if err != nil || startupSec <= 0 {
		log.Fatalln("startup_sec must be a positive number")
		return
	}

	intervalSec, err := strconv.Atoi(intervalSecStr)
	if err != nil || intervalSec < 0 {
		log.Fatalln("interval_sec must be a positive number")
		return
	}

	if intervalSec == 0 {
		fmt.Printf("NOTE: interval_sec is set to 0, so backup will only run once and process will then exit")
	}

	if len(mongoHost) <= 0 {
		log.Fatalln("db_host must not be empty")
		return
	}

	if len(backupBucket) <= 0 {
		log.Fatalln("backup_bucket must be set to the name of the s3 bucket to write to")
		return
	}

	if len(backupS3Path) <= 0 {
		log.Fatalln("backup_path must be set to the path within the s3 bucket to write to")
		return
	}

	if len(dbName) <= 0 {
		log.Fatalln("backup_db must be set to the name of the db")
		return
	}

	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("Failed to create AWS S3 service. Error: %v", err)
	}

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("Failed to create AWS S3 service. Error: %v", err)
	}

	remoteFS := fileaccess.MakeS3Access(s3svc)

	svcs := &services.APIServices{
		Log: &logger.StdOutLoggerForTest{},
		MongoDetails: mongoDBConnection.MongoConnectionDetails{
			Host:     mongoHost,
			User:     mongoUsername,
			Password: mongoPassword,
		},
		TimeStamper: &timestamper.UnixTimeNowStamper{},
		FS:          remoteFS,
	}

	fmt.Printf("Waiting %v seconds to start...\n", startupSec)
	time.Sleep(time.Duration(startupSec) * time.Second)

	errCount := 0
	for {
		fmt.Printf("Running backup...\n")
		err = mongobackup.BackupDB(dbName, backupBucket, backupS3Path, true, svcs)
		if err != nil {
			log.Printf("Error: %v\n", err)
			errCount++
		}

		if intervalSec == 0 {
			fmt.Println("Backup complete, exiting due to interval configuration of 0 meaning don't re-run")
			break
		}

		fmt.Printf("Waiting %v seconds for next backup interval...\n", intervalSec)
		time.Sleep(time.Duration(intervalSec) * time.Second)
	}
}
