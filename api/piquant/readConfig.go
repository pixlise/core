package piquant

import (
	"fmt"

	"github.com/pixlise/core/v4/api/filepaths"
	"github.com/pixlise/core/v4/api/services"
	protos "github.com/pixlise/core/v4/generated-protos"
)

func ReadConfig(id string, svcs *services.APIServices) (*protos.DetectorConfig, []string, error) {
	cfg, err := GetDetectorConfig(id, svcs.MongoDB)
	if err != nil {
		return nil, []string{}, err
	}

	// Read versions
	versions := GetPiquantConfigVersions(svcs, id)
	if len(versions) <= 0 {
		return nil, versions, fmt.Errorf("DetectorConfig %v has no versions defined", id)
	}

	latestVersion := versions[len(versions)-1]

	// Read PIQUANT config file
	piquantCfg, err := GetPIQUANTConfig(svcs, id, latestVersion)
	if err != nil {
		return nil, versions, err
	}

	// Retrieve elevAngle
	cfgPath := filepaths.GetDetectorConfigPath(id, latestVersion, piquantCfg.ConfigFile)
	piquantCfgFile, err := svcs.FS.ReadObject(svcs.Config.ConfigBucket, cfgPath)
	if err != nil {
		return nil, versions, err
	}

	// Find the value
	piquantCfgFileStr := string(piquantCfgFile)
	angle, err := ReadFieldFromPIQUANTConfigMSA(piquantCfgFileStr, "#ELEVANGLE")
	if err != nil {
		svcs.Log.Errorf("Failed to read ELEVANGLE from Piquant config file: %v, trying emerg_angle", cfgPath)

		// EM config has a value "emerg_angle" which is also set to 70, maybe it's an interchangeable name?
		angle, err = ReadFieldFromPIQUANTConfigMSA(piquantCfgFileStr, "emerg_angle")
		if err != nil {
			return nil, versions, fmt.Errorf("Failed to read ELEVANGLE and emerg_angle from Piquant config file: %v", cfgPath)
		}
	}

	cfg.ElevAngle = angle
	return cfg, versions, nil
}
