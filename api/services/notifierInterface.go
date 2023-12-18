package services

type INotifier interface {
	// When a scan downlinks, or is uploaded by a user
	NotifyNewScan(scanName string, scanId string)

	// When a scan import is re-triggered
	NotifyUpdatedScan(scanName string, scanId string)

	// When a scan is deleted, or its metadata edited
	// NOTE: This does NOT send emails, it's of system-level interest only so UI can update caches as required
	SysNotifyScanChanged(scanId string)

	// When an image is added to a scan
	NotifyNewScanImage(scanName string, scanId string, imageName string)

	// When an image is deleted, or set as default, or had its metadata changed
	// NOTE: This does NOT send emails, it's of system-level interest only so UI can update caches as required
	SysNotifyScanImagesChanged(scanIds []string)

	// When a quant is completed or uploaded
	NotifyNewQuant(uploaded bool, quantId string, quantName string, status string, scanName string, scanId string)

	// When a quant is deleted
	// NOTE: This does NOT send emails, it's of system-level interest only so UI can update caches as required
	SysNotifyQuantChanged(quantId string)

	// When something is shared with a group
	NotifyObjectShared(objectType string, objectId string, objectName, sharerName string)

	// When a user sends a message to the group they belong to
	NotifyUserGroupMessage(subject string, message string, groupId string, groupName string, sender string)
}
