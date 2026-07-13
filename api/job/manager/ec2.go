package jobmanager

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/pixlise/core/v4/api/dbCollections"
	"github.com/pixlise/core/v4/api/job/jobnode"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

// Speed comparison: Quantifying Tanda tula sol 1777 on prod with all 1377 PMCs, elements Al,Si,Ca,Fe ran on 15 nodes, 93 spectra/node:
// started 11:51:47
// kubernetes nodes running at 11:51:53
// nodes success at 11:53:58
// job finished 11:54:03
// Job reported elapsed time 2min 16sec

// New: Takes about 1min 30sec to even start the nodes
//      Nodes starting up have to install docker, then each container downloads stuff

// t3.medium, 2cores/node makes 32 jobs, runs on 16 nodes
// t3.medium, 4cores/node makes 16 jobs, runs on 4 nodes
//??????????????? t3.medium, 8cores/node makes 16 jobs, runs on 4 nodes

// New jobs t3.medium, 8 cores/node: elapsed time 1821 sec = 30:21 (!!!!) (failed to combine at end quant-9830m8cpx6zcbo86 missing node 0 output)
// New jobs t3.medium, 4 cores/node: elapsed time 11:04 4 nodes, second run 10:52
// New jobs t3.medium, 2 cores/node: elapsed time 4:03 16 nodes, second run 4:09 (failed to combine at end quant-27eynb9dfhca6oe2), 3rd run worked 4:05, 4th run worked 4:12
// New jobs t3a.medium, 2 cores/node: elapsed time 5:20 16 nodes
// New jobs t3.large, 2 cores/node: elapsed time 5:24 sec (failed to combine at end quant-mv1pzuf3359wrtd4 missing node 10 output) 16 nodes
// New jobs t3.xlarge, 4 cores/node: elapsed time 9:39 (failed to combine at end quant-wtmqejxiulf2x1x1 missing node 15 output) 4 nodes <------- should be faster, each docker was using 50% CPU???
// New jobs t3.2xlarge, 4 cores/node: elapsed time 11:46 top showing 200%/container ?? (failed to combine at end quant-5ikkcvd8hes5yrzq missing node 8 output)
// New jobs t3.2xlarge, 8 cores/node: elapsed time 13:31 top showing 100%/container (failed to combine at end quant-y52kwmyey35n0cu2 missing node 1 output)

// After EstimateNodeCountChange:
// t3.medium 2/node 4 elements as 50 jobs/25 nodes: 221 sec
// t3.xlarge 4/node 4 elements as 50 jobs/13 nodes: 250 sec
// t3.2xlarge 8/node 4 elements as 50 jobs/7 nodes: 270 sec

// Called to start a job node
func (jm *JobManager) startEC2JobNode(jobIds []string, awsKey string, awsSecret string, awsRegion string) ([]*string, error) {
	if len(jobIds) <= 0 || len(jobIds) > int(jm.svcs.Config.Jobs.CoresPerNode) {
		return []*string{}, fmt.Errorf("Invalid job count when starting EC2 job nodes: %v", len(jobIds))
	}

	// Ensure no jobs have , in their ids because we'll be putting them in a string list separated by ,
	for _, id := range jobIds {
		if strings.Contains(id, ",") {
			return []*string{}, fmt.Errorf("Invalid job id specified, illegal , character detected: %v", id)
		}
	}

	jobIdListStr := strings.Join(jobIds, ",")

	if jm.svcs.Config.Jobs.MaxNodeRunTimeSec < 60 {
		return []*string{}, fmt.Errorf("Cannot start job node that runs for only %vsec", jm.svcs.Config.Jobs.MaxNodeRunTimeSec)
	}

	if len(jm.svcs.Config.Jobs.AWSSecret) <= 0 {
		return []*string{}, fmt.Errorf("JobNode AWS secret not set")
	}

	jobNodeInstanceName := fmt.Sprintf("job-node-%v", jm.svcs.Config.EnvironmentName)

	startupScript := fmt.Sprintf(`#!/bin/bash
set -e
shutdown -h +%v
echo "Starting PIXLISE job node, limited to %v sec runtime"

export AWS_ACCESS_KEY_ID="%v"
export AWS_SECRET_ACCESS_KEY="%v"
export AWS_REGION="%v"
export AWS_DEFAULT_REGION="%v"
export AWS_S3_US_EAST_1_REGIONAL_ENDPOINT="regional"

echo "Setting up Docker..."
# Update system packages
dnf update -y

# Install Docker
dnf install -y docker

# Start and enable Docker service
systemctl start docker
systemctl enable docker

echo "Downloading job node..."
mkdir /job-node
cd /job-node
aws s3 cp "%v" "."
chmod +x ./pixlise-job-node

echo "Downloading global-bumdle.pem..."
wget https://truststore.pki.rds.amazonaws.com/global/global-bundle.pem -O global-bundle.pem

echo "Running job node..."
./pixlise-job-node -bucket "%v" -jobContainer "%v" -mongoSecret "%v" -envName "%v" -maxRunTimeSec "%v" -jobs "%v"

echo "PIXLISE job node shutting down in 1 minute..."
shutdown -h +1
`,
		jm.svcs.Config.Jobs.MaxNodeRunTimeSec/60,
		jm.svcs.Config.Jobs.MaxNodeRunTimeSec,
		awsKey, awsSecret,
		awsRegion, awsRegion,
		jm.svcs.Config.Jobs.NodeS3Path,
		jm.svcs.Config.PiquantJobsBucket,
		jm.svcs.Config.Jobs.RunnerDockerImage,
		jm.svcs.Config.MongoSecret,
		jm.svcs.Config.EnvironmentName,
		jm.svcs.Config.Jobs.MaxNodeRunTimeSec-5,
		jobIdListStr,
	)

	input := &ec2.RunInstancesInput{
		// placement (AZ - not setting it here?!)
		ImageId:      aws.String(jm.svcs.Config.Jobs.AMI),
		InstanceType: aws.String(jm.svcs.Config.Jobs.InstanceType),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{Key: aws.String("Name"), Value: aws.String(jobNodeInstanceName)},
					{Key: aws.String("pixlise:instance-use"), Value: aws.String("job-node")},
					{Key: aws.String("pixlise:environment"), Value: aws.String(jm.svcs.Config.EnvironmentName)},
					{Key: aws.String("pixlise:starter-instance-id"), Value: aws.String(jm.svcs.InstanceId)},
					{Key: aws.String("pixlise:job-ids"), Value: aws.String(jobIdListStr)},
				},
			},
		},
		KeyName:          aws.String(jm.svcs.Config.Jobs.KeyName),
		SecurityGroupIds: []*string{aws.String(jm.svcs.Config.Jobs.SecurityGroup)},
		MaxCount:         aws.Int64(int64(1)),
		MinCount:         aws.Int64(int64(1)),
		UserData:         aws.String(base64.StdEncoding.EncodeToString([]byte(startupScript))),
	}
	if len(jm.svcs.Config.Jobs.SubnetId) > 0 {
		input.SetSubnetId(jm.svcs.Config.Jobs.SubnetId)
	}

	res, err := jm.svcs.EC2.RunInstances(input)
	if err != nil {
		return []*string{}, err
	}

	// List all instances started
	instances := []*string{}
	instanceStrs := []string{}
	for _, inst := range res.Instances {
		instances = append(instances, inst.InstanceId)
		instanceStrs = append(instanceStrs, *inst.InstanceId)
	}

	jm.svcs.Log.Infof("   Started %v instance(s) [%v]", len(instances), strings.Join(instanceStrs, ","))
	jm.startedNodeCount = jm.startedNodeCount + 1

	return instances, err
}

// Expects to find secret value JSON of the form:
// {"aws_key": "K", "aws_secret": "S", "aws_region": "R"}
func readSecretsManager(secretsManager *secretsmanager.SecretsManager, secretName string) (string, string, string, error) {
	seccache, err := secretcache.New(func(c *secretcache.Cache) { c.Client = secretsManager })
	if err != nil {
		return "", "", "", err
	}

	secretValue, err := seccache.GetSecretString(secretName)
	if err != nil {
		return "", "", "", err
	}

	// Secret cache seems to return these types... Unmarshall it
	type AWSCredentials struct {
		Key    string `json:"aws_key"`
		Secret string `json:"aws_secret"`
		Region string `json:"aws_region"`
	}

	info := &AWSCredentials{}
	err = json.Unmarshal([]byte(secretValue), &info)
	if err != nil {
		return "", "", "", fmt.Errorf("failed to parse secret: \"%v\". Error: %v", secretName, err)
	}

	return info.Key, info.Secret, info.Region, nil
}

func (jm *JobManager) getRunningNodes() ([]string, error) {
	// For testing/local mode, if we have already started that one thread, say there's just us as the node
	if jm.isLocalTestMode() {
		if jm.localJobNode == nil {
			return []string{}, nil
		}
		return []string{jm.svcs.InstanceId}, nil
	}

	// Only grab instances that are running or just started
	filters := []*ec2.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running"), aws.String("pending"), aws.String("initializing")},
		},
		{
			Name:   aws.String("tag:pixlise:environment"),
			Values: []*string{aws.String(jm.svcs.Config.EnvironmentName)},
		},
		{
			Name:   aws.String("tag:pixlise:instance-use"),
			Values: []*string{aws.String("job-node")},
		},
	}

	request := &ec2.DescribeInstancesInput{Filters: filters}
	result, err := jm.svcs.EC2.DescribeInstances(request)

	if err != nil {
		return []string{}, err
	}

	instanceIds := []string{}
	for _, res := range result.Reservations {
		for _, inst := range res.Instances {
			instanceIds = append(instanceIds, *inst.InstanceId)
		}
	}

	return instanceIds, nil
}

func getJobsPerNode(jobIds []string, jobsPerNode uint) [][]string {
	// Absurd to not allow at least 1 job on a node! What kind of node is that!!
	if jobsPerNode < 1 {
		jobsPerNode = 1
	}
	// Pass jobs to each node to fill up their capacity. The last one can have less than full capacity
	result := [][]string{}
	nodesNeeded := uint(float32(len(jobIds))/float32(jobsPerNode) + 0.5)
	if nodesNeeded <= 0 {
		nodesNeeded = 1 // we need at least ONE! If it has many cores it might run our entire job
	}

	if (len(jobIds) - int(jobsPerNode*nodesNeeded)) > 0 {
		nodesNeeded = nodesNeeded + 1
	}

	for c := uint(0); c < nodesNeeded; c++ {
		jobs := []string{}

		for j := uint(0); j < jobsPerNode; j++ {
			jIdx := uint(c*jobsPerNode + j)
			if jIdx >= uint(len(jobIds)) {
				break
			}

			jobs = append(jobs, jobIds[jIdx])
		}

		result = append(result, jobs)
	}

	return result
}

// Starts enough nodes to handle the job ids passed

// If there is a JobAWSSecret configured we start the node on a new EC2 instance,
// otherwise (for testing really) we just start it in a new thread

func (jm *JobManager) startJobNodes(jobIds []string) error {
	if len(jobIds) <= 0 {
		return fmt.Errorf("startJobNodes: No job ids specified")
	}

	if jm.isLocalTestMode() {
		// No JobAWSSecret configured, so we just run in local mode. If we have not
		// yet started a job node thread, start one now
		jm.svcs.Log.Debugf("  startJobNodes running in local mode, ensuring one job node thread is running...")

		if jm.localJobNode != nil {
			jm.svcs.Log.Infof("  startJobNodes skipped, already running a local one")
			return nil
		}

		// Start a local one
		jm.svcs.Log.Infof("  startJobNodes starting local job node")
		jm.localJobNode = jobnode.CreateJobNode(
			"local-job",
			jm.svcs.Config.Jobs.RunnerDockerImage,
			jm.svcs.Config.PiquantJobsBucket,
			jm.svcs.InstanceId,
			jm.svcs.FS,
			jm.svcs.MongoDB,
			jm.svcs.Log,
			jm.svcs.TimeStamper)

		jm.localJobNode.StartJobs(jobIds)

		return nil
	}

	jm.svcs.Log.Debugf("  Querying running node count...")

	// Read the credentials from secrets manager
	awsKey, awsSecret, awsRegion, err := readSecretsManager(jm.svcs.SecretsManager, jm.svcs.Config.Jobs.AWSSecret)
	if err != nil {
		return fmt.Errorf("JobNode AWS secret read failed: %v", err)
	}

	instanceIds, err := jm.getRunningNodes()
	if err != nil {
		return err
	}

	jm.svcs.Log.Debugf("  Instance IDs retrieved: %v", strings.Join(instanceIds, ","))

	// If this seems like way too many jobs, stop here, so we don't infinitely start up EC2s
	if len(instanceIds) > int(jm.svcs.Config.Jobs.MaxQuantNodes)*4 {
		return fmt.Errorf("Too many job nodes active (%v), no more will be started", len(instanceIds))
	}

	// Change their state, we're assigning them...
	nowUnixSec := jm.svcs.TimeStamper.GetTimeNowSec()
	ctx := context.TODO()

	filter := bson.M{"_id": bson.M{"$in": jobIds}}
	dbResult, err := jm.svcs.MongoDB.Collection(dbCollections.JobQueueName).UpdateMany(ctx, filter, bson.D{{Key: "$set", Value: bson.M{
		"state":                       protos.JobQueueItem_ASSIGNED,
		"lastupdatedtimestampunixsec": nowUnixSec,
	}}})

	if err != nil {
		return fmt.Errorf("Failed to set jobs to assigned state: %v", err)
	}

	if dbResult.ModifiedCount != int64(len(jobIds)) {
		jm.svcs.Log.Infof("  WARNING: startJobNodes expected modified count of %v, got %v", len(jobIds), dbResult.ModifiedCount)
	}

	// Work out how many job nodes are needed.
	jobsForNodes := getJobsPerNode(jobIds, jm.svcs.Config.Jobs.CoresPerNode)

	// Start each node
	allStartedIds := []*string{}
	for _, jobs := range jobsForNodes {
		jm.svcs.Log.Debugf("  Starting EC2 job node for jobs: %v...", strings.Join(jobs, ","))
		startedIds, err := jm.startEC2JobNode(jobs, awsKey, awsSecret, awsRegion)
		if err != nil {
			return err
		}

		allStartedIds = append(allStartedIds, startedIds...)
	}

	input := &ec2.DescribeInstancesInput{InstanceIds: allStartedIds}
	err = jm.svcs.EC2.WaitUntilInstanceRunning(input)
	if err != nil {
		jm.svcs.Log.Infof("  WARNING: Failed to wait for instances to start running: %v", err)
	}

	jm.svcs.Log.Debugf("  %v nodes started.", len(jobsForNodes))
	return nil
}
