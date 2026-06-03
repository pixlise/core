package utils

import (
	"context"
	"fmt"
	"io"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/ec2/imds"
)

func GetInstanceId(timeoutSec int) string {
	instanceId := RandStringBytesMaskImpr(16)

	// Warning: this was AI generated
	cfg, err := config.LoadDefaultConfig(context.TODO())
	if err != nil {
		fmt.Printf("Unable to load SDK config: %v\n", err)
		return instanceId
	}

	client := imds.NewFromConfig(cfg)

	input := &imds.GetMetadataInput{
		Path: "instance-id",
	}

	output, err := client.GetMetadata(context.TODO(), input)
	if err != nil {
		fmt.Printf("Failed to fetch instance ID from IMDS: %v\n", err)
		return instanceId
	}

	instIdBody, err := io.ReadAll(output.Content)
	if err != nil {
		fmt.Printf("Failed to read instance ID from IMDS: %v\n", err)
		return instanceId
	}

	instanceId = string(instIdBody)
	/*
		// If we are on EC2 we may be able to query its instance ID here
		putReq, err := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", bytes.NewBuffer([]byte{}))
		if err != nil {
			fmt.Printf("Failed to create request for EC2 metdata instance_id: %v\n", err)
			return instanceId
		}

		putReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")

		client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
		putRespToken, err := client.Do(putReq)
		if err != nil {
			fmt.Printf("Failed to request EC2 metdata token: %v\n", err)
			return instanceId
		}

		defer putRespToken.Body.Close()
		putBodyToken, err := io.ReadAll(putRespToken.Body)
		if err != nil {
			fmt.Printf("Failed to read EC2 metdata token: %v\n", err)
			return instanceId
		}

		if len(putBodyToken) <= 0 {
			fmt.Printf("Token: %v\n", putBodyToken)
			return instanceId
		}

		// We got the token, now send the actual request for instance id
		token := string(putBodyToken)

		putReq, err = http.NewRequest("PUT", "http://169.254.169.254/latest/meta-data/instance-id", bytes.NewBuffer([]byte{}))
		if err != nil {
			fmt.Printf("Failed to create request for EC2 metdata instance_id: %v\n", err)
			return instanceId
		}

		putReq.Header.Set("X-aws-ec2-metadata-token", token)

		putRespInst, err := client.Do(putReq)
		if err != nil {
			fmt.Printf("Failed to request EC2 metdata instance_id: %v\n", err)
			return instanceId
		}

		defer putRespInst.Body.Close()
		putBodyInst, err := io.ReadAll(putRespInst.Body)
		if err != nil {
			fmt.Printf("Failed to read EC2 metdata instance_id: %v\n", err)
			return instanceId
		}

		if len(putBodyInst) > 0 {
			instanceId = string(putBodyInst)
		}
	*/
	return instanceId
}
