#!/bin/bash

set -e

rm -rf ./generated-protos
mkdir -p ./generated-protos

# Back up old ones if they exist
# if [ -e ./generated/experiment/experiment.pb.go ]
#  then cp ./generated/experiment/experiment.pb.go /tmp/
# fi
# if [ -e ./generated/quantification/quantification.pb.go ]
#  then cp ./generated/quantification/quantification.pb.go /tmp/
# fi
# if [ -e ./generated/diffraction/diffraction.pb.go ]
#  then cp ./generated/diffraction/diffraction.pb.go /tmp/
# fi

# NOTE: Make sure protoc-gen-go is in path so protoc works
# Since Go 1.17, thi should get it ready:
# go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
protoc --go_out=./generated-protos/ ./data-formats/file-formats/experiment.proto
protoc --go_out=./generated-protos/ ./data-formats/file-formats/quantification.proto
protoc --go_out=./generated-protos/ ./data-formats/file-formats/diffraction.proto

# Compare, if they changed, fail this script, because we're now in the habit of checking
# in the generated code... in dev builds we should be failing if it differs!

# if [ "$1" = "checkgen" ]; then
#     echo "Checking generated code matches existing..."
#     if cmp --silent ./generated/experiment/experiment.pb.go /tmp/experiment.pb.go; then
#         echo "Experiment generated file matches checked in version"
#     else
#         echo "Experiment generated file differs"
#         exit 1
#     fi
#     if cmp --silent ./generated/quantification/quantification.pb.go /tmp/quantification.pb.go; then
#         echo "Quantification generated file matches checked in version"
#     else
#         echo "Quantification generated file differs"
#         exit 1
#     fi
#     if cmp --silent ./generated/diffraction/diffraction.pb.go /tmp/diffraction.pb.go; then
#         echo "Diffraction generated file matches checked in version"
#     else
#         echo "Diffraction generated file differs"
#         exit 1
#     fi
# fi

protoc --go_out=./generated-protos/ --proto_path=./data-formats/api-messages/ ./data-formats/api-messages/*.proto
go run data-formats/codegen/main.go -protoPath ./data-formats/api-messages/ -goOutPath ./api/ws/

protoc-go-inject-tag -remove_tag_comment -input="./generated-protos/*.pb.go"
