package quantification

import "github.com/pixlise/core/v4/core/logger"

func RunAutoQuantifications(scanId string, logger logger.ILogger) {
	logger.Infof("Running auto-quantifications for scan: %v", scanId)
	// #3657 - Check if we have auto quant already, if not, run one
	// using the elements in the card and any other parameters. When
	// quant finishes generating it should notify user separately
}
