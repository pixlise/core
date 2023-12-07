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

Executing the API requires several environment variables to be set. These include ones related to AWS (see below). A config file is also read. It's path can be specified with a command line argument: customConfigPath. This config specifies buckets and other configuration parameters to allow the API to execute containers and log errors, etc.

To see the configuration JSON structure, look at the `APIConfig` structure in `/api/config/config.go`

### Docker / Kubernetes

`TODO`

### Required Environment Variables

- AWS_ACCESS_KEY_ID
- AWS_SECRET_ACCESS_KEY
- AWS_REGION=us-west-1

## Developing in Gitpod

If you're wondering what the Gitpod button above is and would like to get a development environment up and running easily, visit the documentation [here](https://pixlise.gitlab.io/documentation/docs/build-and-release/getting-started/) for more info.

## Debugging in VS Code

- Download the source.
- Add a new configuration to your `.vscode/launch.json` file with `program` set to `internal/api`, which supplies the following to start debugger:
```
    "env": {
        "AWS_ACCESS_KEY_ID":"<<< LOOK THIS UP! >>>",
        "AWS_SECRET_ACCESS_KEY":"<<< LOOK THIS UP! >>>",
        "AWS_DEFAULT_REGION":"us-east-1",
        "AWS_S3_US_EAST_1_REGIONAL_ENDPOINT":"regional",
        ... Any other env variables as needed
    },
```

- Start a local mongo DB in docker: run `local-mongo/startdb.sh`. On startup the DB is seeded with data from JSON files. This container can be stopped and will be deleted at that point.
- Hit debug for the config in VS Code

You may encounter errors related to having an old Go version. At time of writing PIXLISE Core requires Go version 1.21. VS Code may also want to install some plugins for Go development.

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