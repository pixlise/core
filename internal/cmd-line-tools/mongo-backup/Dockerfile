FROM golang:1.24-alpine AS builder

ARG VERSION
ARG GITHUB_SHA

RUN apk add --no-cache libc6-compat bash build-base

COPY . /build
ENV CGO_ENABLED=1
ENV BUILD_VERSION=${VERSION}
ENV GITHUB_SHA=${GITHUB_SHA}

RUN mkdir -p _out
RUN cd /build && GOOS=linux GOARCH=amd64 go build -o ./_out/mongo-backup .



FROM alpine:latest

WORKDIR /root

# Copy the Pre-built binary file from the previous stage
COPY --from=builder /build/_out/mongo-backup ./

RUN chmod +x ./mongo-backup

# Command to run the executable
ENTRYPOINT ["./mongo-backup"]
