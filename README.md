# Pixlise Core


[![DOI](https://zenodo.org/badge/520044172.svg)](https://zenodo.org/badge/latestdoi/520044172)


[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/pixlise/core)

 
[![pipeline status](https://github.com/pixlise/go-rest-api/badges/master/pipeline.svg)](https://github.com/pixlise/go-rest-api/-/commits/master)
[![coverage report](https://github.com/pixlise/go-rest-api/badges/master/coverage.svg)](https://github.com/pixlise/go-rest-api/-/commits/master)

## What is it?

Pixlise Core is the API and data management processes for the Pixlise platform. 

## Building

The core package is written in Golang and contains a number of components required for deplyoment of the Pixlise platform. The simplest way to build the code is to run

``` shell
make build
```

within the project root directory. This will build a number of binary files that are then located in the `_out` directory. The main API is called `pixlise-api-xxx` where xxx is the target architecture. 
By default we build for Mac, Linux and Windows.

## Running

Executing the API requires an environment variable to be set, this variable is named CUSTOM_CONFIG and is an encoded JSON string:

Full Example:

```
CUSTOM_CONFIG='{"AWSBucketRegion":"us-east-1","AWSCloudwatchRegion":"us-east-1","AdminEmails":["someemail@myemail.com"],"ArtifactsBucket":"xxx-artifacts-s3-bucket","Auth0Domain":"xxx.auth0.com","Auth0ManagementClientID":"xxx","Auth0ManagementSecret":"xxx","BuildsBucket":"xxx-builds-s3-bucket","ConfigBucket":"xxx-config-s3-bucket","CoresPerNode":4,"DataBucket":"xxx-data-bucket","DataSourceSNSTopic":"xxx-sns-topic","DatasetsBucket":"xxx-datasets-bucket","DatasourceArtifactsBucket":"xxx-artifacts-bucket","DockerLoginString":"xxx-docker-login","ElasticPassword":"xxx-es-password","ElasticURL":"xxx-elasticendpoint","ElasticUser":"logger","EnvironmentName":"xxx-envname","HotQuantNamespace":"piquant-fit","JobBucket":"xxx-job-bucket","KubernetesLocation":"internal","LogLevel":1,"ManualUploadBucket":"xxx-manual-upload","PiquantDockerImage":"xxx-piquant-image","PiquantJobsBucket":"xxx-job-bucket","PosterImage":"xxx-poster-image","QuantDestinationPackage":"xxx-destination-package","QuantExecutor":"kubernetes","QuantNamespace":"xxx-namespace","QuantObjectType":"xxx-quant-object-type","SentryEndpoint":"xxx-sentry-endpoint","UserDbSecretName":"xxx-docdb-secret","UsersBucket":"xxx-users-bucket"}'
```

Minimal Example:

`TODO`

Then execute the binary file and your API should come to life.

### Docker / Kubernetes

`TODO`

### Required Env Vars

AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
AWS_REGION=us-west-1

## Developing in Gitpod

If you're wondering what the Gitpod button above is and would like to get a development environment up and running easily, visit the documentation [here](https://pixlise.gitlab.io/documentation/docs/build-and-release/getting-started/) for more info.

## Debugging in VS Code

- Download the source.
- Configure your .vscode/launch.json file to supply the following to start debugger:
    "env": {
        "AWS_ACCESS_KEY_ID":"<<< LOOK THIS UP! >>>",
        "AWS_SECRET_ACCESS_KEY":"<<< LOOK THIS UP! >>>",
        "AWS_DEFAULT_REGION":"us-east-1",
        "AWS_S3_US_EAST_1_REGIONAL_ENDPOINT":"regional",
    },
    "args": ["-quantExecutor", "docker"]
- Open any file in the main package (cmd/run-api/*.go)
- Hit F5 to start debugging


