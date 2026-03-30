#!/bin/bash

set -e

go install github.com/cshum/vipsgen/cmd/vipsgen@v1.3.8
vipsgen -out ./vips
