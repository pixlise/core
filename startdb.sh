#!/bin/bash

set -e

# Arguments: AWS profile, backup bucket path, local path, excluded collection name (optional)
# If we run without any parameters, we default to PIXLISE stuff
aws_profile="${1:-default}"
s3_path="${2:-pixlise-backup/DB/pixlise-prodv4/}"
local_path="${3:-./db-restore}"
db_name="${4:-pixlise-local-pixlise}"
db_reset="${5:-false}"
exclude_collection=$6
mongo_db_files="./mongo-db-${db_name}"

mongoContainer="mongo:8"

echo "Startup variables:"
echo "  aws_profile: ${aws_profile}"
echo "  s3_path:     ${s3_path}"
echo "  local_path:  ${local_path}"
echo "  db_name:     ${db_name}"
echo "  db_reset:    ${db_reset}"
echo "  exclude_collection: ${exclude_collection}"
echo "  mongo_db_files: ${mongo_db_files}"
echo ""

# If there is no existing DB file, perform a reset
if [ ! -f "${mongo_db_files}" ]; then
    echo "Forcing db reset because ${mongo_db_files} doesn't exist."
    db_reset="true"
fi

if [ "$db_reset" = "true" ]; then
    echo "Resetting dev environment..."

    # Delete DB file from last run
    if [ -f "${mongo_db_files}" ]; then
        echo "Deleting previous mongo DB data file: ${mongo_db_files}..."
        rm -rf ${mongo_db_files}
    fi

    # Sync down S3 bucket
    if [ -z "$exclude_collection" ]; then
        echo "Syncing all db files from s3://$s3_path ..."
        aws --profile $aws_profile s3 sync s3://$s3_path $local_path/
    else
        echo "Syncing all db files from s3://$s3_path excluding '$exclude_collection'..."
        aws --profile $aws_profile s3 sync --exclude $exclude_collection.* s3://$s3_path $local_path/
        rm -f $local_path/$exclude_collection.*
    fi

    if [ "$(ls -A $local_path)" ]; then
        echo "Starting docker MongoDB using DB restore..."
        docker run --rm -d -v /$PWD/$local_path:/db-restore -v "$mongo_db_files:/data" -p 27017:27017 -h $(hostname) --name mongo-test "$mongoContainer" --replSet=test && sleep 4 && docker exec mongo-test mongosh --eval "rs.initiate();" && sleep 2 && docker exec mongo-test mongorestore --gzip --db $db_name db-restore/
        exit 0
    fi
fi

echo "Starting docker MongoDB using previous volume..."
docker run --rm -d -v /$PWD/$local_path:/db-restore -v "$mongo_db_files:/data" -p 27017:27017 -h $(hostname) --name mongo-test "$mongoContainer" --replSet=test && sleep 4 && docker exec mongo-test mongosh --eval "rs.initiate();"
