#!/bin/bash

if [ -z "$1" ]; then
    echo "Syncing all db files..."
    aws s3 sync s3://pixlise-backup/DB/pixlise-prodv4/ ./db-restore/
else
    echo "Syncing all db files excluding '$1'..."
    aws s3 sync --exclude $1.* s3://pixlise-backup/DB/pixlise-prodv4/ ./db-restore/
    rm ./db-restore/$1.*
fi


# Start Mongo
# Initiate a replica set (rs.initiate())
# Call import.sh to reload last saved mongo dump
docker run --rm -d -v /$PWD/db-restore:/db-restore -p 27017:27017 -h $(hostname) --name mongo-test mongo:4.0.28 --replSet=test && sleep 4 && docker exec mongo-test mongo --eval "rs.initiate();" && sleep 2 && docker exec mongo-test mongorestore --gzip --db "pixlise-localdev" db-restore/
