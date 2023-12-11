package quantification

import (
	"strings"

	protos "github.com/pixlise/core/v3/generated-protos"
)

// getWeightPercentColumnsInQuant - returns weight % columns, ones ending in _%
func getWeightPercentColumnsInQuant(quant *protos.Quantification) []string {
	result := []string{}
	for _, label := range quant.Labels {
		if strings.HasSuffix(label, "_%") {
			result = append(result, label)
		}
	}
	return result
}

// getQuantColumnIndex - returns index of column in quantification or -1 if not found
func getQuantColumnIndex(quant *protos.Quantification, column string) int32 {
	for c, label := range quant.Labels {
		if label == column {
			return int32(c)
		}
	}
	return -1
}
