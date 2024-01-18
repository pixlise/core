package quantification

import (
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/indexcompression"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// Validates the create parameters. Side-effect of modifying PmcsEncoded to just be an array of decoded PMCs
func IsValidCreateParam(createParams *protos.QuantCreateParams, hctx wsHelpers.HandlerContext) error {
	if createParams.RoiIDs == nil {
		// Make it an empty list if its nil...
		createParams.RoiIDs = []string{}
		//return errors.New("RoiIDs cannot be nil")
	}

	if len(createParams.Command) <= 0 {
		return errors.New("PIQUANT command to run was not supplied")
	}

	// We only require the name to be set in map mode
	if createParams.Command == "map" {
		if len(createParams.Name) <= 0 {
			return errors.New("Name not supplied")
		}

		// Validate things, eg no quants named the same already, parameters filled out as expected, etc...
		if checkQuantificationNameExists(createParams.Name, createParams.ScanId, hctx) {
			return fmt.Errorf("Name already used: %v", createParams.Name)
		}
	} else if createParams.Command == "quant" {
		// This is the Fit command
		createParams.Name = ""
	} else {
		return fmt.Errorf("Unexpected command requested: %v", createParams.Command)
	}

	// Might be given either empty elements, or if string conversion (with split(',')) maybe we got [""]...
	if len(createParams.Elements) <= 0 || len(createParams.Elements[0]) <= 0 {
		return errors.New("Elements not supplied")
	}

	if len(createParams.DetectorConfig) <= 0 {
		return errors.New("DetectorConfig not supplied")
	}

	// At this point, we're assuming that the detector config is a valid config name / version. We need this to be the path of the config in S3
	// so here we convert it and ensure it's valid
	detectorConfigBits := strings.Split(createParams.DetectorConfig, "/")
	if len(detectorConfigBits) != 2 || len(detectorConfigBits[0]) < 0 || len(detectorConfigBits[1]) < 0 {
		return errors.New("DetectorConfig not in expected format")
	}

	if createParams.RunTimeSec < 1 {
		return errors.New("RunTimeSec is invalid")
	}

	// Expect there to be at least one PMC specified
	if len(createParams.Pmcs) <= 0 {
		return errors.New("No PMCs specified")
	}

	// Decode PMCs here and from this point they won't be "encoded"
	pmcs, err := indexcompression.DecodeIndexList(createParams.Pmcs, -1)
	if err != nil {
		return err
	}

	// Overwrite the encoded array with the decoded one
	createParams.Pmcs = []int32{}
	for _, pmc := range pmcs {
		createParams.Pmcs = append(createParams.Pmcs, int32(pmc))
	}

	// Search for weird characters in parameters. We don't want to allow people to do
	// command injection attacks here!! PIQUANT commands are fairly simple and take
	// flags eg -b often with values right after, comma separated. So we allow
	// only a few characters, to exclude things like ; and & so users can't form other
	// commands
	if len(createParams.Parameters) > 0 {
		err := validateParameters(createParams.Parameters)
		if err != nil {
			return err
		}
	}

	return nil
}

// Checks parameters don't contain something unexpected, to filter out
// code execution. Returns error or nil
func validateParameters(params string) error {
	r, err := regexp.Compile("^[a-zA-Z0-9 -.,_\"]+$")
	if err != nil {
		return err
	}

	if !r.MatchString(params) {
		return errors.New("Invalid parameters passed: " + params)
	}
	return nil
}
