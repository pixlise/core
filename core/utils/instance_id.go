package utils

import (
	"fmt"
	"io"
	"net/http"
	"time"
)

func GetInstanceId(timeoutSec int) string {
	instanceId := RandStringBytesMaskImpr(16)

	// If we are on EC2 we may be able to query its instance ID here
	putReq, err := http.NewRequest("PUT", "http://169.254.169.254/latest/api/token", nil)
	if err != nil {
		fmt.Printf("Failed to create request for EC2 metdata instance_id: %v\n", err)
	} else {
		putReq.Header.Set("X-aws-ec2-metadata-token-ttl-seconds", "21600")

		client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
		putResponse, err := client.Do(putReq)
		if err != nil {
			fmt.Printf("Failed to request EC2 metdata token: %v\n", err)
		} else {
			defer putResponse.Body.Close()
			putBody, err := io.ReadAll(putResponse.Body)
			if err != nil {
				fmt.Printf("Failed to read EC2 metdata token: %v\n", err)
			} else if len(putBody) > 0 {
				// We got the token, now send the actual request for instance id
				token := string(putBody)

				putReq, err = http.NewRequest("PUT", "http://169.254.169.254/latest/meta-data/instance-id", nil)
				if err != nil {
					fmt.Printf("Failed to create request for EC2 metdata instance_id: %v\n", err)
				} else {
					putReq.Header.Set("X-aws-ec2-metadata-token", token)

					client := &http.Client{Timeout: time.Duration(timeoutSec) * time.Second}
					putResponse, err := client.Do(putReq)
					if err != nil {
						fmt.Printf("Failed to request EC2 metdata instance_id: %v\n", err)
					} else {
						defer putResponse.Body.Close()
						putBody, err := io.ReadAll(putResponse.Body)
						if err != nil {
							fmt.Printf("Failed to read EC2 metdata instance_id: %v\n", err)
						} else if len(putBody) > 0 {
							instanceId = string(putBody)
						}
					}
				}
			}
		}
	}

	return instanceId
}
