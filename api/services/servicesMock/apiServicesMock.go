package servicesMock

import (
	"github.com/pixlise/core/v4/api/config"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/core/awsutil"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/logger"
)

const DatasetsBucketForUnitTest = "datasets-bucket"
const ConfigBucketForUnitTest = "config-bucket"
const UsersBucketForUnitTest = "users-bucket"
const JobBucketForUnitTest = "job-bucket"

func MakeMockSvcs(mockS3 *awsutil.MockS3Client, idGen idgen.IDGenerator, logLevel *logger.LogLevel) services.APIServices {
	return makeMockSvcs(fileaccess.MakeS3Access(mockS3), idGen, logLevel)
}

func MakeMockSvcsWithFS(bucketRootPath string, idGen idgen.IDGenerator, logLevel *logger.LogLevel) services.APIServices {
	return makeMockSvcs(fileaccess.MakeFSAccessS3Simulator(bucketRootPath), idGen, logLevel)
}

func makeMockSvcs(fs fileaccess.FileAccess, idGen idgen.IDGenerator, logLevel *logger.LogLevel) services.APIServices {
	logging := logger.LogDebug
	if logLevel != nil {
		logging = *logLevel
	}

	cfg := config.APIConfig{
		DatasetsBucket:     DatasetsBucketForUnitTest,
		ConfigBucket:       ConfigBucketForUnitTest,
		UsersBucket:        UsersBucketForUnitTest,
		PiquantJobsBucket:  JobBucketForUnitTest,
		EnvironmentName:    "unit-test",
		LogLevel:           logging,
		KubernetesLocation: "external",
		QuantExecutor:      "null",
		NodeCountOverride:  0,
		DataSourceSNSTopic: "arn:1:2:3:4:5",
	}

	return services.APIServices{
		Config: cfg,
		Log:    &logger.NullLogger{},
		//AWSSessionCW: nil,
		//S3:           mockS3,
		SNS:       &awsutil.MockSNS{},
		JWTReader: MockJWTReader{},
		IDGen:     idGen,
		//Signer:       signer,
		FS: fs,
	}
}
