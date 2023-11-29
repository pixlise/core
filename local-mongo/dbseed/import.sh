#!/bin/bash

mongoimport --host localhost --db expressions-prodCOPY --collection expressions /dbseed/migration-source/expressions.json
mongoimport --host localhost --db expressions-prodCOPY --collection modules /dbseed/migration-source/modules.json
mongoimport --host localhost --db expressions-prodCOPY --collection moduleVersions /dbseed/migration-source/moduleVersions.json
mongoimport --host localhost --db userdatabase-prodCOPY --collection users /dbseed/migration-source/users.json
mongorestore dbseed/migrated/
