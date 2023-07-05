#!/bin/bash

mongoimport mongodb://localhost --db expressions-prodCOPY --collection expressions /dbseed/expressions.json
mongoimport mongodb://localhost --db expressions-prodCOPY --collection modules /dbseed/modules.json
mongoimport mongodb://localhost --db expressions-prodCOPY --collection moduleVersions /dbseed/moduleVersions.json
mongoimport mongodb://localhost --db userdatabase-prodCOPY --collection users /dbseed/users.json
mongorestore dbseed/migrated/
