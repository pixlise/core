# Pixlise Core


[![DOI](https://zenodo.org/badge/520044172.svg)](https://zenodo.org/badge/latestdoi/520044172)


[![Open in Gitpod](https://gitpod.io/button/open-in-gitpod.svg)](https://gitpod.io/#https://github.com/pixlise/core)

[![build](https://github.com/pixlise/core/actions/workflows/features.yml/badge.svg)](https://github.com/pixlise/core/actions/workflows/features.yml)
[![build](https://github.com/pixlise/core/actions/workflows/dev.yml/badge.svg?branch=development)](https://github.com/pixlise/core/actions/workflows/dev.yml)
[![build](https://github.com/pixlise/core/actions/workflows/release.yml/badge.svg?branch=main)](https://github.com/pixlise/core/actions/workflows/release.yml)

## What is it?

PIXLISE Core is the API and data management processes for the PIXLISE platform.

PIXLISE is deployed to https://www.pixlise.org

## Building

The core package is written in Golang and contains a number of components required for deplyoment of the PIXLISE platform. The simplest way to build the code is to run

``` shell
make build
```

within the project root directory. This will build a number of binary files that are then located in the `_out` directory. The main API is called `pixlise-api-xxx` where xxx is the target architecture. 
By default we build for Mac, Linux and Windows.

## Code Generation
- go install github.com/favadi/protoc-go-inject-tag@latest
- Run ./genproto.sh


## Run-time Configuration

Executing the API requires several environment variables to be set. These include ones related to AWS (see below), but we also supply a JSON configuration string in a single environment variable called CUSTOM_CONFIG. This specifies buckets and other configuration parameters to allow the API to execute containers and log errors, etc.

Full Example:

```
CUSTOM_CONFIG='{"AWSBucketRegion":"us-east-1","AWSCloudwatchRegion":"us-east-1","AdminEmails":["someemail@myemail.com"],"ArtifactsBucket":"xxx-artifacts-s3-bucket","Auth0Domain":"xxx.auth0.com","Auth0ManagementClientID":"xxx","Auth0ManagementSecret":"xxx","BuildsBucket":"xxx-builds-s3-bucket","ConfigBucket":"xxx-config-s3-bucket","CoresPerNode":4,"DataBucket":"xxx-data-bucket","DataSourceSNSTopic":"xxx-sns-topic","DatasetsBucket":"xxx-datasets-bucket","DockerLoginString":"xxx-docker-login","MongoSecret":"xxx-mongo-secret","MongoEndpoint":"xxx-mongo-endpoint","MongoUsername":"mongo-user","EnvironmentName":"xxx-envname","HotQuantNamespace":"piquant-fit","JobBucket":"xxx-job-bucket","KubernetesLocation":"internal","LogLevel":1,"ManualUploadBucket":"xxx-manual-upload","PiquantDockerImage":"xxx-piquant-image","PiquantJobsBucket":"xxx-job-bucket","PosterImage":"xxx-poster-image","QuantDestinationPackage":"xxx-destination-package","QuantExecutor":"kubernetes","QuantNamespace":"xxx-namespace","QuantObjectType":"xxx-quant-object-type","SentryEndpoint":"xxx-sentry-endpoint","UserDbSecretName":"xxx-docdb-secret","UsersBucket":"xxx-users-bucket"}'
```

Minimal Example:

`TODO`

Then execute the binary file and your API should come to life.

### Docker / Kubernetes

`TODO`

### Required Environment Variables

- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- AWS_REGION=us-west-1
- CUSTOM_CONFIG

## Developing in Gitpod

If you're wondering what the Gitpod button above is and would like to get a development environment up and running easily, visit the documentation [here](https://pixlise.gitlab.io/documentation/docs/build-and-release/getting-started/) for more info.

## Debugging in VS Code

- Download the source.
- Configure your .vscode/launch.json file to supply the following to start debugger:
```
    "env": {
        "AWS_ACCESS_KEY_ID":"<<< LOOK THIS UP! >>>",
        "AWS_SECRET_ACCESS_KEY":"<<< LOOK THIS UP! >>>",
        "AWS_DEFAULT_REGION":"us-east-1",
        "AWS_S3_US_EAST_1_REGIONAL_ENDPOINT":"regional",
    },
    "args": ["-quantExecutor", "docker"]
```
- Start a local mongo DB in docker: `docker run -d  --name mongo-on-docker  -p 27888:27017 -e MONGO_INITDB_ROOT_USERNAME=mongoadmin -e MONGO_INITDB_ROOT_PASSWORD=secret mongo`. This container can be stopped, deleted and recreated as needed.
- Open any file in the main package (`internal/pixlise-api/*.go`)
- Hit F5 to start debugging

You may encounter errors related to having an old Go version. At time of writing PIXLISE Core requires Go version 1.18. VS Code may also want to install some plugins for Go development.

The API takes a few seconds to start up. Watch the Debug Console in VS Code! You will see:
- A dump of the configuration the API started with
- Mongo DB connection status
- A listing of all API endpoints and what permission they require
- `"INFO: API Started..."` signifying the API is ready to accept requests

## Local Mongo database access

Download "MongoDB Compass" and when the docker container is running locally (in docker), connect to it with this connection string:
`mongodb://mongoadmin:secret@localhost:27888/?authMechanism=DEFAULT`

### Example CLI flags

`-quantExecutor docker` - this tells the API to use local docker as the quant executor, meaning PIQUANT jobs will start on your local development machine.

## Documentation

Given this is written in Go, it supports godoc! Being a public repository, documentation automatically is pulled into the online Go
documentation site, but to view documentation locally, you can run `godoc -http=:6060` and to export to a zip file you can create
a directory and run `godoc-static --destination=./doctest ./`. That last parameter being the current directory - if it's missed, then all go packages are documented (and somehow the ones in this project are not!)