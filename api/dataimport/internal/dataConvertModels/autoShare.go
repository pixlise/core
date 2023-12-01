package dataConvertModels

import protos "github.com/pixlise/core/v3/generated-protos"

type AutoShareConfigItem struct {
	Sharer  string `bson:"_id"` // Either a user ID or some special string that the importer sets
	Viewers *protos.UserGroupList
	Editors *protos.UserGroupList
}
