SHELL=/bin/bash -o pipefail

PROJECT_NAME := core
PKG := github.com/pixlise/$(PROJECT_NAME)
PKG_LIST := $(shell go list ${PKG}/... | grep -v /vendor/)

.PHONY: all build clean unittest integrationtest lint

all: codegen build unittest integrationtest lint

lint: ## Lint the files
	echo "${PKG}"
	#golint -set_exit_status ${PKG_LIST}
	golint ${PKG_LIST}

unittest: ## Run unittests
	pwd
	cd ..
	mkdir -p _out
	go install github.com/favadi/protoc-go-inject-tag@latest
	go run ./data-formats/codegen/main.go -protoPath ./data-formats/api-messages/ -goOutPath ./api/ws/
	protoc-go-inject-tag -remove_tag_comment -input="./generated-protos/*.pb.go"
	go test -v ./...

integrationtest:
	mkdir -p _out
	echo "version: ${BUILD_VERSION}"
	echo "sha: ${GITHUB_SHA}"
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/pixlise/core/v4/api/services.ApiVersion=${BUILD_VERSION}' -X 'github.com/pixlise/core/v4/api/services.GitHash=${GITHUB_SHA}'" -v -o ./api-service ./internal/api
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/pixlise/core/v4/api/services.ApiVersion=${BUILD_VERSION}' -X 'github.com/pixlise/core/v4/api/services.GitHash=${GITHUB_SHA}'" -v -o ./internal/cmd-line-tools/api-integration-test/tester ./internal/cmd-line-tools/api-integration-test

codegen:
	./genproto.sh checkgen

build: build-linux build-mac

build-linux:
	mkdir -p _out
	echo "version: ${BUILD_VERSION}"
	echo "sha: ${GITHUB_SHA}"
	GOOS=linux GOARCH=amd64 go run ./data-formats/codegen/main.go -protoPath ./data-formats/api-messages/ -goOutPath ./api/ws/
	GOOS=linux GOARCH=amd64 go build -ldflags "-X 'github.com/pixlise/core/v4/api/services.ApiVersion=${BUILD_VERSION}' -X 'github.com/pixlise/core/v4/api/services.GitHash=${GITHUB_SHA}'" -v -o ./_out/pixlise-api-linux ./internal/api
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-X 'github.com/pixlise/core/v4/api/services.ApiVersion=${BUILD_VERSION}' -X 'github.com/pixlise/core/v4/api/services.GitHash=${GITHUB_SHA}'" -v -o ./_out/bootstrap ./internal/lambdas/data-import
	GOOS=linux GOARCH=amd64 go build -v -o ./_out/job-runner ./internal/cmd-line-tools/job-runner
#	GOOS=linux GOARCH=amd64 go build -v -o ./_out/importtest-linux ./internal/cmdline-tools/import-integration-test
#	GOOS=linux GOARCH=amd64 go build -v -o ./_out/integrationtest-linux ./internal/cmdline-tools/api-integration-test

# build-mac:
# 	GOPRIVATE=github.com/pixlise GOOS=darwin GOARCH=amd64 go build -ldflags "-X services.ApiVersion=${BUILD_VERSION} -X services.GitHash=${GITHUB_SHA}" -v -o ./_out/pixlise-api-mac ./internal/api
# 	GOPRIVATE=github.com/pixlise GOOS=darwin GOARCH=amd64 go build -v -o ./_out/jobupdater-mac ./internal/lambdas/quant-job-updater

# build-windows:
# 	GOPRIVATE=github.com/pixlise GOOS=windows GOARCH=amd64 go build -ldflags "-X services.ApiVersion=${BUILD_VERSION} -X services.GitHash=${GITHUB_SHA}" -v -o ./_out/pixlise-api-windows ./internal/api

clean: ## Remove previous build
	@rm -f $(PROJECT_NAME)

help: ## Display this help screen
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'
