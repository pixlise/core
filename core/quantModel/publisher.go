package quantModel

import (
	"errors"
	"fmt"
	"path"
	"strings"
	"time"

	"github.com/pixlise/core/v3/api/filepaths"
	datasetModel "github.com/pixlise/core/v3/core/dataset"
	"github.com/pixlise/core/v3/core/fileaccess"
	"github.com/pixlise/core/v3/core/logger"
	"github.com/pixlise/core/v3/core/notifications"
	"github.com/pixlise/core/v3/core/pixlUser"
	gdsfilename "github.com/pixlise/core/v3/data-import/gds-filename"
)

type PublisherConfig struct {
	KubernetesLocation      string
	QuantDestinationPackage string
	QuantObjectType         string
	PosterImage             string
	DatasetsBucket          string
	EnvironmentName         string
	Kubeconfig              string
	UsersBucket             string
}

type ProductSet struct {
	OcsPath         string
	SourceBucket    string
	SourcePrefix    string
	DatasetID       string
	JobID           string
	PqrFileName     string
	PqrMetaFileName string
	PqpFileName     string
	PqpMetaFileName string
	// The following lines are related to the ROI ID Map file that is targeted for inclusion but not yet ready
	// RimFileName     string
	// RimMetaFileName string
}

/*
{
  datasets: [
    {
	  dataset-id: blah
      job-id: blah
      publications: [
        {
          publisher:
		  version:
          timestamp:
        }, ...
      ]
    }, ...
  ]
}
*/

// Publication keeps details about a particular publication
type Publication struct {
	Version         int       `json:"version"`
	PublicationTime time.Time `json:"timestamp"`
	Publisher       string    `json:"publisher"`
}

// PublicationRecord keeps tracks all the publications for a quantification
type PublicationRecord struct {
	DatasetID    string        `json:"dataset-id"`
	JobID        string        `json:"job-id"`
	Publications []Publication `json:"publications"`
}

// Publications keeps a list of publications for all Datasets
type Publications struct {
	Datasets []PublicationRecord `json:"datasets"`
}

// OcsMetaData metadata file that provides information to OCS about published Datasets
type OcsMetaData struct {
	Description string `json:"description"`
}

// DetectorCode Indicates the detector used in a Quantification; A/B or combined
type DetectorCode string

const (
	Combined DetectorCode = "C" // Combined uses the combined detector for quantification
	Separate              = "S" // Separate uses the separate A/B detectors for quantification
)

// ProductType Indicates the Product Type to publish to OCS
type ProductType string

const (
	PiQuantResults ProductType = "PQR" // PiQuantResults PQR Products contain the PiQuant Quantification Results
	PiQuantParams  ProductType = "PQP" // PiQuantParams PQP Products contain the PiQuant Runtime Parameters
	// RoiMap         ProductType = "RIM" // RoiMap RIM Products contain additional info for each ROI present in the Quantification
)

// OcsPath directs publishing to ODS
const OcsPath string = "/ods/surface/sol/XXXXX/soas/rdr/pixl/PQA"

// OcsStagingPath path in s3 bucket where files are staged before publishing to ODS
const OcsStagingPath string = "Publish/Staging"

// PublicationsPath path in s3 bucket where the list of publication records are kept
const PublicationsPath string = "Publish/publications.json"

func PublishQuant(fs fileaccess.FileAccess, config PublisherConfig, creator pixlUser.UserInfo, log logger.ILogger, datasetID, jobID string, notifications notifications.NotificationManager) error {
	//  3. Lookup QuantID's for dataset (list of camSpecific codes) in DataBucket/Publish/publications.json
	//  4. if already published, log that fact
	//  5. Generate OCS-compliant productName
	// generateFilename(quantname, currVersionNumber)
	//  7. Save combined.csv, params.json to above productName{.csv,.json}
	// stageToS3()
	//  8. Generate productName.csv.met and productName.json.met
	// stageMetFiles()
	//  9. Generate ocs destination path
	// ocsPath := generateDestinationPath()
	//  10. Initiate ocs-poster pod
	// triggerQuantPublish()
	//  11. Await success code from ocs-poster pod
	//  12. Send success/failure notification
	//  13. Update JobData/publications.json
	log.Infof("Publishing quantification for dataset-job: %s-%s", datasetID, jobID)
	currVersionNumber, err := checkCurrentlyPublishedQuantVersion(fs, config.DatasetsBucket, datasetID)
	if err != nil {
		return err
	}
	log.Infof("Currently published quant version: %v; incrementing for publication", currVersionNumber)
	ocsProducts, err := makeQuantProducts(fs, config.UsersBucket, config.DatasetsBucket, datasetID, jobID, currVersionNumber+1)
	if err != nil {
		return err
	}
	err = stageMetFiles(fs, config.DatasetsBucket, ocsProducts)
	if err != nil {
		return err
	}
	err = stageQuant(fs, config.DatasetsBucket, datasetID, jobID, ocsProducts)
	if err != nil {
		return err
	}
	err = triggerOCSPoster(config, log, creator, ocsProducts, datasetID, config.EnvironmentName)
	if err != nil {
		return err
	}
	err = sendNotifications(notifications, jobID, creator)
	if err != nil {
		return err
	}
	return savePublicationRecord(fs, config.DatasetsBucket, datasetID, jobID, creator.Name)
}

// sendNotifications - Dispatch notifications to users via a notifications stack.
func sendNotifications(notifications notifications.NotificationManager, quantname string, creator pixlUser.UserInfo) error {
	template := make(map[string]interface{})
	template["quantname"] = quantname
	template["subject"] = fmt.Sprintf("New quantification(%v) has been published.", quantname)
	users := []string{"auth0|" + creator.UserID}
	return notifications.SendAll("quant-published", template, users, false)
}

// checkCurrentlyPublishedQuantVersion - Check the latest version of a quant published to datadrive
func checkCurrentlyPublishedQuantVersion(fs fileaccess.FileAccess, dataBucket string, datasetID string) (int, error) {
	var currVersion int

	var publications Publications
	err := fs.ReadJSON(dataBucket, PublicationsPath, &publications, false)
	if err != nil {
		return currVersion, err
	}

	currVersion = getQuantVersion(publications, datasetID)
	return currVersion, nil
}

// getQuantVersion - Get the greatest version number from the list of publications for a particular dataset;
//
//	return 0 if datasetID is not found in the publications list
func getQuantVersion(publications Publications, datasetID string) int {
	var currVersion int
	for _, publicationSet := range publications.Datasets {
		if publicationSet.DatasetID == datasetID {
			for _, publication := range publicationSet.Publications {
				if publication.Version > currVersion {
					currVersion = publication.Version
				}
			}
			return currVersion
		}
	}
	return currVersion
}

// savePublicationRecord - Save the publication metadata back to S3
func savePublicationRecord(fs fileaccess.FileAccess, dataBucket string, datasetID string, jobId string, publisher string) error {
	// Ensure that publications for an existing dataset are appended to publications list so that we don't
	//   create multiple records with the same dataset-id
	var publications Publications
	err := fs.ReadJSON(dataBucket, PublicationsPath, &publications, false)
	if err != nil {
		return err
	}
	currVersion := getQuantVersion(publications, datasetID)

	appended := false
	for i := 0; i < len(publications.Datasets); i++ {
		ds := &publications.Datasets[i]
		if ds.DatasetID == datasetID {
			ds.Publications = append(ds.Publications, Publication{
				Version:         currVersion + 1,
				PublicationTime: time.Now(),
				Publisher:       publisher,
			})
			appended = true
			break
		}
	}

	if !appended {
		publications.Datasets = append(publications.Datasets, PublicationRecord{
			DatasetID: datasetID,
			JobID:     jobId,
			Publications: []Publication{
				{
					Version:         1,
					PublicationTime: time.Now(),
					Publisher:       publisher,
				},
			},
		})
	}

	return fs.WriteJSON(dataBucket, PublicationsPath, publications)
}

// stageOcsData - Stage OCS data to S3 prior to publishing
func stageOcsData(fs fileaccess.FileAccess, databucket string, object interface{}, datasetID string, filename string) (error, string) {
	datasetStagingPath := path.Join(OcsStagingPath, datasetID)
	s3path := path.Join(datasetStagingPath, filename)
	err := fs.WriteJSON(databucket, s3path, object)
	return err, s3path
}

// stageQuant - Stage Quant data to S3 prior to publishing
func stageQuant(fs fileaccess.FileAccess, bucket string, datasetID string, jobID string, products ProductSet) error {
	var err error

	datasetStagingPath := path.Join(OcsStagingPath, datasetID)
	// We can assume that any Quant being published will have already been "shared" and available in the ShareUserId directory

	// Stage PiQuant Results
	quantSourcePath := filepaths.GetSharedQuantPath(datasetID, filepaths.MakeQuantCSVFileName(jobID))
	quantDestPath := path.Join(datasetStagingPath, products.PqrFileName)
	err = fs.CopyObject(bucket, quantSourcePath, bucket, quantDestPath)
	if err != nil {
		return err
	}

	// Stage PiQuant Params
	summaryFilePath := filepaths.GetSharedQuantPath(datasetID, filepaths.MakeQuantSummaryFileName(jobID))
	summaryDestPath := path.Join(datasetStagingPath, products.PqpFileName)
	err = fs.CopyObject(bucket, summaryFilePath, bucket, summaryDestPath)
	if err != nil {
		return err
	}

	// Stage ROI Map for Quantification

	return nil
}

// stageMetFiles - Stage Met files to S3 prior to publishing.
func stageMetFiles(fs fileaccess.FileAccess, bucket string, products ProductSet) error {
	var err error
	// TODO: Fill the descriptions below based on config/template kept elsewhere
	pqrMeta := OcsMetaData{"PiQuant Results File"}
	pqpMeta := OcsMetaData{"PiQuant Runtime Parameters File"}
	// rimMeta := OcsMetaData{"ROI Map gives information about each of the Regions of Interest included in the Quantification"}

	err, _ = stageOcsData(fs, bucket, pqrMeta, products.DatasetID, products.PqrMetaFileName)
	if err != nil {
		return err
	}
	err, _ = stageOcsData(fs, bucket, pqpMeta, products.DatasetID, products.PqpMetaFileName)
	if err != nil {
		return err
	}
	// err, _ = stageOcsData(fs, bucket, rimMeta, products.DatasetID, products.RimMetaFileName)
	// if err != nil {
	// 	return err
	// }
	return nil
}

// makeQuantProducts - Make the various quant products relating to the files about to be published.
func makeQuantProducts(fs fileaccess.FileAccess, usersBucket string, datasetsBucket string, datasetID string, jobID string, version int) (ProductSet, error) {
	var err error
	var products ProductSet

	// We can assume that any Quant being published will have already been "shared" and available in the ShareUserId directory
	jobSummary, err := GetJobSummary(fs, usersBucket, pixlUser.ShareUserID, datasetID, jobID)
	if err != nil {
		return products, err
	}

	datasetSummary, err := datasetModel.ReadDataSetSummary(fs, datasetsBucket, datasetID)
	if err != nil {
		return products, err
	}

	// To generate product names for OCS, we start with the context image file for this dataset
	// ex: PCW_0208_0685431075_000RAD_N007183607687424400710LUJ01.png
	// This filename contains much information that is also valid for the published Quantification files;
	//   we parse it to a struct to tweak the parts relevant to PiQuant publications
	contextImageName := datasetSummary.ContextImage
	contextImageFileMeta, err := gdsfilename.ParseFileName(contextImageName)
	if err != nil {
		return products, err
	}

	// Add the version number ("01" for new publication, incremented for republished files)
	if version > 99 {
		return products, errors.New("cannot yet publish versions greater than 99")
	}
	contextImageFileMeta.SetVersionStr(fmt.Sprintf("%02d", version))
	contextImageFileMeta.SetInstrumentType("PE")
	// Set the quantCode based on the job summary params and note it in the repurposed ColourFilter filename component
	var quantCode DetectorCode
	if jobSummary.Params.QuantMode == quantModeCombinedAB {
		quantCode = Combined
	} else {
		quantCode = Separate
	}
	contextImageFileMeta.SetColourFilter(string(quantCode))

	// We will need extract the sol from the filename for use in the OCS path (returns string with "%05d" format)
	sol, err := contextImageFileMeta.SOL()
	if err != nil {
		return products, err
	}

	if len(sol) == 4 {
		sol = "0" + sol
	}
	quantResults := contextImageFileMeta
	quantResults.SetProdType(string(PiQuantResults))
	quantParams := contextImageFileMeta
	quantParams.SetProdType(string(PiQuantParams))
	// rim := contextImageFileMeta
	// rim.SetProdType("RIM")

	products = ProductSet{
		OcsPath:         strings.Replace(OcsPath, "XXXXX", sol, 1),
		SourceBucket:    usersBucket,
		SourcePrefix:    OcsStagingPath,
		DatasetID:       datasetID,
		JobID:           jobID,
		PqrFileName:     fmt.Sprintf("%s.CSV", quantResults.ToString()),
		PqrMetaFileName: fmt.Sprintf("%s.CSV.MET", quantResults.ToString()),
		PqpFileName:     fmt.Sprintf("%s.JSON", quantParams.ToString()),
		PqpMetaFileName: fmt.Sprintf("%s.JSON.MET", quantParams.ToString()),
		// RimFileName:     fmt.Sprintf("%s.JSON", rim.ToString()),
		// RimMetaFileName: fmt.Sprintf("%s.JSON.MET", rim.ToString()),
	}
	return products, nil
}

// triggerOCSPoster - Trigger a run of the OCS poster docker container with the associated metadata.
func triggerOCSPoster(config PublisherConfig, log logger.ILogger, creator pixlUser.UserInfo, products ProductSet, dataset string, kenv string) error {
	//k := kubernetes.KubeHelper{
	//	Kubeconfig: config.Kubeconfig,
	//}
	//
	//k.Bootstrap(config.KubernetesLocation, log)
	//filenames := fmt.Sprintf("%v/%v/%s,%v/%v/%s", OcsStagingPath, dataset, products.PqpFileName, OcsStagingPath, dataset, products.PqrFileName)
	//env := make(map[string]string)
	//env["venue"] = "sstage"
	//env["credss_username"] = "m20-sstage-pixlise"
	//env["credss_appaccount"] = "true"
	//
	//_, err := k.RunPod(nil, ocs.GeneratePosterPodCmd(filenames, products.SourceBucket, products.OcsPath, config.QuantDestinationPackage, config.QuantObjectType), env, volumes, volumemounts, config.PosterImage,
	//	"api", generatePodNamePrefix(products.JobID), generatePodLabels(products.JobID, products.DatasetID, kenv), creator, log, false)
	//if err != nil {
	//	return err
	//}
	return nil
}
