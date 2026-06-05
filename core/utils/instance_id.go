package utils

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

// Always returns an ID
// If it's an EC2 instance ID, the boolean returned is true
// If there's an error, error is set - but even then, a random ID is returned
func GetInstanceId() (string, bool, error) {
	instanceId := RandStringBytesMaskImpr(16)

	// Warning: this was AI generated
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return instanceId, false, fmt.Errorf("Unable to load SDK config: %v\n", err)
	}

	client := imds.NewFromConfig(cfg)

	input := &imds.GetMetadataInput{
		Path: "instance-id",
	}

	output, err := client.GetMetadata(context.TODO(), input)
	if err != nil {
		return instanceId, false, fmt.Errorf("Failed to fetch instance ID from IMDS: %v\n", err)
	}

	instIdBody, err := io.ReadAll(output.Content)
	if err != nil {
		return instanceId, false, fmt.Errorf("Failed to read instance ID from IMDS: %v\n", err)
	}

	return string(instIdBody), true, nil
}
