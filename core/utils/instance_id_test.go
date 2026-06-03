package utils

func Example_utils_GetInstanceId() {
	GetInstanceId(1)

	// Output:
	// Failed to request EC2 metdata token: Put "http://169.254.169.254/latest/api/token": context deadline exceeded (Client.Timeout exceeded while awaiting headers)
}
