# Data migration tool for PIXLISE v3 to v4

Data storage changed significantly between v3 and v4. Previously mostly relied on files in S3, while v4 relies heavily on Mongo DB and only stores large files in S3 (eg images, datasets, diffraction, quantifications). This tool can read the old files from S3 and generate a MongoDB and set of buckets that work with v4.

## Inputs required

This tool reads from existing S3 buckets parsing the paths of files it finds to determine relationships with other files, and interprets them as the v3 API did. They are then written to MongoDB in the new format, while files in S3 are copied to their new location in the new buckets. Note that root paths of files have changed so the v4 data can co-exist with v3 data in the same buckets but clearly that's not recommended!

V3 also stored users and expression/data-modules in 2 separate mongo DBs, and this tool will need access to those too.

Finally, Joel supplied some expression groups which were useful while testing, and this tool has hard-coded them to be written to the Mongo DB for v4 to pick up. 

## How to run migration tool
From VS code you'll need a new launch configuration. This allows setting some important command line parameters which are documented below. Note that not ALL datasets are blindly imported, as we had a few which failed over the years or were test datasets that needed to be ignored. The source and destination databases aren't named directly but are named by providing the environment name. For example, *srcEnvName=abc* becomes *expressions-abc* and *userdatabase-abc*, *destEnvName=def* becomes *pixlise-def*. This tool expects to find the following in the source databases:
*expressions-abc* should have collections: "expressions", "moduleVersions", "modules"
*userdatabase-abc* should have the collection: "users"

These we can copy by hand from the prod DocumentDB and imported into the locally running mongo DB (in Docker). The destination DB should also be pointing to the local mongo DB. 

### Command line arguments

Arguments are passed by the VS code launch configuration in as "args". Clearly, this can be run as a command line tool but given the ability to debug, the large number of parameters and niceness of it being reproducabile, running from VS code makes sense.

There is a list of dataset IDs to import (so we can exclude bad datasets), and also note that there are flags that can be passed to limit what kinds of data are being imported, for example in case you want to re-run only the importing of datasets and not quants.

This is an example VS code launch config used (with keys/bucket names removed):
```
{
    "name": "DB Migration",
    "type": "go",
    "request": "launch",
    "mode": "auto",
    "program": "internal/cmd-line-tools/v3-importer",
    //"showLog": true,
    "env": {},
    // NOTE: deliberately leaving out the following data sets:
    // 000000001 - keeps appearing
    // 060883460 - anomalous sol160
    // 088932868 - sol 290 quartier copy seems empty?
    // 178913797 - sol 514, seems empty?
    // 276365832 - Sol 790 ouzel falls 3 (empty?)
    // 299172357 - Sol 851 partial cal target
    // 302318087 - sol 861 anomaly
    // 376898049 - cal target, just downlinked
    // combined253
    // combined303

    "args": [
        "-sourceMongoSecret", "",
        "-destMongoSecret", "",
        "-dataBucket", "V3_DATA_BUCKET",
        "-destDataBucket", "V4_DATA_BUCKET",
        "-userContentBucket", "V3_USER_CONTENT_BUCKET",
        "-destUserContentBucket", "V4_USER_CONTENT_BUCKET",
        "-configBucket", "V3_CONFIG_BUCKET",
        "-srcEnvName", "prodCOPY",
        "-destEnvName", "prodImportJan2024",
        "-auth0Domain", "pixlise.au.auth0.com",
        "-auth0ClientId", "AUTH0CLIENT",
        "-auth0Secret", "AUTH0SECRET",
        "-limitToDatasetIDs", ",034931203,034931211,038863364,039256576,048300551,052822532,053281281,053871108,063111681,069927431,083624452,083624454,085393924,089063943,093258245,0x002A0201,1010101,1010102,101384711,104202753,110000453,122552837,123456789,123535879,130089473,130613765,130744834,154206725,155648517,161677829,167444997,168624645,170721793,176030213,176882177,189137412,194118145,194773505,197329413,198509061,199557637,200737285,204669441,207880709,208536069,208601602,212992517,214303237,214827527,220000453,222222001,222222002,222222003,222222004,222222005,222222006,222222007,222222008,222222009,222222011,222222012,222222013,222222014,222222015,222222016,222222017,222222018,222222019,230031877,243335685,247726593,261161477,262603269,271057409,272761349,273154565,275776005,276169217,284951045,296944133,297075201,297796101,299172359,299565573,301990405,303694341,306840069,308937221,309395969,311493121,313983493,322634245,323027463,327418369,327418372,327680513,327680516,328794625,329187841,330000453,363659777,371196417,38142470,440000453,550000453,590340,660000453,76481028,983561,987654322,Abraded_sample,Amelia_Albite2,BHVO2-G_2022_01_26,BHVO2-G_2022_05_03,crystal_geyser_degraded,crystal_geyser_intermediate,G060,G090,H1-B,HM_MSR_01,HM_MSR_04B_fixed,Lassen_Glass,Meyer_SS_SBU_V2,Meyer_SS_SBU_V3,Non-Abraded_sample,Pigeonite_Wo10X20,Pigeonite_Wo8X20,Pigeonite_Wo8X30,Pigeonite_Wo8X40,RUM_MSR_1_Scan_1,RUM_MSR_1_Scan_2_2000,test-baker-springs,test-em-cal-target,test-fintry,test-fm-5x11,test-fm-5x5-full,test-fm-5x5-tactical,test-ice-springs,test-kingscourt,test-kingscourt-ir,test-kingscourt-tactical,test-laguna,test-lone-volcano,test-los-angeles,test-NMSB12,test-NMSB18,test-NMSB31,test-NMSB64,test-NMSB70,test-NMSB72,test-NMSB73,test-rr4,test-STM1C,test-STM5,test-strelley-altered-basalt,test-strelley-left,test-strelley-right,test-sunstone-knoll,test-tabernacle-basalt,test-troughite,test-w1-polished,test-w1-polished-minimum,test-w1-polished-thresholded-a4-n6,test-w1-rough,YL_DAKP_rock_08_16_2022,YL_pink_green_rock_08_16_2022",
        "-migrateDatasetsEnabled=true",
        "-migrateROIsEnabled=true",
        "-migrateDiffractionPeaksEnabled=true",
        "-migrateRGBMixesEnabled=true",
        "-migrateTagsEnabled=true",
        "-migrateQuantsEnabled=true",
        "-migrateElementSetsEnabled=true",
        "-migrateZStacksEnabled=true",
        "-migrateExpressionsEnabled=true"
    ]
}
```

## On Completion
When the tool finishes, you will have files in the S3 buckets, and a complete DB. Convention for starting the local DB docker container when running the API are that you run `/local-mongo/start-db.sh`. This looks for files in `/local-mongo/dbseed/migrated/pixlise-<EnvName>` to read from. Note the similarity to the `destEnvName` parameter for the migration tool. It is recommended you do a `mongodump` of the DB that was created through this process, so the next time you restart the DB docker container it will reload the snapshot from the mongo dump instead of having to re-run this migration tool again.

To make this snapshot:
* Find docker container for mongo and get its id
* `docker exec -ti dockercontainerid bash`
* `cd /dbseed/migrated` within the container
* `mongodump -d databasename` within the container
* exit out of the container

Make sure the files generated by this (at time of writing, 42 collectionName.metadata.json and 42 collection.bson files) end up in `/local-mongo/dbseed/migrated/pixlise-<EnvName>` where the `EnvName` matches the one the API is configured to use.

## Running time
This tool uses many go routines to read files in parallel from S3 for speed. The conversion of all prod data as of late January 2024 takes about 1 hour.
