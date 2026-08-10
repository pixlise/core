package notificationSender

import (
	"fmt"

	protos "github.com/pixlise/core/v4/generated-protos"
)

// For testing, this "implementation" simply printf's each call so we can capture it in Example test Output blocks

type MockNotificationSender struct{}

func (ns *MockNotificationSender) NotifyNewScan(scanName string, scanId string) {
	fmt.Printf("==>NotifyNewScan(%v,%v)\n", scanName, scanId)
}

func (ns *MockNotificationSender) NotifyUpdatedScan(scanName string, scanId string) {
	fmt.Printf("==>NotifyUpdatedScan(%v,%v)\n", scanName, scanId)
}

func (ns *MockNotificationSender) SysNotifyScanChanged(scanId string) {
	fmt.Printf("==>SysNotifyScanChanged(%v)\n", scanId)
}

func (ns *MockNotificationSender) SysNotifyROIChanged(roiId string) {
	fmt.Printf("==>SysNotifyROIChanged(%v)\n", roiId)
}

func (ns *MockNotificationSender) SysNotifyMapChanged(mapId string) {
	fmt.Printf("==>SysNotifyMapChanged(%v)\n", mapId)
}

func (ns *MockNotificationSender) NotifyNewScanImage(scanName string, scanId string, imageName string) {
	fmt.Printf("==>NotifyNewScanImage(%v,%v,%v)\n", scanName, scanId, imageName)
}

func (ns *MockNotificationSender) SysNotifyScanImagesChanged(imageName string, scanIds []string) {
	fmt.Printf("==>SysNotifyScanImagesChanged(%v,%v)\n", imageName, scanIds)
}

func (ns *MockNotificationSender) NotifyNewQuant(uploaded bool, quantId string, quantName string, status string, scanName string, scanId string) {
	fmt.Printf("==>NotifyNewQuant(%v,%v,%v,%v,%v,%v)\n", uploaded, quantId, quantName, status, scanName, scanId)
}

func (ns *MockNotificationSender) SysNotifyQuantChanged(quantId string) {
	fmt.Printf("==>SysNotifyQuantChanged(%v)\n", quantId)
}

func (ns *MockNotificationSender) NotifyObjectShared(objectType string, objectId string, objectName, sharerName string) {
	fmt.Printf("==>NotifyObjectShared(%v,%v,%v,%v)\n", objectType, objectId, objectName, sharerName)
}

func (ns *MockNotificationSender) NotifyUserGroupMessage(subject string, message string, notificationType protos.NotificationType, actionLink string, groupId string, groupName string, sender string) {
	fmt.Printf("==>NotifyUserGroupMessage(%v,%v,%v,%v,%v,%v,%v)\n", subject, message, notificationType, actionLink, groupId, groupName, sender)
}

func (ns *MockNotificationSender) NotifyUserMessage(subject string, message string, notificationType protos.NotificationType, actionLink string, requestorUserId string, destUserIds []string, sender string) {
	fmt.Printf("==>NotifyUserMessage(%v,%v,%v,%v,%v,%v,%v)\n", subject, message, notificationType, actionLink, requestorUserId, destUserIds, sender)
}
