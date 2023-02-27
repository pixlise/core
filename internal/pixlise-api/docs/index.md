# Building the codebase

## Running the API locally

Assuming you have cloned the repository to a development computer, you can then open the code in your favourite IDE. If you're looking to work on the API then you can launch it locally by running the main function in internal/pixlise-api/apiMain.go [https://github.com/pixlise/core/blob/development/internal/pixlise-api/apiMain.go#L82](https://github.com/pixlise/core/blob/development/internal/pixlise-api/apiMain.go#L82)

You also need to set the CUSTOM\_CONFIG environment variable with a config similar to the following:

```json
{
	"AWSBucketRegion": "us-east-1",
	"AWSCloudwatchRegion": "us-east-1",
	"AdminEmails": ["someemail@myemail.com"],
	"ArtifactsBucket": "xxx-artifacts-s3-bucket",
	"Auth0Domain": "xxx.auth0.com",
	"Auth0ManagementClientID": "xxx",
	"Auth0ManagementSecret": "xxx",
	"BuildsBucket": "xxx-builds-s3-bucket",
	"ConfigBucket": "xxx-config-s3-bucket",
	"CoresPerNode": 4,
	"DataBucket": "xxx-data-bucket",
	"DataSourceSNSTopic": "xxx-sns-topic",
	"DatasetsBucket": "xxx-datasets-bucket",
	"DockerLoginString": "xxx-docker-login",
	"MongoSecret": "xxx-mongo-secret",
	"MongoEndpoint": "xxx-mongo-endpoint",
	"MongoUsername": "mongo-user",
	"EnvironmentName": "xxx-envname",
	"HotQuantNamespace": "piquant-fit",
	"JobBucket": "xxx-job-bucket",
	"KubernetesLocation": "internal",
	"LogLevel": 1,
	"ManualUploadBucket": "xxx-manual-upload",
	"PiquantDockerImage": "xxx-piquant-image",
	"PiquantJobsBucket": "xxx-job-bucket",
	"PosterImage": "xxx-poster-image",
	"QuantDestinationPackage": "xxx-destination-package",
	"QuantExecutor": "kubernetes",
	"QuantNamespace": "xxx-namespace",
	"QuantObjectType": "xxx-quant-object-type",
	"SentryEndpoint": "xxx-sentry-endpoint",
	"UserDbSecretName": "xxx-docdb-secret",
	"UsersBucket": "xxx-users-bucket"
}
```

## Compiling all the packages

There is a makefile available to build all the required packages.

### Default build

`make`

the default make command will build all the tooling for linux and the API for Mac along with the quantjob updater. It will also run the test suite prior to packaging.

### Build for Linux

`make build-linux`

### Build for Mac

`make build-mac`

### Build for Windows

`make build-windows`
