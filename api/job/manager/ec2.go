package jobmanager

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/ec2"
	"github.com/aws/aws-sdk-go/service/secretsmanager"
	"github.com/aws/aws-secretsmanager-caching-go/secretcache"
	"github.com/pixlise/core/v4/api/job/jobnode"
)

// Called to start a job node
func (jm *JobManager) startEC2JobNode(jobIds []string, waitTillStarted bool) error {
	if len(jobIds) <= 0 || len(jobIds) > int(jm.svcs.Config.CoresPerNode) {
		return fmt.Errorf("Invalid job count when starting EC2 job nodes: %v", len(jobIds))
	}

	// Ensure no jobs have , in their ids because we'll be putting them in a string list separated by ,
	for _, id := range jobIds {
		if strings.Contains(id, ",") {
			return fmt.Errorf("Invalid job id specified, illegal , character detected: %v", id)
		}
	}

	jobIdListStr := strings.Join(jobIds, ",")

	if jm.svcs.Config.JobMaxNodeRunTimeSec < 60 {
		return fmt.Errorf("Cannot start job node that runs for only %vsec", jm.svcs.Config.JobMaxNodeRunTimeSec)
	}

	if len(jm.svcs.Config.JobAWSSecret) <= 0 {
		return fmt.Errorf("JobNode AWS secret not set")
	}

	if jm.startedNodeCount > jm.svcs.Config.MaxQuantNodes || jm.startedNodeCount > 10 {
		return fmt.Errorf("Not starting job node, hard testing limit has been reached")
	}

	// Read the credentials from secrets manager
	awsKey, awsSecret, awsRegion, err := readSecretsManager(jm.svcs.SecretsManager, jm.svcs.Config.JobAWSSecret)
	if err != nil {
		return fmt.Errorf("JobNode AWS secret read failed: %v", err)
	}

	jobNodeInstanceName := fmt.Sprintf("job-node-%v", jm.svcs.Config.EnvironmentName)

	startupScript := fmt.Sprintf(`#!/bin/bash
set -e
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

echo "Running job node..."
./pixlise-job-node -bucket "%v" -jobContainer "%v" -mongoSecret "%v" -envName "%v" -maxRunTimeSec "%v" -jobs "%v"

echo "PIXLISE job node shutting down"
shutdown -h now
`,
		jm.svcs.Config.JobMaxNodeRunTimeSec,
		awsKey, awsSecret,
		awsRegion, awsRegion,
		jm.svcs.Config.JobNodeS3Path,
		jm.svcs.Config.PiquantJobsBucket,
		jm.svcs.Config.JobRunnerDockerImage,
		jm.svcs.Config.MongoSecret,
		jm.svcs.Config.EnvironmentName,
		jm.svcs.Config.JobMaxNodeRunTimeSec-5,
		jobIdListStr,
	)

	input := &ec2.RunInstancesInput{
		// placement (AZ - not setting it here?!)
		ImageId:      aws.String(jm.svcs.Config.JobAMI),
		InstanceType: aws.String(jm.svcs.Config.JobInstanceType),
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
		KeyName:          aws.String(jm.svcs.Config.JobKeyName),
		SecurityGroupIds: []*string{aws.String(jm.svcs.Config.JobSecurityGroup)},
		MaxCount:         aws.Int64(int64(1)),
		MinCount:         aws.Int64(int64(1)),
		UserData:         aws.String(base64.StdEncoding.EncodeToString([]byte(startupScript))),
	}

	res, err := jm.svcs.EC2.RunInstances(input)
	if err != nil {
		return err
	}

	// List all instances started
	instances := []*string{}
	instanceStrs := []string{}
	for _, inst := range res.Instances {
		instances = append(instances, inst.InstanceId)
		instanceStrs = append(instanceStrs, *inst.InstanceId)
	}

	jm.svcs.Log.Infof("Started %v instances [%v]", len(instances), strings.Join(instanceStrs, ","))
	jm.startedNodeCount = jm.startedNodeCount + 1

	if waitTillStarted {
		input := &ec2.DescribeInstancesInput{InstanceIds: instances}
		err = jm.svcs.EC2.WaitUntilInstanceRunning(input)
	}

	return err
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
	if len(jm.svcs.Config.JobAWSSecret) <= 0 {
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

	if len(jm.svcs.Config.JobAWSSecret) > 0 {
		jm.svcs.Log.Debugf("  Querying running node count...")
		instanceIds, err := jm.getRunningNodes()
		if err != nil {
			return err
		}

		jm.svcs.Log.Debugf("  Instance IDs retrieved: %v", strings.Join(instanceIds, ","))

		// If this seems like way too many jobs, stop here, so we don't infinitely start up EC2s
		if len(instanceIds) > int(jm.svcs.Config.MaxQuantNodes)*4 {
			return fmt.Errorf("Too many job nodes active (%v), no more will be started", len(instanceIds))
		}

		// Work out how many job nodes are needed.
		jobsForNodes := getJobsPerNode(jobIds, jm.svcs.Config.CoresPerNode)

		// Start each node
		for _, jobs := range jobsForNodes {
			jm.svcs.Log.Debugf("  Starting EC2 job node for jobs: %v...", strings.Join(jobs, ","))
			err = jm.startEC2JobNode(jobs, true)
			if err != nil {
				return err
			}
		}
		jm.svcs.Log.Debugf("  %v nodes started.", len(jobsForNodes))
		return nil
	}

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
		jm.svcs.Config.JobRunnerDockerImage,
		jm.svcs.Config.PiquantJobsBucket,
		jm.svcs.InstanceId,
		jm.svcs.FS,
		jm.svcs.MongoDB,
		jm.svcs.Log,
		jm.svcs.TimeStamper)

	jm.localJobNode.StartJobs(jobIds)

	return nil
}
