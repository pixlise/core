#!/bin/bash

# mongoimport --host localhost --db userdatabase-prodCOPY --collection users /dbseed/migration-source/users.json
mongorestore dbseed/migrated/
