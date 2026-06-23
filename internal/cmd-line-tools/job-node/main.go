package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/pixlise/core/v4/api/job/jobnode"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/mongoDBConnection"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/utils"
)

func main() {
	fmt.Printf("Job Node version \"%v\"...\n", services.ApiVersion)

	instanceIdObtained, isEC2, err := utils.GetInstanceId()
	if err != nil {
		fmt.Printf("Assuming not running in EC2 due to failure retrieve EC2 instance id: %v\n", err)
	} // else still OK to continue, GetInstanceId should've generated a random string

	defer shutdown(instanceIdObtained, isEC2)

	// Read args
	var bucket, jobContainer, instanceId, mongoSecret, envName, jobs string
	var maxRunTimeSec int64

	flag.StringVar(&bucket, "bucket", "", "Bucket to read job data from")
	flag.StringVar(&jobContainer, "jobContainer", "", "The docker container to run jobs with")
	flag.StringVar(&instanceId, "instanceId", instanceIdObtained, "Instance ID (defaults to EC2 instance id or random string) - a unique number that identifies this node")
	flag.StringVar(&mongoSecret, "mongoSecret", "", "Name of mongo login secret")
	flag.StringVar(&envName, "envName", "", "Name of PIXLISE environment, eg dev, prod. Forms the DB name we connect to")
	flag.StringVar(&jobs, "jobs", "", "List of job IDs for this job node to run")
	flag.Int64Var(&maxRunTimeSec, "maxRunTimeSec", 60*15, "Max number seconds this node can exist")

	flag.Parse()

	// Some of this stuff can't be left empty
	if len(bucket) <= 0 {
		log.Fatalln("bucket can not be empty")
	}
	if len(envName) <= 0 {
		log.Fatalln("envName can not be empty")
	}

	jobIds := []string{}
	if len(jobs) <= 0 {
		log.Fatalln("jobs can not be empty")
	} else {
		// Ensure we can parse it into a list
		jobIds = strings.Split(jobs, ",")
		// If there's an empty one, remove it
		if len(jobIds) > 0 && len(jobIds[len(jobIds)-1]) <= 0 {
			jobIds = jobIds[0 : len(jobIds)-1]
		}

		// At this point there should be one or more jobs...
		if len(jobIds) <= 0 {
			log.Fatalln("Invalid/empty job id list specified")
		}
		for _, id := range jobIds {
			if len(id) <= 0 {
				log.Fatalf("Job id list specified had empty ids")
			}
		}
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
	l.SetLogLevel(logger.LogDebug)

	ts := timestamper.UnixTimeNowStamper{}
	fs := fileaccess.MakeS3Access(s3svc)

	mongoClient, _, err := mongoDBConnection.ConnectToMongo(sess, mongoSecret, &l, false)
	if err != nil {
		log.Fatalf("Failed to connect to mongo DB: %v", err)
	}

	dbName := mongoDBConnection.GetDatabaseName("pixlise", envName)
	db := mongoClient.Database(dbName)

	l.Infof("Running node until all jobs complete or up to %v seconds...", maxRunTimeSec)

	// Create job node
	jobNode := jobnode.CreateJobNode("job-"+envName, jobContainer, bucket, instanceId, fs, db, &l, &ts)

	// Check if there are any jobs waiting to be picked up
	jobNode.StartJobs(jobIds)

	// At this point we wait until the active job count drops to
	// zero or our max job time in seconds is breached, and we quit
	endReason, graceful := waitForJobCompletion(maxRunTimeSec, jobNode, &ts, &l)

	if graceful {
		l.Infof("Exiting due to %v", endReason)
		os.Exit(0)
	}

	log.Fatalf("Forced exit due to: %v", endReason)
}

func shutdown(instanceId string, isEC2 bool) {
	if !isEC2 {
		return // we can't shut down
	}

	// We got this instance ID from EC2, so we can shut down this machine at this point
	fmt.Printf("Instance %v should be shut down now...", instanceId)
}

func waitForJobCompletion(maxRunTimeSec int64, jobNode *jobnode.JobNode, ts timestamper.ITimeStamper, l logger.ILogger) (string, bool) {
	startTime := ts.GetTimeNowSec()
	errors := 0

	for range time.Tick(time.Second * 5) {
		now := ts.GetTimeNowSec()

		if now-startTime > maxRunTimeSec {
			return "max run time seconds exceeding limit", false
		}

		// If we have no jobs left, check how much time has elapsed since the last job finished
		activeJobs, err := jobNode.GetActiveJobCount()
		if err != nil {
			l.Errorf("Error querying active job count: %v", err)
			errors = errors + 1

			if errors > 3 {
				return "failing to query active job count several times", false
			}
			continue
		}

		if activeJobs == 0 {
			return "all jobs completed", true
		}

		l.Infof("Waiting for %v active jobs...", activeJobs)
	}

	return "unknown reason", false
}
