package jobmanager

import (
	"fmt"
)

func Example_jobmanager_getJobsPerNode() {
	jobIds := []string{"id1", "id2", "id3", "id4", "id5", "id6", "id7"}

	for c := 0; c < 20; c++ {
		fmt.Printf("%v/node: %v\n", c, getJobsPerNode(jobIds, uint(c)))
	}

	jobIds = []string{"id1", "id2", "id3", "id4", "id5", "id6", "id7", "id8", "id9", "id10", "id11", "id12", "id13"}
	fmt.Printf("13job %v/node: %v\n", 4, getJobsPerNode(jobIds, uint(4)))

	// Output:
	// 0/node: [[id1] [id2] [id3] [id4] [id5] [id6] [id7]]
	// 1/node: [[id1] [id2] [id3] [id4] [id5] [id6] [id7]]
	// 2/node: [[id1 id2] [id3 id4] [id5 id6] [id7]]
	// 3/node: [[id1 id2 id3] [id4 id5 id6] [id7]]
	// 4/node: [[id1 id2 id3 id4] [id5 id6 id7]]
	// 5/node: [[id1 id2 id3 id4 id5] [id6 id7]]
	// 6/node: [[id1 id2 id3 id4 id5 id6] [id7]]
	// 7/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 8/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 9/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 10/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 11/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 12/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 13/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 14/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 15/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 16/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 17/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 18/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 19/node: [[id1 id2 id3 id4 id5 id6 id7]]
	// 13job 4/node: [[id1 id2 id3 id4] [id5 id6 id7 id8] [id9 id10 id11 id12] [id13]]
}

/*
func Example_jobmanager_ec2Start() {
	//region := "us-east-1"
	//region := "ap-southeast-2"
	//sess, err := awsutil.GetSessionWithRegion(region)
	sess, err := session.NewSessionWithOptions(session.Options{Profile: "pixlisedeploy"}) //awsutil.GetSession()
	if err != nil {
		log.Fatalln(err)
	}

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalln(err)
	}

	logLevel := logger.LogDebug
	svcs := servicesMock.MakeMockSvcs(&awsutil.MockS3Client{}, nil, &logLevel)
	svcs.S3 = s3svc

	svcs.EC2 = ec2.New(sess)
	svcs.SecretsManager = secretsmanager.New(sess)
	/*
		// Auscope account
		svcs.Config.JobAMI = "ami-01bd06a7d961327f0"
		svcs.Config.JobInstanceType = "t3.2xlarge"
		svcs.Config.JobKeyName = "AuscopeMongo"
		svcs.Config.JobSecurityGroup = "sg-0bd86d41ab0ee8df2"
	* /
	// PIXLISE account
	svcs.Config.JobAMI = "ami-00e801948462f718a" // arm64 -> "ami-0b11e0ed3f8697f97"
	svcs.Config.JobInstanceType = "t3.2xlarge"
	svcs.Config.JobKeyName = "PixliseEBMongo"
	svcs.Config.JobSecurityGroup = "sg-03617d16414a3431d"
	svcs.Config.JobMaxNodeRunTimeSec = 60 * 15
	svcs.Config.JobAWSSecret = "pixlise/job/credentials"
	svcs.Config.JobNodeS3Path = "s3://pixlise-prod-config/JobNode/pixlise-job-node"
	svcs.Config.CoresPerNode = 4
	svcs.Config.MongoSecret = "pixlise/mongo/login"
	svcs.Config.PiquantJobsBucket = "pixlise-prod-jobs"
	svcs.Config.JobRunnerDockerImage = "ghcr.io/pixlise/job-runner:latest"

	jm, err := Create(&svcs, 0, false, false, false)
	if err != nil {
		log.Fatalln(err)
	}

	err = jm.startEC2JobNode([]string{"id1", "id2"}, true)
	if err != nil {
		log.Fatalln(err)
	}

	time.Sleep(120 * time.Second)

	// Output:
	// <nil>
}

func Example_jobmanager_getRunningNodes() {

	//region := "us-east-1"
	//region := "ap-southeast-2"
	//sess, err := awsutil.GetSessionWithRegion(region)
	sess, err := session.NewSessionWithOptions(session.Options{Profile: "pixlisesvc"}) //awsutil.GetSession()
	if err != nil {
		log.Fatalln(err)
	}

	s3svc, err := awsutil.GetS3(sess)
	if err != nil {
		log.Fatalln(err)
	}

	logLevel := logger.LogDebug
	svcs := servicesMock.MakeMockSvcs(&awsutil.MockS3Client{}, nil, &logLevel)
	svcs.S3 = s3svc

	svcs.EC2 = ec2.New(sess)
	svcs.SecretsManager = secretsmanager.New(sess)
	/*
		// Auscope account
		svcs.Config.JobAMI = "ami-01bd06a7d961327f0"
		svcs.Config.JobInstanceType = "t3.2xlarge"
		svcs.Config.JobKeyName = "AuscopeMongo"
		svcs.Config.JobSecurityGroup = "sg-0bd86d41ab0ee8df2"
	* /
	// PIXLISE account
	svcs.Config.JobAMI = "ami-00e801948462f718a" // arm64 -> "ami-0b11e0ed3f8697f97"
	svcs.Config.JobInstanceType = "t3.2xlarge"
	svcs.Config.JobKeyName = "PixliseEBMongo"
	svcs.Config.JobSecurityGroup = "sg-03617d16414a3431d"
	svcs.Config.JobMaxNodeRunTimeSec = 60 * 15
	svcs.Config.JobAWSSecret = "pixlise/job/credentials"
	svcs.Config.JobNodeS3Path = "s3://pixlise-prod-config/JobNode/pixlise-job-node"
	svcs.Config.CoresPerNode = 4
	svcs.Config.MongoSecret = "pixlise/mongo/login"
	svcs.Config.PiquantJobsBucket = "pixlise-prod-jobs"
	svcs.Config.JobRunnerDockerImage = "ghcr.io/pixlise/job-runner:latest"
	svcs.Config.EnvironmentName = "prod"

	jm, err := Create(&svcs, 0, false, false, false)
	if err != nil {
		log.Fatalln(err)
	}

	instances, err := jm.getRunningNodes()
	fmt.Printf("%v|%v\n", err, strings.Join(instances, ","))

	// Output:
	// Something
}
*/
