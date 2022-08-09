package main

import "github.com/pixlise/core/core/fileaccess"

type configfile struct {
	Name       string `json:"name"`
	Detector   string `json:"detector"`
	Group      string `json:"group"`
	UpdateType string `json:"updateType"`
}

func getConfigFile() (configfile, error) {
	localFS := fileaccess.FSAccess{}

	var config configfile
	err := localFS.ReadJSON(localUnzipPath, "config.json", &config, false)
	return config, err
}

// TODO: This should probably take a configfile struct as a parameter...
func computeName() (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}
	return config.Name, nil
}

// TODO: This should probably take a configfile struct as a parameter...
func customDetector(sol string) (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}

	if config.Detector != "" {
		// Return a custom detector string.
		return config.Detector, nil
	} else if sol[0] >= '0' && sol[0] <= '9' {
		// Usual Sol number and no custom string, don't override.
		return "", nil
	} else if sol[0] == 'D' || sol[0] == 'C' {
		return "", nil
	} else {
		// Sol starts with a character, non-standard, use the EM detector.
		return "PIXL-EM-E2E", nil
	}
}

// TODO: This should probably take a configfile struct as a parameter...
func customGroup(detector string) (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}

	if config.Group != "" {
		// Return a custom detector string.
		return config.Group, nil
	} else if detector == "PIXL-EM-E2E" {
		return "PIXL-EM", nil
	} else {
		return "PIXL-FM", nil
	}
}

// TODO: This should probably take a configfile struct as a parameter...
func overrideUpdateType() (string, error) {
	config, err := getConfigFile()
	if err != nil {
		return "", err
	}

	if config.UpdateType != "" {
		// Return a custom detector string.
		return config.UpdateType, nil
	} else {
		return "full", nil
	}
}
