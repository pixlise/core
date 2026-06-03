package utils

func Example_utils_GetInstanceId() {
	GetInstanceId(1)

	// If using HTTP requests, we got:
	// Failed to request EC2 metdata token: Put "http://169.254.169.254/latest/api/token": context deadline exceeded (Client.Timeout exceeded while awaiting headers)

	// Output:
	// Failed to fetch instance ID from IMDS: operation error ec2imds: GetMetadata, request canceled, context deadline exceeded
}
