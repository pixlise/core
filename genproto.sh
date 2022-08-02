#!/bin/bash

set -e

mkdir -p ./generated-protos

# Back up old ones if they exist
if [ -e ./generated/experiment/experiment.pb.go ]
 then cp ./generated/experiment/experiment.pb.go /tmp/
fi
if [ -e ./generated/quantification/quantification.pb.go ]
 then cp ./generated/quantification/quantification.pb.go /tmp/
fi
if [ -e ./generated/diffraction/diffraction.pb.go ]
 then cp ./generated/diffraction/diffraction.pb.go /tmp/
fi

# NOTE: Make sure protoc-gen-go is in path so protoc works
protoc --go_out=./generated-protos/ ./data-formats/experiment.proto
protoc --go_out=./generated-protos/ ./data-formats/quantification.proto
protoc --go_out=./generated-protos/ ./data-formats/diffraction.proto

# Compare, if they changed, fail this script, because we're now in the habit of checking
# in the generated code... in dev builds we should be failing if it differs!

if [ "$1" = "checkgen" ]; then
    echo "Checking generated code matches existing..."
    if cmp --silent ./generated/experiment/experiment.pb.go /tmp/experiment.pb.go; then
        echo "Experiment generated file matches checked in version"
    else
        echo "Experiment generated file differs"
        exit 1
    fi
    if cmp --silent ./generated/quantification/quantification.pb.go /tmp/quantification.pb.go; then
        echo "Quantification generated file matches checked in version"
    else
        echo "Quantification generated file differs"
        exit 1
    fi
    if cmp --silent ./generated/diffraction/diffraction.pb.go /tmp/diffraction.pb.go; then
        echo "Diffraction generated file matches checked in version"
    else
        echo "Diffraction generated file differs"
        exit 1
    fi
fi
