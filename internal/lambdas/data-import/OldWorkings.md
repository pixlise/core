What it does:

Lambda handler:
- Sets global vars for paths
- if record.EventSource == "aws:s3": processS3(record)
  else if record.EventSource == "aws:sns": processSns(record)


Process S3:
    if record.S3.Object.Key contains "dataset-addons" {
        Gets dataset ID from 2nd part of path /dataset-addons/<id>/...
        executeReprocess(datasetID, DATASETS_BUCKET)
    } else {
        Seems to assume record.S3.Object is a directory?
        executePipeline(record.S3.Bucket.Name, record.S3.Object.Key, DATASETS_BUCKET)
        Deletes record.S3.Object
    }


Process SNS:
    if record.SNS.Message starts with "{"datasetaddons":" {
        Gets dataset ID from 2nd part of path snsMsg.Key.Dir
        executeReprocess(datasetID, DATASETS_BUCKET)
    } else if record.SNS.Message.Records[0].EventSource == "aws:s3" {
        keys = checkExisting(DATASETS_BUCKET, record.SNS.Message.Records[0].S3.Object.Key)
        if len(keys) > 0 {
            File exists, don't reprocess
        } else {
            executePipeline(record.SNS.Message.Records[0].S3.Bucket.Name, record.SNS.Message.Records[0].S3.Object.Key, DATASETS_BUCKET)
        }
    } else if record.SNS.Message starts with "datasource:" {
        DO NOTHING - comment though says "run execution"??
    } else {
        IS THIS WRONG BECAUSE ITS NOT PASSING A DATASET ID????
        executeReprocess(record.SNS.Message, DATASETS_BUCKET)
    }



executePipeline(datasourcePath, sourcebucket, targetbucket):
    Create but ensure empty: localUnzipPath
    make importers
    fetchRanges -> localRangesCSVPath
    Copy datasourcePath (a zip file?) from sourcebucket to targetbucket/Datasets/archive/
    Split datasourcePath by -, grab first part, that's the Dataset ID
    files = checkExistingArchive([], datasetID, may return a updateExisting flag??)
    if len(files) > 0 {
        updateExisting=true
    }
    downloadDirectoryZip(sourcebucket, datasourcePath) -> writes to localInputPath temp zip file -> unzips to localUnzipPath
    Loops through "allthefiles" which seems to be [] at this point, if path contains zip or ends in .zip: utils.UnzipDirectory(path -> localUnzipPath)
        NOTE: this also overwrites datasourcePath with the localUnzipPath???

    downloadExtraFiles(datasetID) - "Get the extra manual stuff"

    processFiles(localUnzipPath, datasourcePath, importers, updateExisting, targetbucket)




executeReprocess(datasetID, datasourcePath, targetbucket):
    fetchRanges -> localRangesCSVPath
    make importers
    allthefiles = checkExistingArchive([], datasetID, may return a updateExisting flag??) (WEIRD, append() with 1 parameter??)
    Loops through "allthefiles", if path contains zip or ends in .zip: utils.UnzipDirectory(path -> localUnzipPath)
        NOTE: this also overwrites datasourcePath with the localUnzipPath???

    downloadExtraFiles(datasetID) - "Get the extra manual stuff"

    processFiles(localUnzipPath, datasourcePath or localUnzipPath depending on loop, importers, updateExisting, targetbucket)



processFiles(inpath, name, importers, updateExisting, targetbucket):
    computeName()
    flag.Parse() ????
    data, contextImgPath, err = importers["pixl-fm"].Import(inpath, localRangesCSVPath)
    apply dataset ID override if needed
    Customise detector if needed (based on SOL having characters in there, eg PIXL-EM?) vs PIXL
    Customise dataset group (based on detector)
    outPath = name.Outpath+data.DatasetID
    importAutoQuickLook(outPath)

    output.PIXLISEDataSaver.Save(data, contextImgPath, outPath, createTimeUNIX)

    createPeakDiffractionDB(outPath/dataset.bin, outpath/diffraction.bin)

    copyAdditionalDirectories(outPath)
    uploadDirectoryToAllEnvironments(outPath, data.DatasetID, [targetbucket])

    trigger notifications


checkExistingArchive(allthefiles, datasetID, *updateExisting):
    prefix = ((datasetID.split(".")[0]).split("-"))[0]      <-- In actual fact it's being passed the RTT... but maybe was previously
                                                                expecting file names like this maybe? 161677829-12-06-2022-06-41-00.zip
    paths = checkExisting(DATASETS_BUCKET, prefix)          <-- Reads from /Datasets/archive
    for p in paths:
        *updateExisting = true
        downloadDirectoryZip(DATASETS_BUCKET, p)

    return allthefiles         <-- THIS IS JUST PASSED THROUGH???


checkExisting(bucket, prefix):
    files = list all files starting with: s3://bucket/Datasets/archive/prefix

    map[timestamp]file
    keys = []string

    for f in files:
        Split file name to extract timestamp (eg 161677829-12-06-2022-06-41-00.zip)
        Store in map[timestamp]=f
    Extract timestamps from map
    Sort timestamps
    Store filenames in keys ordered by timestamp, reading from map
    Print out filenames
    return filenames


downloadDirectoryZip(bucket, path):
    Download file referenced by path to localInputPath/zip temp file
    Unzip file to localUnzipPath
    Delete file
    Return filename