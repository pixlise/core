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
func (jm *JobManager) startEC2JobNode(nodeCount int, waitTillStarted bool) error {
	if nodeCount <= 0 || nodeCount > int(jm.svcs.Config.MaxQuantNodes) {
		return fmt.Errorf("Invalid job count when starting EC2 job nodes: %v", nodeCount)
	}
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

	jobNodeInstanceName := fmt.Sprintf("job-node-%v-node", jm.svcs.Config.EnvironmentName)

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
		jobNodeInstanceName,
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
					{Key: aws.String("Name"), Value: aws.String(jobNodeInstanceName)},
					{Key: aws.String("pixlise:instance-use"), Value: aws.String("job-node")},
					{Key: aws.String("pixlise:environment"), Value: aws.String(jm.svcs.Config.EnvironmentName)},
					{Key: aws.String("pixlise:starter-instance-id"), Value: aws.String(jm.svcs.InstanceId)},
				},
			},
		},
		KeyName:          aws.String(jm.svcs.Config.JobKeyName),
		SecurityGroupIds: []*string{aws.String(jm.svcs.Config.JobSecurityGroup)},
		MaxCount:         aws.Int64(int64(nodeCount)),
		MinCount:         aws.Int64(int64(nodeCount)),
		UserData:         aws.String(base64.StdEncoding.EncodeToString([]byte(startupScript))),
	}

	res, err := jm.svcs.EC2.RunInstances(input)
	if err != nil {
		return err
	}

	// List all instances started
	instances := []*string{}
	for _, inst := range res.Instances {
		instances = append(instances, inst.InstanceId)
	}

	jm.svcs.Log.Infof("Started %v instances [%v]", len(instances), strings.Join(instances, ","))

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

		jm.svcs.Log.Debugf("  Instance IDs retrieved: %v", strings.Join(instanceIds, ","))

		if outstandingJobCount > 0 {
			// Work out how many job nodes are needed. If we have N outstanding jobs, and we can run X
			// jobs on a node...
			nodesNeeded := outstandingJobCount / int(jm.svcs.Config.CoresPerNode)

			if nodesNeeded <= 0 {
				nodesNeeded = 1
			} else if nodesNeeded > int(jm.svcs.Config.MaxQuantNodes) {
				nodesNeeded = int(jm.svcs.Config.MaxQuantNodes)
			}

			jm.svcs.Log.Debugf("  Starting %v EC2 job nodes...", nodesNeeded)
			err = jm.startEC2JobNode(nodesNeeded, true)
			if err != nil {
				return err
			}

			// Check how many instances we see now
			instanceIds, err = jm.getRunningNodes()

			if err != nil {
				jm.svcs.Log.Errorf("  Error after instance start and getRunningNodes: %v", err)
			}

			jm.svcs.Log.Infof("  After instance start, getRunningNodes sees %v instances: [%v]", len(instanceIds), string.Join(instanceIds, ","))
			return nil
		}

		jm.svcs.Log.Debugf("  No job node started.")
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


All jobs started with the same node name??? Did we just start individual nodes??
JobNodes seem to only have 1 docker container running???
JobNodes should probably write a log file!


DEBUG: CheckJobQueue found 1 job groups
DEBUG:   CheckJobQueue job group quant-q261zhj8qztn4xgb has 0 ran, 0 completed nodes
DEBUG:   CheckJobQueue found 6 not-started jobs
DEBUG:   Querying running node count...
DEBUG:   Instance IDs retrieved: 
DEBUG:   Starting 1 EC2 job nodes...
DEBUG:   Instance IDs retrieved: 
DEBUG:   Starting 1 EC2 job nodes...
DEBUG:   Instance IDs retrieved: 
DEBUG:   Starting 1 EC2 job nodes...
DEBUG:   Instance IDs retrieved: 
DEBUG:   Starting 1 EC2 job nodes...
DEBUG:   Instance IDs retrieved: 
DEBUG:   Starting 1 EC2 job nodes...
DEBUG:   Instance IDs retrieved: 
DEBUG:   Starting 1 EC2 job nodes...
{
  Instances: [{
      AmiLaunchIndex: 0,
      Architecture: "x86_64",
      BootMode: "uefi-preferred",
      CapacityReservationSpecification: {
        CapacityReservationPreference: "open"
      },
      ClientToken: "4E2A5638-6B84-4F93-A2E1-5ECE53021A16",
      CpuOptions: {
        CoreCount: 4,
        ThreadsPerCore: 2
      },
      CurrentInstanceBootMode: "uefi",
      EbsOptimized: false,
      EnaSupport: true,
      EnclaveOptions: {
        Enabled: false
      },
      Hypervisor: "xen",
      ImageId: "ami-00e801948462f718a",
      InstanceId: "i-0643585eb1e66ec2d",
      InstanceType: "t3.2xlarge",
      KeyName: "PixliseEBMongo",
      LaunchTime: 2026-06-04 04:33:56 +0000 UTC,
      MaintenanceOptions: {
        AutoRecovery: "default"
      },
      MetadataOptions: {
        HttpEndpoint: "enabled",
        HttpProtocolIpv6: "disabled",
        HttpPutResponseHopLimit: 2,
        HttpTokens: "required",
        InstanceMetadataTags: "disabled",
        State: "pending"
      },
      Monitoring: {
        State: "disabled"
      },
      NetworkInterfaces: [{
          Attachment: {
            AttachTime: 2026-06-04 04:33:56 +0000 UTC,
            AttachmentId: "eni-attach-0f40ad213e5b66508",
            DeleteOnTermination: true,
            DeviceIndex: 0,
            NetworkCardIndex: 0,
            Status: "attaching"
          },
          Description: "",
          Groups: [{
              GroupId: "sg-03617d16414a3431d",
              GroupName: "PixliseMongo"
            }],
          InterfaceType: "interface",
          MacAddress: "0e:d3:73:d0:b9:5b",
          NetworkInterfaceId: "eni-03144092e3382fc18",
          OwnerId: "963058736014",
          PrivateDnsName: "ip-172-31-35-79.ec2.internal",
          PrivateIpAddress: "172.31.35.79",
          PrivateIpAddresses: [{
              Primary: true,
              PrivateDnsName: "ip-172-31-35-79.ec2.internal",
              PrivateIpAddress: "172.31.35.79"
            }],
          SourceDestCheck: true,
          Status: "in-use",
          SubnetId: "subnet-6712143b",
          VpcId: "vpc-00d5837a"
        }],
      Placement: {
        AvailabilityZone: "us-east-1a",
        GroupName: "",
        Tenancy: "default"
      },
      PrivateDnsName: "ip-172-31-35-79.ec2.internal",
      PrivateDnsNameOptions: {
        EnableResourceNameDnsAAAARecord: false,
        EnableResourceNameDnsARecord: false,
        HostnameType: "ip-name"
      },
      PrivateIpAddress: "172.31.35.79",
      PublicDnsName: "",
      RootDeviceName: "/dev/xvda",
      RootDeviceType: "ebs",
      SecurityGroups: [{
          GroupId: "sg-03617d16414a3431d",
          GroupName: "PixliseMongo"
        }],
      SourceDestCheck: true,
      State: {
        Code: 0,
        Name: "pending"
      },
      StateReason: {
        Code: "pending",
        Message: "pending"
      },
      StateTransitionReason: "",
      SubnetId: "subnet-6712143b",
      Tags: [
        {
          Key: "pixlise:starter-instance-id",
          Value: "i-0bf0159123e1be326"
        },
        {
          Key: "pixlise:environment",
          Value: "prod"
        },
        {
          Key: "Name",
          Value: "job-prod-node-6-[i-0bf0159123e1be326]"
        },
        {
          Key: "pixlise:instance-use",
          Value: "job-node"
        }
      ],
      VirtualizationType: "hvm",
      VpcId: "vpc-00d5837a"
    }],
  OwnerId: "963058736014",
  ReservationId: "r-096f985d1ec38483a"
}

...
later on we seem to have started more job nodes?

}
INFO: HandleOnce: i-0bf0159123e1be326 chosen to handle job jobmanager-queue
DEBUG: CheckJobQueue found 1 job groups
DEBUG:   CheckJobQueue job group quant-q261zhj8qztn4xgb has 0 ran, 0 completed nodes
DEBUG:   CheckJobQueue found 2 not-started jobs
DEBUG:   Querying running node count...
INFO: HandleOnce: i-0bf0159123e1be326 chosen to handle job jobmanager-queue
DEBUG: CheckJobQueue found 1 job groups
DEBUG:   CheckJobQueue job group quant-q261zhj8qztn4xgb has 0 ran, 0 completed nodes
DEBUG:   CheckJobQueue found 2 not-started jobs
DEBUG:   Querying running node count...
DEBUG:   Instance IDs retrieved: i-0f3f62e09cff0645a,i-0e74fd0e014a5265d,i-0643585eb1e66ec2d,i-0526e2a0c2cfaa4a0,i-0b9fb76c50448063d,i-02f4921bcd42e8a62,i-0bb2de7f9b39ad2e6
DEBUG:   Starting 1 EC2 job nodes...
DEBUG:   Instance IDs retrieved: i-0f3f62e09cff0645a,i-0e74fd0e014a5265d,i-0643585eb1e66ec2d,i-0526e2a0c2cfaa4a0,i-0b9fb76c50448063d,i-02f4921bcd42e8a62,i-0bb2de7f9b39ad2e6
DEBUG:   Starting 1 EC2 job nodes...
INFO: HandleOnce: i-0bf0159123e1be326 chosen to handle job jobmanager-queue
DEBUG: CheckJobQueue found 1 job groups
DEBUG:   CheckJobQueue job group quant-q261zhj8qztn4xgb has 0 ran, 0 completed nodes
DEBUG:   CheckJobQueue found 2 not-started jobs
DEBUG:   Querying running node count...
DEBUG:   Instance IDs retrieved: i-0f3f62e09cff0645a,i-0e74fd0e014a5265d,i-0643585eb1e66ec2d,i-0526e2a0c2cfaa4a0,i-0b9fb76c50448063d,i-02f4921bcd42e8a62,i-0bb2de7f9b39ad2e6
DEBUG:   Starting 1 EC2 job nodes...
{
  Instances: [{
      AmiLaunchIndex: 0,
      Architecture: "x86_64",
      BootMode: "uefi-preferred",
      CapacityReservationSpecification: {
        CapacityReservationPreference: "open"
      },
      ClientToken: "C093B6C4-31DA-4A0F-8447-7C58460127FC",
      CpuOptions: {
        CoreCount: 4,
        ThreadsPerCore: 2
      },
      CurrentInstanceBootMode: "uefi",
      EbsOptimized: false,
      EnaSupport: true,
      EnclaveOptions: {
        Enabled: false
      },
      Hypervisor: "xen",
      ImageId: "ami-00e801948462f718a",
      InstanceId: "i-0855468baaed9bfe6",
      InstanceType: "t3.2xlarge",
      KeyName: "PixliseEBMongo",
      LaunchTime: 2026-06-04 04:35:12 +0000 UTC,
      MaintenanceOptions: {
        AutoRecovery: "default"
      },
      MetadataOptions: {
        HttpEndpoint: "enabled",
        HttpProtocolIpv6: "disabled",
        HttpPutResponseHopLimit: 2,
        HttpTokens: "required",
        InstanceMetadataTags: "disabled",
        State: "pending"
      },
      Monitoring: {
        State: "disabled"
      },
      NetworkInterfaces: [{
          Attachment: {
            AttachTime: 2026-06-04 04:35:12 +0000 UTC,
            AttachmentId: "eni-attach-0647baaaeaffca1d2",
            DeleteOnTermination: true,
            DeviceIndex: 0,
            NetworkCardIndex: 0,
            Status: "attaching"
          },
          Description: "",
          Groups: [{
              GroupId: "sg-03617d16414a3431d",
              GroupName: "PixliseMongo"
            }],
          InterfaceType: "interface",
          MacAddress: "0e:8a:bf:db:5d:b3",
          NetworkInterfaceId: "eni-091825d8d2bafdf64",
          OwnerId: "963058736014",
          PrivateDnsName: "ip-172-31-41-4.ec2.internal",
          PrivateIpAddress: "172.31.41.4",
          PrivateIpAddresses: [{
              Primary: true,
              PrivateDnsName: "ip-172-31-41-4.ec2.internal",
              PrivateIpAddress: "172.31.41.4"
            }],
          SourceDestCheck: true,
          Status: "in-use",
          SubnetId: "subnet-6712143b",
          VpcId: "vpc-00d5837a"
        }],
      Placement: {
        AvailabilityZone: "us-east-1a",
        GroupName: "",
        Tenancy: "default"
      },
      PrivateDnsName: "ip-172-31-41-4.ec2.internal",
      PrivateDnsNameOptions: {
        EnableResourceNameDnsAAAARecord: false,
        EnableResourceNameDnsARecord: false,
        HostnameType: "ip-name"
      },
      PrivateIpAddress: "172.31.41.4",
      PublicDnsName: "",
      RootDeviceName: "/dev/xvda",
      RootDeviceType: "ebs",
      SecurityGroups: [{
          GroupId: "sg-03617d16414a3431d",
          GroupName: "PixliseMongo"
        }],
      SourceDestCheck: true,
      State: {
        Code: 0,
        Name: "pending"
      },
      StateReason: {
        Code: "pending",
        Message: "pending"
      },
      StateTransitionReason: "",
      SubnetId: "subnet-6712143b",
      Tags: [
        {
          Key: "pixlise:starter-instance-id",
          Value: "i-0bf0159123e1be326"
        },
        {
          Key: "Name",
          Value: "job-prod-node-9-[i-0bf0159123e1be326]"
        },
        {
          Key: "pixlise:instance-use",
          Value: "job-node"
        },
        {
          Key: "pixlise:environment",
          Value: "prod"
        }
      ],
      VirtualizationType: "hvm",
      VpcId: "vpc-00d5837a"
    }],
  OwnerId: "963058736014",
  ReservationId: "r-0e5250b2a6cb93506"
}
{