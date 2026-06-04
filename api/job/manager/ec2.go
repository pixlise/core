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
func (jm *JobManager) startEC2JobNode(waitTillStarted bool) error {
	jm.nodesStarted = jm.nodesStarted + 1
	if jm.svcs.Config.JobMaxNodeRunTimeSec < 60 {
		return fmt.Errorf("Cannot start job node that runs for only %vsec", jm.svcs.Config.JobMaxNodeRunTimeSec)
	}

	if len(jm.svcs.Config.JobAWSSecret) <= 0 {
		return fmt.Errorf("JobNode AWS secret not set")
	}

	// Read the credentials from secrets manager
	awsKey, awsSecret, awsRegion, err := readSecretsManager(jm.svcs.SecretsManager, jm.svcs.Config.JobAWSSecret)
	if err != nil {
		return fmt.Errorf("JobNode AWS secret read failed: %v", err)
	}

	startupScript := fmt.Sprintf(`#!/bin/bash
set -e
echo "Starting PIXLISE job node, limited to %v sec runtime"

# Auto-shutdown instance in requested time
(sleep %v && shutdown -h now) &

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
./pixlise-job-node -bucket "%v" -jobContainer "%v" -instanceId "%v" -mongoSecret "%v" -envName "%v" -maxJobs "%v" -maxIdleSec "%v" -maxRunTimeSec "%v"

echo "PIXLISE job node running"
`,
		jm.svcs.Config.JobMaxNodeRunTimeSec,
		jm.svcs.Config.JobMaxNodeRunTimeSec,
		awsKey, awsSecret,
		awsRegion, awsRegion,
		jm.svcs.Config.JobNodeS3Path,
		jm.svcs.Config.PiquantJobsBucket,
		jm.svcs.Config.JobRunnerDockerImage,
		jm.svcs.InstanceId,
		jm.svcs.Config.MongoSecret,
		jm.svcs.Config.EnvironmentName,
		jm.svcs.Config.CoresPerNode,
		120,
		jm.svcs.Config.JobMaxNodeRunTimeSec-5,
	)

	input := &ec2.RunInstancesInput{
		// placement (AZ - not setting it here?!)
		ImageId:      aws.String(jm.svcs.Config.JobAMI),
		InstanceType: aws.String(jm.svcs.Config.JobInstanceType),
		TagSpecifications: []*ec2.TagSpecification{
			{
				ResourceType: aws.String("instance"),
				Tags: []*ec2.Tag{
					{Key: aws.String("Name"), Value: aws.String(fmt.Sprintf("job-%v-node-%v-[%v]", jm.svcs.Config.EnvironmentName, jm.nodesStarted, jm.svcs.InstanceId))},
					{Key: aws.String("pixlise:environment"), Value: aws.String(jm.svcs.Config.EnvironmentName)},
					{Key: aws.String("pixlise:starter-instance-id"), Value: aws.String(jm.svcs.InstanceId)},
				},
			},
		},
		KeyName:          aws.String(jm.svcs.Config.JobKeyName),
		SecurityGroupIds: []*string{aws.String(jm.svcs.Config.JobSecurityGroup)},
		MaxCount:         aws.Int64(1),
		MinCount:         aws.Int64(1),
		UserData:         aws.String(base64.StdEncoding.EncodeToString([]byte(startupScript))),
	}

	res, err := jm.svcs.EC2.RunInstances(input)
	if err != nil {
		return err
	}
	fmt.Printf("%+v\n", res)

	if waitTillStarted {
		input := &ec2.DescribeInstancesInput{InstanceIds: []*string{aws.String(*res.Instances[0].InstanceId)}}
		err = jm.svcs.EC2.WaitUntilInstanceRunning(input)
	}

	return nil
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
	// Only grab instances that are running or just started
	filters := []*ec2.Filter{
		{
			Name:   aws.String("instance-state-name"),
			Values: []*string{aws.String("running"), aws.String("pending")},
		},
		{
			Name:   aws.String("pixlise:environment"),
			Values: []*string{aws.String(jm.svcs.Config.EnvironmentName)},
		},
	}

	// TODO: check names too!

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

// Ensures there's enough nodes waiting for jobs - if we haven't had a quant
// in a while the old nodes would've shut down already!

// If there is a JobAWSSecret configured we start the node on a new EC2 instance,
// otherwise (for testing really) we just start it in a new thread and only create
// one, so ignore future calls

func (jm *JobManager) ensureJobNodesRunning(outstandingJobCount int) error {
	if len(jm.svcs.Config.JobAWSSecret) > 0 {
		jm.svcs.Log.Debugf("  Querying running node count...")
		instanceIds, err := jm.getRunningNodes()
		if err != nil {
			return err
		}

		jm.svcs.Log.Debugf("  Instance IDs retrieved: %v\n", strings.Join(instanceIds, ","))

		if outstandingJobCount > 0 && len(instanceIds) <= 0 {
			jm.svcs.Log.Debugf("  Starting EC2 job node...")
			return jm.startEC2JobNode(false)
		}

		jm.svcs.Log.Debugf("  No job node started.\n")
		return nil
	}

	// No JobAWSSecret configured, so we just run in local mode. If we have not
	// yet started a job node thread, start one now
	jm.svcs.Log.Debugf("  EnsureJobNodesRunning running in local mode, ensuring one job node thread is running...")

	if jm.localJobNode != nil {
		jm.svcs.Log.Infof("  EnsureJobNodesRunning skipped, already running a local one")
		return nil
	}

	// Start a local one
	jm.svcs.Log.Infof("  EnsureJobNodesRunning starting local job node")
	jm.localJobNode = jobnode.CreateJobNode("local-job", jm.svcs.Config.JobRunnerDockerImage, jm.svcs.Config.PiquantJobsBucket, 6, jm.svcs.InstanceId, jm.svcs.FS, jm.svcs.MongoDB, jm.svcs.Log, jm.svcs.TimeStamper)

	jm.localJobNode.CheckStartupJobs()
	go jm.localJobNode.ListenToJobQueue()

	return nil
}
