package client

import (
	"fmt"
	"slices"

	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

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
		Counts:   spectrum.Counts,
		MaxCount: spectrum.MaxCount,
		Meta:     meta,
	}, nil
}

func (c *APIClient) makeSendableSpectrum(scanId string, spectrum *protos.ClientSpectrum) (*protos.Spectrum, error) {
	if err := c.ensureScanMetaLabels(scanId); err != nil {
		return nil, err
	}

	labels := c.scanMetaLabels[scanId]

	meta := map[int32]*protos.ScanMetaDataItem{}
	for k, item := range spectrum.Meta {
		// Find the string label
		labelIdx := slices.Index(labels.MetaLabels, k)
		if labelIdx < 0 {
			return nil, fmt.Errorf("No meta label index for %v", k)
		}
		meta[int32(labelIdx)] = item
	}

	return &protos.Spectrum{
		Detector: spectrum.Detector,
		Type:     spectrum.Type,
		Counts:   utils.ZeroRunEncode(spectrum.Counts),
		MaxCount: spectrum.MaxCount,
		Meta:     meta,
	}, nil
}
