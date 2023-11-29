#!/bin/bash

# Start Mongo
# Initiate a replica set (rs.initiate())
# Call import.sh to reload last saved mongo dump
docker run --rm -d -v /$PWD/dbseed:/dbseed -p 27017:27017 -h $(hostname) --name mongo-test mongo:4.0.28 --replSet=test && sleep 4 && docker exec mongo-test mongo --eval "rs.initiate();" && sleep 2 && docker exec mongo-test dbseed/import.sh
