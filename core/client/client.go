package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"time"

	"github.com/pixlise/core/v4/core/indexcompression"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// Based on: https://fluhus.github.io/snopher/
// and: https://medium.com/analytics-vidhya/running-go-code-from-python-a65b3ae34a2d

type ClientConfig struct {
	Host     string
	User     string
	Password string
}

type PIXLISEConfig struct {
	Auth0Domain   string `json:"auth0_domain"`
	Auth0Client   string `json:"auth0_client"`
	Auth0Audience string `json:"auth0_audience"`
	ApiUrl        string `json:"apiUrl"`
}

var configEnvVar = "PIXLISE_CLIENT_CONFIG"
var configFileName = ".pixlise-config.json" // We look for this file in home dir
var responseTimeoutSec = 10

type APIClient struct {
	socket *SocketConn

	// Local caching of things we need to build responses to things that are easier to digest on client-side
	// For example, we download meta labels, and pass back maps of string->value to client
	scanPMCToLocIdx              map[string]map[int]int
	scanLocIdxToPMC              map[string]map[int]int
	scanMetaLabels               map[string]*protos.ScanMetaLabelsAndTypesResp
	imageBeamVersions            map[string]map[string]*protos.ImageBeamLocationVersionsResp_AvailableVersions
	scanEntries                  map[string]*protos.ScanEntryResp
	scanEntryMeta                map[string]*protos.ScanEntryMetadataResp
	scanSpectra                  map[string]*protos.SpectrumResp
	quants                       map[string]*protos.QuantGetResp
	scanDiffractionStatuses      map[string]*protos.DiffractionPeakStatusListResp
	scanDiffractionDetected      map[string]*protos.DetectedDiffractionPeaksResp
	scanBulkSumEnergyCalibration map[string]*protos.ClientEnergyCalibration
	scanUserEnergyCalibration    map[string]*protos.ClientEnergyCalibration
}

// Authenticates using one of several methods:
// - First it looks for the environment variable, if found, it uses that, but if errors out we continue...
// - Second it looks for the config file in pre-defined path. If found, uses that.
func Authenticate() (*APIClient, error) {
	configStr := ""
	source := ""

	configPath, err := os.UserHomeDir()
	if err == nil {
		configPath = filepath.Join(configPath, ".pixlise-config.json") // "$HOME/.pixlise-config.json"

		_, err := os.Stat(configPath)
		if err == nil {
			// File seems to exist, try to read it
			cfgBytes, err := os.ReadFile(configPath)
			if err != nil {
				return nil, fmt.Errorf("failed to read config %v. Error: %v", configPath, err)
			}

			configStr = string(cfgBytes)
			source = configPath
		}
	} else {
		fmt.Printf("Failed to get user home directory to find pixlise config. Error: %v", err)
	}

	// If we haven't read anything useful yet, lets try read the environment variable
	if len(configStr) <= 0 {
		configStr = os.Getenv(configEnvVar)
		if len(configStr) > 0 {
			source = configEnvVar
		}
	}

	// If we haven't read anything useful, stop here
	if len(source) <= 0 || len(configStr) <= 0 {
		return nil, fmt.Errorf(`Couldn't read config file "%v" and no environment variable (%v) defined. Cannot authenticate. To configure, create the file or set the environment variable containing a JSON encoded structure with the following fields: "host", "user", "pass" where host is the URL of the PIXLISE webpage`, configPath, configEnvVar)
	}

	// Try to decode it
	cfg := ClientConfig{}
	err = json.Unmarshal([]byte(configStr), &cfg)
	if err != nil {
		return nil, fmt.Errorf("failed to read pixlise connection from \"%v\" config: %v", source, err)
	}

	// Now that we've got this, read the auth0 connection information from the PIXLISE instance
	url := cfg.Host + "/" + "pixlise-config.json"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to read auth config from \"%v\": %v", cfg.Host, err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to decode auth config from \"%v\": %v", cfg.Host, err)
	}

	pixliseConfig := PIXLISEConfig{}
	err = json.Unmarshal(body, &pixliseConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to read pixlise connection config: %v", err)
	}

	auth0Params := Auth0Info{
		ClientId: pixliseConfig.Auth0Client,
		Audience: pixliseConfig.Auth0Audience,
		Domain:   pixliseConfig.Auth0Domain,
	}
	connectParams := ConnectInfo{
		Host: pixliseConfig.ApiUrl,
		User: cfg.User,
		Pass: cfg.Password,
	}
	return AuthenticateWithAuth0Info(connectParams, auth0Params)
}

func AuthenticateWithAuth0Info(connectParams ConnectInfo, auth0Params Auth0Info) (*APIClient, error) {
	socket := &SocketConn{}
	err := socket.Connect(connectParams, auth0Params)

	if err != nil {
		return nil, err
	}

	return &APIClient{
		socket:                       socket,
		scanPMCToLocIdx:              map[string]map[int]int{},
		scanLocIdxToPMC:              map[string]map[int]int{},
		scanMetaLabels:               map[string]*protos.ScanMetaLabelsAndTypesResp{},
		imageBeamVersions:            map[string]map[string]*protos.ImageBeamLocationVersionsResp_AvailableVersions{},
		scanEntries:                  map[string]*protos.ScanEntryResp{},
		scanEntryMeta:                map[string]*protos.ScanEntryMetadataResp{},
		scanSpectra:                  map[string]*protos.SpectrumResp{},
		quants:                       map[string]*protos.QuantGetResp{},
		scanDiffractionStatuses:      map[string]*protos.DiffractionPeakStatusListResp{},
		scanDiffractionDetected:      map[string]*protos.DetectedDiffractionPeaksResp{},
		scanBulkSumEnergyCalibration: map[string]*protos.ClientEnergyCalibration{},
		scanUserEnergyCalibration:    map[string]*protos.ClientEnergyCalibration{},
	}, err
}

func (c *APIClient) sendMessageWaitResponse(msg *protos.WSMessage) ([]*protos.WSMessage, error) {
	if err := c.socket.SendMessage(msg); err != nil {
		return []*protos.WSMessage{}, err
	}

	resps := c.socket.WaitForMessages(1, time.Duration(responseTimeoutSec)*time.Second)
	if len(resps) != 1 {
		return []*protos.WSMessage{}, fmt.Errorf("Expected 1 response, got %v", len(resps))
	}

	if len(resps[0].ErrorText) > 0 {
		return []*protos.WSMessage{}, fmt.Errorf("Response status: %v. Error: %v", resps[0].Status, resps[0].ErrorText)
	}

	return resps, nil
}

func (c *APIClient) ensureScanSpectra(scanId string) error {
	req := &protos.SpectrumReq{ScanId: scanId, BulkSum: true, MaxValue: true /*, Entries: &protos.ScanEntryRange{}*/}
	msg := &protos.WSMessage{Contents: &protos.WSMessage_SpectrumReq{
		SpectrumReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	c.scanSpectra[scanId] = resps[0].GetSpectrumResp()
	return nil
}

func (c *APIClient) makeClientSpectrum(scanId string, spectrum *protos.Spectrum) (*protos.ClientSpectrum, error) {
	if err := c.ensureScanMetaLabels(scanId); err != nil {
		return nil, err
	}

	labels := c.scanMetaLabels[scanId]

	meta := map[string]*protos.ScanMetaDataItem{}
	for idx, item := range spectrum.Meta {
		// Find the string label
		label := labels.MetaLabels[idx]
		meta[label] = item
	}

	return &protos.ClientSpectrum{
		Detector: spectrum.Detector,
		Type:     spectrum.Type,
		Counts:   zeroRunDecode(spectrum.Counts),
		MaxCount: spectrum.MaxCount,
		Meta:     meta,
	}, nil
}

func (c *APIClient) GetScanSpectrum(scanId string, pmc int32, spectrumType protos.SpectrumType, detector string) (*protos.ClientSpectrum, error) {
	if err := c.ensureScanEntries(scanId); err != nil {
		return nil, err
	}
	if err := c.ensureScanSpectra(scanId); err != nil {
		return nil, err
	}

	// Read the specified spectrum
	spectraResp := c.scanSpectra[scanId]

	// If they're requesting bulk or max value, we don't look up by PMC
	spectraByDetector := []*protos.Spectrum{}
	if spectrumType == protos.SpectrumType_SPECTRUM_BULK {
		spectraByDetector = spectraResp.BulkSpectra
	} else if spectrumType == protos.SpectrumType_SPECTRUM_MAX {
		spectraByDetector = spectraResp.MaxSpectra
	}

	// Otherwise, find by PMC/type match
	if len(spectraByDetector) <= 0 {
		// Find the location index for the given PMC
		scanPMCToLocIdx := c.scanPMCToLocIdx[scanId]
		if locIdx, ok := scanPMCToLocIdx[int(pmc)]; ok && locIdx < len(spectraResp.SpectraPerLocation) {
			spectraByDetector = spectraResp.SpectraPerLocation[locIdx].Spectra
		}
	}

	for _, spectrum := range spectraByDetector {
		if spectrum.Detector == detector {
			return c.makeClientSpectrum(scanId, spectrum)
		}
	}

	return nil, fmt.Errorf("Failed to find spectrum for scan %v, pmc %v, spectrumType %v, detector %v", scanId, pmc, spectrumType, detector)
}

func (c *APIClient) ListScans(scanId string) (*protos.ScanListResp, error) {
	req := &protos.ScanListReq{}

	if len(scanId) > 0 {
		req.SearchFilters = map[string]string{"scanId": scanId}
	}
	msg := &protos.WSMessage{Contents: &protos.WSMessage_ScanListReq{
		ScanListReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}

	return resps[0].GetScanListResp(), nil
}

func (c *APIClient) ensureScanMetaLabels(scanId string) error {
	if _, ok := c.scanMetaLabels[scanId]; ok {
		return nil // already downloaded
	}

	req := &protos.ScanMetaLabelsAndTypesReq{ScanId: scanId}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_ScanMetaLabelsAndTypesReq{
		ScanMetaLabelsAndTypesReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	resp := resps[0].GetScanMetaLabelsAndTypesResp()
	c.scanMetaLabels[scanId] = resp
	return nil
}

func (c *APIClient) GetScanMetaList(scanId string) (*protos.ScanMetaLabelsAndTypesResp, error) {
	err := c.ensureScanMetaLabels(scanId)
	if err != nil {
		return nil, err
	}

	return c.scanMetaLabels[scanId], nil
}

func (c *APIClient) ensureScanMetaData(scanId string) error {
	if _, ok := c.scanEntryMeta[scanId]; ok {
		return nil // already downloaded
	}

	req := &protos.ScanEntryMetadataReq{ScanId: scanId}
	// Not filling out entries, we just get all

	msg := &protos.WSMessage{Contents: &protos.WSMessage_ScanEntryMetadataReq{
		ScanEntryMetadataReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	resp := resps[0].GetScanEntryMetadataResp()
	c.scanEntryMeta[scanId] = resp
	return nil
}

func (c *APIClient) GetScanMetaData(scanId string) (*protos.ScanEntryMetadataResp, error) {
	if err := c.ensureScanMetaData(scanId); err != nil {
		return nil, err
	}

	// Return the raw message. A better use is to get it by column, see GetScanEntryDataColumn
	return c.scanEntryMeta[scanId], nil
}

func (c *APIClient) GetScanEntryDataColumns(scanId string) (*protos.ClientStringList, error) {
	if err := c.ensureScanMetaData(scanId); err != nil {
		return nil, err
	}
	if err := c.ensureScanMetaLabels(scanId); err != nil {
		return nil, err
	}

	// Find all column names that are possible to query - not all PMCs will have all of these, but we
	// read from all PMCs to find the union of all possible names
	nameIdxs := map[int32]bool{}

	for _, m := range c.scanEntryMeta[scanId].Entries {
		keys := utils.GetMapKeys(m.Meta)
		for _, key := range keys {
			nameIdxs[key] = true
		}
	}

	// Finally, read the string names out
	names := []string{}
	scanLabels := c.scanMetaLabels[scanId].MetaLabels
	for i := range nameIdxs {
		names = append(names, scanLabels[i])
	}

	sort.Strings(names)

	return &protos.ClientStringList{Strings: names}, nil
}

func (c *APIClient) GetScanEntryDataColumn(scanId string, columnName string) (*protos.ClientMap, error) {
	if err := c.ensureScanMetaData(scanId); err != nil {
		return nil, err
	}
	if err := c.ensureScanMetaLabels(scanId); err != nil {
		return nil, err
	}
	if err := c.ensureScanEntries(scanId); err != nil {
		return nil, err
	}

	// Find the index of the label
	columnIdx := -1
	columnType := protos.ScanMetaDataType_MT_INT
	metaLabels := c.scanMetaLabels[scanId]
	for i, l := range metaLabels.MetaLabels {
		if columnName == l {
			columnIdx = i
			columnType = metaLabels.MetaTypes[i]
			break
		}
	}

	if columnIdx < 0 {
		return nil, fmt.Errorf("No meta for column named %v", columnName)
	}

	// Return a map containing all the items
	clientMap := &protos.ClientMap{
		EntryPMCs: []int32{},
	}

	// Depending on type, init the right array
	if columnType == protos.ScanMetaDataType_MT_INT {
		clientMap.IntValues = []int64{}
	} else if columnType == protos.ScanMetaDataType_MT_FLOAT {
		clientMap.FloatValues = []float64{}
	} else {
		clientMap.StringValues = []string{}
	}

	scanMeta := c.scanEntryMeta[scanId].Entries
	scanEntries := c.scanEntries[scanId].Entries
	for i, meta := range scanMeta {
		// If we have a value, add it to the map
		if val, ok := meta.Meta[int32(columnIdx)]; ok {
			if columnType == protos.ScanMetaDataType_MT_INT {
				clientMap.IntValues = append(clientMap.IntValues, int64(val.GetIvalue()))
			} else if columnType == protos.ScanMetaDataType_MT_FLOAT {
				clientMap.FloatValues = append(clientMap.FloatValues, float64(val.GetFvalue()))
			} else {
				clientMap.StringValues = append(clientMap.StringValues, val.GetSvalue())
			}

			// We must've added something, so add the PMC
			pmc := scanEntries[i].Id
			clientMap.EntryPMCs = append(clientMap.EntryPMCs, pmc)
		}
	}

	return clientMap, nil
}

func (c *APIClient) ListScanQuants(scanId string) (*protos.QuantListResp, error) {
	req := &protos.QuantListReq{}

	if len(scanId) > 0 {
		req.SearchParams = &protos.SearchParams{ScanId: scanId}
	}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_QuantListReq{
		QuantListReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}

	return resps[0].GetQuantListResp(), nil
}

func (c *APIClient) ensureQuant(quantId string) error {
	if _, ok := c.quants[quantId]; ok {
		return nil // already downloaded
	}

	req := &protos.QuantGetReq{QuantId: quantId, SummaryOnly: false}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_QuantGetReq{
		QuantGetReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	resp := resps[0].GetQuantGetResp()
	c.quants[quantId] = resp
	return nil
}

func (c *APIClient) GetQuant(quantId string, summaryOnly bool) (*protos.QuantGetResp, error) {
	if err := c.ensureQuant(quantId); err != nil {
		return nil, err
	}

	// Grab the quant data, if we're set to summaryOnly, don't return the rest
	if summaryOnly {
		return &protos.QuantGetResp{
			Summary: c.quants[quantId].Summary,
		}, nil
	}

	return c.quants[quantId], nil
}

func (c *APIClient) GetQuantColumns(quantId string) (*protos.ClientStringList, error) {
	if err := c.ensureQuant(quantId); err != nil {
		return nil, err
	}

	return &protos.ClientStringList{Strings: c.quants[quantId].Data.Labels}, nil
}

func (c *APIClient) GetQuantColumn(quantId string, columnName string, detector string) (*protos.ClientMap, error) {
	if err := c.ensureQuant(quantId); err != nil {
		return nil, err
	}

	quant := c.quants[quantId]
	columnIdx := -1
	columnType := protos.Quantification_QT_FLOAT
	for i, l := range quant.Data.Labels {
		if l == columnName {
			fmt.Println(l)
			// We've found our column!
			columnIdx = i
			columnType = quant.Data.Types[i]
			break
		}
	}

	if columnIdx < 0 {
		return nil, fmt.Errorf("No quant column named %v", columnName)
	}

	quantMap := &protos.ClientMap{
		EntryPMCs: []int32{},
	}

	// Depending on type, init the right array
	if columnType == protos.Quantification_QT_INT {
		quantMap.IntValues = []int64{}
	} else {
		quantMap.FloatValues = []float64{}
	}

	detectorFound := false
	for _, locSet := range quant.Data.LocationSet {
		if detector == locSet.Detector {
			detectorFound = true
			// This is what we're returning!
			for _, loc := range locSet.Location {
				quantMap.EntryPMCs = append(quantMap.EntryPMCs, loc.Pmc)

				if columnType == protos.Quantification_QT_INT {
					quantMap.IntValues = append(quantMap.IntValues, int64(loc.Values[columnIdx].Ivalue))
				} else {
					quantMap.FloatValues = append(quantMap.FloatValues, float64(loc.Values[columnIdx].Fvalue))
				}
			}
		}
	}

	if !detectorFound {
		return nil, fmt.Errorf("Detector \"%v\" not found in quant", detector)
	}

	return quantMap, nil
}

func (c *APIClient) ListScanImages(scanIds []string, mustIncludeAll bool) (*protos.ImageListResp, error) {
	req := &protos.ImageListReq{
		ScanIds:        scanIds,
		MustIncludeAll: mustIncludeAll,
	}

	fmt.Printf("%v", scanIds)
	msg := &protos.WSMessage{Contents: &protos.WSMessage_ImageListReq{
		ImageListReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}

	return resps[0].GetImageListResp(), nil
}

func (c *APIClient) ListScanROIs(scanId string) (*protos.RegionOfInterestListResp, error) {
	req := &protos.RegionOfInterestListReq{
		SearchParams: &protos.SearchParams{ScanId: scanId},
	}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_RegionOfInterestListReq{
		RegionOfInterestListReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}

	return resps[0].GetRegionOfInterestListResp(), nil
}

func (c *APIClient) GetROI(id string, isMist bool) (*protos.RegionOfInterestGetResp, error) {
	req := &protos.RegionOfInterestGetReq{Id: id, IsMIST: isMist}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_RegionOfInterestGetReq{
		RegionOfInterestGetReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}
	resp := resps[0].GetRegionOfInterestGetResp()

	// Decode the PMCs
	decoded, err := indexcompression.DecodeIndexList(resp.RegionOfInterest.ScanEntryIndexesEncoded, -1)

	if err != nil {
		return nil, err
	}

	resp.RegionOfInterest.ScanEntryIndexesEncoded = []int32{}
	for _, v := range decoded {
		resp.RegionOfInterest.ScanEntryIndexesEncoded = append(resp.RegionOfInterest.ScanEntryIndexesEncoded, int32(v))
	}

	return resp, nil
}

func (c *APIClient) GetScanBeamLocations(scanId string) (*protos.ClientBeamLocations, error) {
	if err := c.ensureScanEntries(scanId); err != nil {
		return nil, err
	}

	req := &protos.ScanBeamLocationsReq{ScanId: scanId}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_ScanBeamLocationsReq{
		ScanBeamLocationsReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}

	resp := resps[0].GetScanBeamLocationsResp()

	// Form our output data to contain PMCs along side the x,y,z values we just downloaded
	result := &protos.ClientBeamLocations{
		Locations: []*protos.ClientBeamLocation{},
	}

	for c, entry := range c.scanEntries[scanId].Entries {
		if entry.Location {
			loc := &protos.ClientBeamLocation{
				PMC:        entry.Id,
				Coordinate: resp.BeamLocations[c],
			}

			result.Locations = append(result.Locations, loc)
		}
	}

	return result, nil
}

func (c *APIClient) ensureScanEntries(scanId string) error {
	if _, ok := c.scanEntries[scanId]; ok {
		return nil // already downloaded
	}

	req := &protos.ScanEntryReq{ScanId: scanId}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_ScanEntryReq{
		ScanEntryReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	resp := resps[0].GetScanEntryResp()
	c.scanEntries[scanId] = resp

	// We also build some fast lookups
	scanPMCToLocIdx := map[int]int{}
	scanLocIdxToPMC := map[int]int{}
	for locIdx, item := range resp.Entries {
		scanPMCToLocIdx[int(item.Id)] = locIdx
		scanLocIdxToPMC[locIdx] = int(item.Id)
	}

	c.scanPMCToLocIdx[scanId] = scanPMCToLocIdx
	c.scanLocIdxToPMC[scanId] = scanLocIdxToPMC
	return nil
}

func (c *APIClient) GetScanEntries(scanId string) (*protos.ScanEntryResp, error) {
	if err := c.ensureScanEntries(scanId); err != nil {
		return nil, err
	}

	return c.scanEntries[scanId], nil
}

func (c *APIClient) ensureImageBeamVersions(imageName string) error {
	if _, ok := c.imageBeamVersions[imageName]; ok {
		return nil // already downloaded
	}

	req := &protos.ImageBeamLocationVersionsReq{ImageName: imageName}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_ImageBeamLocationVersionsReq{
		ImageBeamLocationVersionsReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	resp := resps[0].GetImageBeamLocationVersionsResp()

	// Cache it
	c.imageBeamVersions[imageName] = resp.BeamVersionPerScan
	return nil
}

func (c *APIClient) GetScanImageBeamLocationVersions(imageName string) (*protos.ImageBeamLocationVersionsResp, error) {
	err := c.ensureImageBeamVersions(imageName)
	if err != nil {
		return nil, err
	}

	return &protos.ImageBeamLocationVersionsResp{
		BeamVersionPerScan: c.imageBeamVersions[imageName],
	}, nil
}

// version can be -1 to indicate we just want the latest
func (c *APIClient) GetScanImageBeamLocations(imageName string, scanId string, version int32) (*protos.ImageBeamLocationsResp, error) {
	if version < 0 {
		if err := c.ensureImageBeamVersions(imageName); err != nil {
			return nil, err
		}

		if imageScanBeams, ok := c.imageBeamVersions[imageName][scanId]; ok {
			for _, v := range imageScanBeams.Versions {
				if int32(v) > version {
					version = int32(v)
				}
			}
		}
	}

	req := &protos.ImageBeamLocationsReq{ImageName: imageName, ScanBeamVersions: map[string]uint32{scanId: uint32(version)}}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_ImageBeamLocationsReq{
		ImageBeamLocationsReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}

	return resps[0].GetImageBeamLocationsResp(), nil
}

func getSpectraPMCs(scanEntries []*protos.ScanEntry) []int32 {
	idxs := []int32{}
	for _, entry := range scanEntries {
		if entry.NormalSpectra > 0 || entry.DwellSpectra > 0 {
			idxs = append(idxs, entry.Id)
		}
	}
	return idxs
}

func (c *APIClient) ensureDiffractionPeakStatuses(scanId string) error {
	req := &protos.DiffractionPeakStatusListReq{ScanId: scanId}
	// Not filling out entries, we just get all

	msg := &protos.WSMessage{Contents: &protos.WSMessage_DiffractionPeakStatusListReq{
		DiffractionPeakStatusListReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	// Really only caching this in case someone wants the whole response
	resp := resps[0].GetDiffractionPeakStatusListResp()
	c.scanDiffractionStatuses[scanId] = resp
	return nil
}

func (c *APIClient) ensureDiffractionDetected(scanId string) error {
	if err := c.ensureScanEntries(scanId); err != nil {
		return err
	}

	// Specify all indexes
	idxs := getSpectraPMCs(c.scanEntries[scanId].Entries)

	req := &protos.DetectedDiffractionPeaksReq{ScanId: scanId, Entries: &protos.ScanEntryRange{Indexes: idxs}}
	// Not filling out entries, we just get all

	msg := &protos.WSMessage{Contents: &protos.WSMessage_DetectedDiffractionPeaksReq{
		DetectedDiffractionPeaksReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return err
	}

	c.scanDiffractionDetected[scanId] = resps[0].GetDetectedDiffractionPeaksResp()
	return nil
}

func (c *APIClient) ensureBulkSumScanCalibration(scanId string) error {
	if err := c.ensureScanSpectra(scanId); err != nil {
		return err
	}
	if err := c.ensureScanMetaLabels(scanId); err != nil {
		return err
	}

	spectra := c.scanSpectra[scanId]

	if len(spectra.BulkSpectra) <= 0 {
		return fmt.Errorf("No bulk spectra found for scan %v when determining calibration", scanId)
	}

	// Find the indexes of the data we're looking for
	labels := c.scanMetaLabels[scanId].MetaLabels
	evStartIdx := -1
	evPerChanIdx := -1
	for idx, label := range labels {
		if label == "OFFSET" {
			evStartIdx = idx
		} else if label == "XPERCHAN" {
			evPerChanIdx = idx
		}

		if evStartIdx > -1 && evPerChanIdx > -1 {
			break
		}
	}

	if evStartIdx < 0 {
		return fmt.Errorf("Failed to find OFFSET label index for scan %v when determining calibration", scanId)
	}

	if evPerChanIdx < 0 {
		return fmt.Errorf("Failed to find XPERCHAN label index for scan %v when determining calibration", scanId)
	}

	calibrations := &protos.ClientEnergyCalibration{
		DetectorCalibrations: map[string]*protos.ClientSpectrumEnergyCalibration{},
	}

	for _, spectrum := range spectra.BulkSpectra {
		evStart := spectrum.Meta[int32(evStartIdx)].GetFvalue()
		evPerChannel := spectrum.Meta[int32(evPerChanIdx)].GetFvalue()

		calibrations.DetectorCalibrations[spectrum.Detector] = &protos.ClientSpectrumEnergyCalibration{StarteV: evStart, PerChanneleV: evPerChannel}
	}

	c.scanBulkSumEnergyCalibration[scanId] = calibrations
	return nil
}

func channelTokeV(channels []float32, cal *protos.ClientSpectrumEnergyCalibration) []float64 {
	result := []float64{}
	for _, ch := range channels {
		result = append(result, (float64(cal.StarteV)+float64(ch)*float64(cal.PerChanneleV))*0.001) // eV->keV conversion
	}
	return result
}

func (c *APIClient) GetScanBulkSumCalibration(scanId string) (*protos.ClientEnergyCalibration, error) {
	if err := c.ensureBulkSumScanCalibration(scanId); err != nil {
		return nil, err
	}

	return c.scanBulkSumEnergyCalibration[scanId], nil
}

func (c *APIClient) SetUserScanCalibration(scanId string, detector string, starteV float32, perChanneleV float32) (*protos.ClientEnergyCalibration, error) {
	// Ensure we have one stored for this scan:
	if _, ok := c.scanUserEnergyCalibration[scanId]; !ok {
		c.scanUserEnergyCalibration[scanId] = &protos.ClientEnergyCalibration{
			DetectorCalibrations: map[string]*protos.ClientSpectrumEnergyCalibration{},
		}
	}

	cal := c.scanUserEnergyCalibration[scanId]
	if _, ok := cal.DetectorCalibrations[detector]; !ok {
		cal.DetectorCalibrations[detector] = &protos.ClientSpectrumEnergyCalibration{}
	}

	// Write this calibration in
	cal.DetectorCalibrations[detector].StarteV = starteV
	cal.DetectorCalibrations[detector].PerChanneleV = perChanneleV
	return c.scanUserEnergyCalibration[scanId], nil
}

func (c *APIClient) GetDiffractionPeaks(scanId string, energyCalibrationSource protos.EnergyCalibrationSource) (*protos.ClientDiffractionData, error) {
	if err := c.ensureDiffractionDetected(scanId); err != nil {
		return nil, err
	}
	if err := c.ensureDiffractionPeakStatuses(scanId); err != nil {
		return nil, err
	}

	if energyCalibrationSource == protos.EnergyCalibrationSource_CAL_BULK_SUM {
		if err := c.ensureBulkSumScanCalibration(scanId); err != nil {
			return nil, err
		}
	}

	// Now that we have all data, form one view of it and return to client. This is intended to match the
	// view that is formed by UI code expression-data-sources.ts readDiffractionData function
	// Maybe it should be united somehow?

	detectedPeaks := c.scanDiffractionDetected[scanId]
	peakStatuses := c.scanDiffractionStatuses[scanId]
	var spectrumEnergyCalibration *protos.ClientEnergyCalibration

	if energyCalibrationSource == protos.EnergyCalibrationSource_CAL_BULK_SUM {
		spectrumEnergyCalibration = c.scanBulkSumEnergyCalibration[scanId]
	} else if energyCalibrationSource == protos.EnergyCalibrationSource_CAL_USER {
		// Check user has one set
		if cal, ok := c.scanUserEnergyCalibration[scanId]; !ok {
			return nil, fmt.Errorf("Failed to get user energy calibration for scan: %v", scanId)
		} else {
			// Use it!
			spectrumEnergyCalibration = cal
		}
	} else {
		return nil, fmt.Errorf("Failed to get energy calibration for scan: %v, source: %v", scanId, energyCalibrationSource)
	}

	// Some constants, along with others in this code!
	roughnessItemThreshold := float32(0.16)
	diffractionPeakHalfWidth := float32(15) * 0.5
	eVCalibrationDetector := "A"

	allPeaks := []*protos.ClientDiffractionPeak{}

	roughnessItems := []*protos.ClientRoughnessItem{}
	roughnessPMCs := map[int]bool{}

	for _, item := range detectedPeaks.PeaksPerLocation {
		pmc, err := strconv.Atoi(item.Id)
		if err != nil {
			fmt.Printf("Warning: Diffraction data contained invalid location id: %v", item.Id)
			continue
		}

		for _, peak := range item.Peaks {
			if peak.EffectSize <= 6 {
				continue
			}
			statusId := fmt.Sprintf("%v-%v", pmc, peak.PeakChannel)

			if peak.GlobalDifference > roughnessItemThreshold {
				// It's roughness, can repeat so ensure we only save once
				if _, ok := roughnessPMCs[pmc]; !ok {
					status := "intensity-mismatch"
					if s, ok := peakStatuses.PeakStatuses.Statuses[statusId]; ok {
						status = s.Status
					}

					roughnessItems = append(roughnessItems, &protos.ClientRoughnessItem{
						Id:               int32(pmc),
						GlobalDifference: peak.GlobalDifference,
						Deleted:          status != "intensity-mismatch",
					})
					roughnessPMCs[pmc] = true
				}
			} else if peak.PeakHeight > 0.64 {
				startChannel := float32(peak.PeakChannel) - diffractionPeakHalfWidth
				endChannel := float32(peak.PeakChannel) + diffractionPeakHalfWidth

				channels := []float32{float32(peak.PeakChannel), startChannel, endChannel}
				keVs := []float64{}
				for det, cal := range spectrumEnergyCalibration.DetectorCalibrations {
					if det == eVCalibrationDetector {
						keVs = channelTokeV(channels, cal)
					}
				}

				if len(keVs) == 3 {
					status := "diffraction-peak"
					if s, ok := peakStatuses.PeakStatuses.Statuses[statusId]; ok {
						status = s.Status
					}

					allPeaks = append(allPeaks, &protos.ClientDiffractionPeak{
						Id: int32(pmc),
						Peak: &protos.DetectedDiffractionPerLocation_DetectedDiffractionPeak{
							PeakChannel:       peak.PeakChannel,
							EffectSize:        peak.EffectSize,
							BaselineVariation: peak.BaselineVariation,
							GlobalDifference:  peak.GlobalDifference,
							DifferenceSigma:   peak.DifferenceSigma,
							PeakHeight:        peak.PeakHeight,
							Detector:          peak.Detector,
						},
						EnergykeV:      float32(keVs[0]),
						StartEnergykeV: float32(keVs[1]),
						EndEnergykeV:   float32(keVs[2]),
						Status:         status,
					})
				}
			}
		}
	}

	// We return these separately
	return &protos.ClientDiffractionData{Peaks: allPeaks, Roughnesses: roughnessItems}, nil
}

func (c *APIClient) CreateROI(roiItem *protos.ROIItem, isMist bool) (*protos.RegionOfInterestWriteResp, error) {
	req := &protos.RegionOfInterestWriteReq{RegionOfInterest: roiItem, IsMIST: isMist}

	msg := &protos.WSMessage{Contents: &protos.WSMessage_RegionOfInterestWriteReq{
		RegionOfInterestWriteReq: req,
	}}

	resps, err := c.sendMessageWaitResponse(msg)
	if err != nil {
		return nil, err
	}

	return resps[0].GetRegionOfInterestWriteResp(), nil
}
