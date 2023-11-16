#!/bin/bash

# --rm
docker run --rm -d -v /$PWD/dbseed:/dbseed -p 27017:27017 -h $(hostname) --name mongo-test mongo:5 --replSet=test && sleep 4 && docker exec mongo-test mongo --eval "rs.initiate();" && sleep 2 && docker exec mongo-test dbseed/import.sh
