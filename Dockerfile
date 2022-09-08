# Licensed to NASA JPL under one or more contributor
# license agreements. See the NOTICE file distributed with
# this work for additional information regarding copyright
# ownership. NASA JPL licenses this file to you under
# the Apache License, Version 2.0 (the "License"); you may
# not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http:#www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing,
# software distributed under the License is distributed on an
# "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
# KIND, either express or implied.  See the License for the
# specific language governing permissions and limitations
# under the License.
FROM golang:1.18-alpine
ARG VERSION
ARG GITHUB_SHA

RUN apk --no-cache add ca-certificates libc6-compat wget make bash

COPY . /build

RUN cd /build && BUILD_VERSION=${VERSION} GITHUB_SHA=${GITHUB_SHA} make build-linux

FROM alpine:latest
RUN apk --no-cache add ca-certificates libc6-compat wget
WORKDIR /root
# Copy the Pre-built binary file from the previous stage

COPY --from=0 /build/_out/pixlise-api-linux ./
#COPY ./_out/pixlise-api-linux ./

RUN chmod +x ./pixlise-api-linux && wget https://s3.amazonaws.com/rds-downloads/rds-combined-ca-bundle.pem
# Expose port 8080 to the outside world
EXPOSE 8080

# Command to run the executable
ENTRYPOINT ["./pixlise-api-linux"]
