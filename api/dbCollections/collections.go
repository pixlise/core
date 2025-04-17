package dbCollections

// All the DB collection names, so there's no confusion among all our code, also one place to rename

// NOTE DON'T FORGET TO UPDATE GetAllCollections() BELOW!!!

const CoregJobCollection = "coregJobs"
const DetectorConfigsName = "detectorConfigs"
const DiffractionDetectedPeakStatusesName = "diffractionDetectedPeakStatuses"
const DiffractionManualPeaksName = "diffractionManualPeaks"
const DOIName = "doi"
const ElementSetsName = "elementSets"
const ExpressionGroupsName = "expressionGroups"
const ExpressionsName = "expressions"
const ImageBeamLocationsName = "imageBeamLocations"
const ImagesName = "images"
const JobStatusName = "jobStatuses"
const JobHandlersName = "jobHandlers"
const MemoisedItemsName = "memoisedItems"
const MistROIsName = "mistROIs"
const ModulesName = "modules"
const ModuleVersionsName = "moduleVersions"
const NotificationsName = "notifications"
const OwnershipName = "ownership"
const PiquantVersionName = "piquantVersion"
const QuantificationsName = "quantifications"
const QuantificationZStacksName = "quantificationZStacks"
const RegionsOfInterestName = "regionsOfInterest"
const ScanAutoShareName = "scanAutoShare"
const ScanDefaultImagesName = "scanDefaultImages"
const ScansName = "scans"
const ScreenConfigurationName = "screenConfigurations"
const SelectionName = "selection"
const TagsName = "tags"
const UserGroupJoinRequestsName = "userGroupJoinRequests"
const UserGroupsName = "userGroups"
const UserROIDisplaySettings = "userROIDisplaySettings"
const UserExpressionDisplaySettings = "userExpressionDisplaySettings"
const UsersName = "users"
const UserImpersonatorsName = "userImpersonators"
const ViewStatesName = "viewStates"
const WidgetDataName = "widgetData"
const ConnectTempTokensName = "connectTempTokens"
const JobsName = "jobs"

func GetAllCollections() []string {
	return []string{
		CoregJobCollection,
		DetectorConfigsName,
		DiffractionDetectedPeakStatusesName,
		DiffractionManualPeaksName,
		DOIName,
		ElementSetsName,
		ExpressionGroupsName,
		ExpressionsName,
		ImageBeamLocationsName,
		ImagesName,
		JobStatusName,
		MemoisedItemsName,
		MistROIsName,
		ModulesName,
		ModuleVersionsName,
		NotificationsName,
		OwnershipName,
		PiquantVersionName,
		QuantificationsName,
		QuantificationZStacksName,
		RegionsOfInterestName,
		ScanAutoShareName,
		ScanDefaultImagesName,
		ScansName,
		ScreenConfigurationName,
		SelectionName,
		TagsName,
		UserGroupJoinRequestsName,
		UserGroupsName,
		UserROIDisplaySettings,
		UsersName,
		ViewStatesName,
		WidgetDataName,
		ConnectTempTokensName,
		JobsName,
	}
}
