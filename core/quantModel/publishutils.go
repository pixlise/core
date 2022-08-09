package quantModel

import (
	"fmt"
)

func generatePodNamePrefix(jobId string) string {
	return fmt.Sprintf("%s-%s", "quantpublish", jobId)
}

func generatePodLabels(jobid string, datasetid string, environment string) map[string]string {
	m := make(map[string]string)
	m["jobId"] = jobid
	m["datasetId"] = datasetid
	m["environment"] = environment

	return m
}
