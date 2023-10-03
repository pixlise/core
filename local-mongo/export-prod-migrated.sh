mongoexport --db pixlise-prodMigrated --collection diffractionDetectedPeakStatuses --out diffractionDetectedPeakStatuses.json 
mongoexport --db pixlise-prodMigrated --collection diffractionManualPeaks --out diffractionManualPeaks.json 
mongoexport --db pixlise-prodMigrated --collection elementSets --out elementSets.json 
mongoexport --db pixlise-prodMigrated --collection expressionGroups --out expressionGroups.json 
mongoexport --db pixlise-prodMigrated --collection expressions --out expressions.json 
mongoexport --db pixlise-prodMigrated --collection modules --out modules.json 
mongoexport --db pixlise-prodMigrated --collection moduleVersions --out moduleVersions.json 
mongoexport --db pixlise-prodMigrated --collection imageBeamLocations --out imageBeamLocations.json 
mongoexport --db pixlise-prodMigrated --collection images --out images.json 
mongoexport --db pixlise-prodMigrated --collection mistROIs --out mistROIs.json 
mongoexport --db pixlise-prodMigrated --collection modules --out modules.json 
mongoexport --db pixlise-prodMigrated --collection moduleVersions --out moduleVersions.json 
mongoexport --db pixlise-prodMigrated --collection piquantVersion --out piquantVersion.json 
mongoexport --db pixlise-prodMigrated --collection quantifications --out quantifications.json 
mongoexport --db pixlise-prodMigrated --collection quantificationZStacks --out quantificationZStacks.json 
mongoexport --db pixlise-prodMigrated --collection regionsOfInterest --out regionsOfInterest.json 
mongoexport --db pixlise-prodMigrated --collection scans --out scans.json 
mongoexport --db pixlise-prodMigrated --collection tags --out tags.json 
mongoexport --db pixlise-prodMigrated --collection userGroups --out userGroups.json 
mongoexport --db pixlise-prodMigrated --collection viewStates --out viewStates.json 
mongoexport --db pixlise-prodMigrated --collection ownership --out ownership.json 
mongoexport --db pixlise-prodMigrated --collection users --out users.json 
mongoexport --db pixlise-prodMigrated --collection detectorConfigs --out detectorConfigs.json 


mongoexport --db expressions-prodCOPY --collection expressions --out expressions.json 
mongoexport --db expressions-prodCOPY --collection modules --out modules.json 
mongoexport --db expressions-prodCOPY --collection moduleVersions --out moduleVersions.json 