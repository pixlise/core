SHELL=/bin/bash -o pipefail

PROJECT_NAME := core
PKG := github.com/pixlise/$(PROJECT_NAME)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)

.PHONY: all build clean test lint

all: codegen build test lint

lint: ## Lint the files
	echo "${PKG}"
	#golint -set_exit_status ${PKG_LIST}
	golint ${PKG_LIST}

test: ## Run unittests
	mkdir -p _out
	go test -p 1 -v ./...

codegen:
	./genproto.sh checkgen

build: build-linux build-mac

build-linux:
	mkdir -p _out
	echo "version: ${BUILD_VERSION}"
	echo "sha: ${GITHUB_SHA}"
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/pixlise/core/v2/api/services.ApiVersion=${BUILD_VERSION}' -X 'github.com/pixlise/core/v2/api/services.GitHash=${GITHUB_SHA}'" -v -o ./_out/pixlise-api-linux ./internal/pixlise-api
	GOOS=linux GOARCH=amd64 go build -v -o ./_out/jobupdater-linux ./internal/lambdas/quant-job-updater
	GOOS=linux GOARCH=amd64 go build -v -o ./_out/datasourceupdater-linux ./internal/lambdas/dataset-tile-updater
	GOOS=linux GOARCH=amd64 go build -v -o ./_out/integrationtest-linux ./internal/cmdline-tools/api-integration-test
	GOOS=linux GOARCH=amd64 go build -v -o ./_out/dataimport-linux ./internal/lambdas/data-import
	GOOS=linux GOARCH=amd64 go build -v -o ./_out/importtest-linux ./internal/cmdline-tools/import-integration-test

build-mac:
	GOPRIVATE=github.com/pixlise GOOS=darwin GOARCH=amd64 go build -ldflags "-X services.ApiVersion=${BUILD_VERSION} -X services.GitHash=${GITHUB_SHA}" -v -o ./_out/pixlise-api-mac ./internal/pixlise-api
	GOPRIVATE=github.com/pixlise GOOS=darwin GOARCH=amd64 go build -v -o ./_out/jobupdater-mac ./internal/lambdas/quant-job-updater

build-windows:
	GOPRIVATE=github.com/pixlise GOOS=windows GOARCH=amd64 go build -ldflags "-X services.ApiVersion=${BUILD_VERSION} -X services.GitHash=${GITHUB_SHA}" -v -o ./_out/pixlise-api-windows ./internal/pixlise-api

clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
