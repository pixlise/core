FROM alpine:latest

WORKDIR /root

# Copy the Pre-built binary file from the previous stage
COPY ./mongo-backup ./

RUN chmod +x ./mongo-backup

# Command to run the executable
ENTRYPOINT ["./mongo-backup"]
