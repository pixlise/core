package main

import "github.com/pixlise/core/v4/core/fileaccess"

func migrateConfigs(srcConfigBucket string, destConfigBucket string, fs fileaccess.FileAccess) {
	// New PIXLISE only requires these:
	toCopy := []string{
		"PixliseConfig/auth0.pem",
		"DatasetConfig/StandardPseudoIntensities-2023.csv",
		"DatasetConfig/StandardPseudoIntensities.csv",
	}
	failOnError := make([]bool, len(toCopy))
	for c := range failOnError {
		failOnError[c] = true
	}

	s3Copy(fs, srcConfigBucket, toCopy, destConfigBucket, toCopy, failOnError)
}
