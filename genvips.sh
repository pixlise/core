#!/bin/bash

set -e

go install github.com/cshum/vipsgen/cmd/vipsgen@latest
vipsgen -out ./vips
