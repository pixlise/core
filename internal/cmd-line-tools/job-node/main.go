package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/pixlise/core/v4/api/job/jobnode"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/internal/cmd-line-tools/job-node/idleTime"
)

func main() {
	fmt.Printf("Job Node version %v...\n", services.ApiVersion)

	// Read args
	var bucket, jobContainer, instanceId, mongoSecret, envName string
	var maxJobs, maxIdleSec, maxRunTimeSec int64

	flag.StringVar(&bucket, "bucket", "", "Bucket to read job data from")
	flag.StringVar(&jobContainer, "jobContainer", "", "The docker container to run jobs with")
	flag.StringVar(&instanceId, "instanceId", "", "Instance ID (eg of the EC2 instance) - a unique number that identifies this node")
	flag.StringVar(&mongoSecret, "mongoSecret", "", "Name of mongo login secret")
	flag.StringVar(&envName, "envName", "", "Name of PIXLISE environment, eg dev, prod. Forms the DB name we connect to")
	flag.Int64Var(&maxJobs, "maxJobs", -1, "Max number of jobs to run simultaneously - set this to how many CPUs or threads this machine can run")
	flag.Int64Var(&maxIdleSec, "maxIdleSec", 60, "Max number seconds to wait for new jobs before shutdown")
	flag.Int64Var(&maxRunTimeSec, "maxRunTimeSec", 60*15, "Max number seconds this node can exist")

	flag.Parse()

	// Some of this stuff can't be left empty
	if len(bucket) <= 0 {
		log.Fatalln("bucket can not be empty")
	}
	if len(instanceId) <= 0 {
		log.Fatalln("instanceId can not be empty")
	}
	if len(envName) <= 0 {
		log.Fatalln("envName can not be empty")
	}
	if maxJobs < 1 {
		log.Fatalln("maxJobs must be > 0")
	}
	if maxIdleSec < 0 {
		log.Fatalln("maxIdleSec must be > 0")
	}

	if len(jobContainer) <= 0 {
		fmt.Println("Job container is empty - will run jobs locally in process")
	}
	if len(mongoSecret) <= 0 {
		fmt.Println("Mongo secret is empty - will attempt to connect to local mongo")
	}

	// Set up services
	sess, err := awsutil.GetSession()
	if err != nil {
		log.Fatalf("Failed to get AWS session")
	}
	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalf("Failed to get S3")
	}

	l := logger.StdOutLogger{}
	ts := timestamper.UnixTimeNowStamper{}
	fs := fileaccess.MakeS3Access(s3svc)

	mongoClient, _, err := mongoDBConnection.ConnectToMongo(sess, mongoSecret, &l, false)
	if err != nil {
		log.Fatalf("Failed to connect to mongo DB: %v", err)
	}

	dbName := mongoDBConnection.GetDatabaseName("pixlise", envName)
	db := mongoClient.Database(dbName)

	l.Infof("Running up to %v nodes. Node will run for up to %v seconds or %v idle seconds...", maxJobs, maxRunTimeSec, maxIdleSec)

	// Create job node
	jobNode := jobnode.CreateJobNode("job-"+envName, jobContainer, bucket, uint(maxJobs), uint(maxIdleSec), instanceId, fs, db, &l, &ts)

	// Check if there are any jobs waiting to be picked up
	jobNode.CheckStartupJobs()

	// At this point we fork into 2 tasks. One monitors how much time we've been idle since the last job completed
	// while the other listens for further jobs arriving. One of these will cause us to quit eventually!
	go monitorIdleTime(int64(maxIdleSec), int64(maxRunTimeSec), jobNode, &ts, &l)

	// Set it to listen for jobs. This keeps running until the host machine is shut down (therefore this process is killed)
	for jobNode.ListenToJobQueue() {
		// Our listener ended but we can re-listen, so wait a bit and try again
		time.Sleep(time.Second)
	}

	l.Infof("Exiting due job listener completing")
}

func monitorIdleTime(maxIdleSec, maxRunTimeSec int64, jobNode *jobnode.JobNode, ts timestamper.ITimeStamper, l logger.ILogger) {
	icheck := idleTime.MakeIdleTimeChecker(maxIdleSec, ts)
	consecutiveJobCheckFails := 0
	startTime := ts.GetTimeNowSec()

	for range time.Tick(time.Second * 5) {
		now := ts.GetTimeNowSec()

		if now-startTime > maxRunTimeSec {
			l.Infof("Exiting due to max run time seconds exceeding limit")
			os.Exit(0)
		}

		// If we have no jobs left, check how much time has elapsed since the last job finished
		activeJobs, err := jobNode.GetActiveJobCount()
		if err == nil {
			consecutiveJobCheckFails = 0
			if icheck.HasIdleTimeExpired(activeJobs) {
				l.Infof("Exiting due to idle time seconds exceeding limit")
				os.Exit(0)
			}
		} else {
			consecutiveJobCheckFails = consecutiveJobCheckFails + 1
			l.Errorf("Failed to check active job count %v times. Error: %v", consecutiveJobCheckFails, err)
		}

		if consecutiveJobCheckFails > 10 {
			log.Fatalf("Exiting due to %v failures to check active job count", consecutiveJobCheckFails)
		}
	}
}
