package utils

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

func GetInstanceId() (string, error) {
	instanceId := RandStringBytesMaskImpr(16)

	// Warning: this was AI generated
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		return instanceId, fmt.Errorf("Unable to load SDK config: %v\n", err)
	}

	client := imds.NewFromConfig(cfg)

	input := &imds.GetMetadataInput{
		Path: "instance-id",
	}

	output, err := client.GetMetadata(context.TODO(), input)
	if err != nil {
		return instanceId, fmt.Errorf("Failed to fetch instance ID from IMDS: %v\n", err)
	}

	instIdBody, err := io.ReadAll(output.Content)
	if err != nil {
		return instanceId, fmt.Errorf("Failed to read instance ID from IMDS: %v\n", err)
	}

	instanceId = string(instIdBody)

	return instanceId, nil
}
