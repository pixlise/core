package bruker

import (
	"fmt"
	"path/filepath"
	"strings"

	brukerreader "github.com/pixlise/core/v4/api/dataimport/internal/converters/bruker/bruker-reader"
	"github.com/pixlise/core/v4/core/fileaccess"
)

func getBCFFile(importPath string) string {
	localFS := &fileaccess.FSAccess{}

	items, err := localFS.ListObjects(importPath, "")
	if err != nil {
		return ""
	}

	for _, file := range items {
		if strings.HasSuffix(file, ".bcf") {
			return filepath.Join(importPath, file)
		}
	}

	return ""
}

func getMetaField(meta map[string]interface{}, names ...string) (interface{}, error) {
	curr := meta
	for c, name := range names {
		if item, ok := curr[name]; !ok {
			return nil, fmt.Errorf("Field not found: %v", strings.Join(names, ","))
		} else {
			if c == len(names)-1 {
				return item, nil
			}

			// Otherwise traverse in...
			curr = item.(map[string]interface{})
		}
	}

	return nil, fmt.Errorf("Field find failed: %v", strings.Join(names, ","))
}

func readBasicItemMeta(item *brukerreader.HspyItem) (string, string, error) {
	signal := ""
	titleStr := ""
	if s, err := getMetaField(item.Metadata, "Signal", "record_by"); err == nil {
		signal = s.(string)
	} else {
		return "", "", err
	}

	// Grab the sample name as the title - but if it's empty we can use original file name (but cut off the extension!)
	if t, err := getMetaField(item.Metadata, "Sample", "name"); err == nil {
		//titleStr = t.(string)

		// During debugging it seems sometimes title above was "" after running this code?? If we add the following
		// lines though it seems to work
		if tt, ok := t.(string); ok {
			titleStr = tt
		}
	} else {
		return "", "", err
	}

	if len(titleStr) <= 0 {
		if t, err := getMetaField(item.Metadata, "General", "original_filename"); err == nil {
			titleStr = t.(string)

			idx := strings.Index(titleStr, ".")
			if idx > 0 {
				titleStr = titleStr[0:idx]
			}
		} else {
			return "", "", err
		}
	}

	return signal, titleStr, nil
}

type allAxes struct {
	width, height, channels int

	units string

	xOffset, yOffset, zOffset float64
	xScale, yScale, zScale    float64
}

func readAxes(axes []brukerreader.Axis) (allAxes, error) {
	result := allAxes{}
	result.xScale = 1
	result.yScale = 1
	result.zScale = 1

	for _, axis := range axes {
		if len(axis.Units) <= 0 {
			return result, fmt.Errorf("No axis units defined for: %v", axis.Name)
		}
		if axis.Units != "µm" && axis.Units != "keV" {
			return result, fmt.Errorf("Unknown %v axis units: %v", axis.Name, axis.Units)
		}

		if axis.Name == "width" {
			result.xOffset = axis.Offset
			result.xScale = axis.Scale
			result.width = axis.Size

			//result.xScale /= float64(result.width)
		} else if axis.Name == "height" {
			result.yOffset = axis.Offset
			result.yScale = axis.Scale
			result.height = axis.Size

			//result.yScale /= float64(result.height)
		} else if axis.Name == "Energy" {
			result.zOffset = axis.Offset
			result.zScale = axis.Scale
			result.channels = axis.Size

			// This is the max channel, but we want the number of channels
			//channels++

			continue // NOTE: we don't check units match or whatever, because they don't! We're assuming Z must be Energy...
		} else {
			return result, fmt.Errorf("Unexpected axis: %v", axis.Name)
		}

		if len(result.units) > 0 {
			// Check that other axis has same units
			if result.units != axis.Units {
				return result, fmt.Errorf("Axes must have same units, detected %v and %v", axis.Units, result.units)
			}
		} else {
			// First unit, save it
			result.units = axis.Units
		}
	}

	return result, nil
}
