package utils

import protos "github.com/pixlise/core/v4/generated-protos"

func SpectrumTypeToString(spectrumType protos.SpectrumType) string {
	switch spectrumType {
	case protos.SpectrumType_SPECTRUM_NORMAL:
		return "Normal"
	case protos.SpectrumType_SPECTRUM_DWELL:
		return "Dwell"
	case protos.SpectrumType_SPECTRUM_BULK:
		return "BulkSum"
	case protos.SpectrumType_SPECTRUM_MAX:
		return "MaxValue"
	}
	return ""
}

func StringToSpectrumType(spectrumType string) protos.SpectrumType {
	switch spectrumType {
	case "Normal":
		return protos.SpectrumType_SPECTRUM_NORMAL
	case "Dwell":
		return protos.SpectrumType_SPECTRUM_DWELL
	case "BulkSum":
		return protos.SpectrumType_SPECTRUM_BULK
	case "MaxValue":
		return protos.SpectrumType_SPECTRUM_MAX
	}

	return protos.SpectrumType_SPECTRUM_UNKNOWN
}
