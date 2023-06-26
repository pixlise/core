package wsHandler

import (
	"context"
	"errors"

	"github.com/olahol/melody"
	"github.com/pixlise/core/v3/api/services"
	protos "github.com/pixlise/core/v3/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const scanCollection = "scans"

func HandleScanListReq(req *protos.ScanListReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ScanListResp, error) {
	filter := bson.D{}
	opts := options.Find()
	cursor, err := svcs.MongoDB.Collection(scanCollection).Find(context.TODO(), filter, opts)
	if err != nil {
		return nil, err
	}

	scans := []*protos.ScanItem{}
	err = cursor.All(context.TODO(), &scans)
	if err != nil {
		return nil, err
	}

	return &protos.ScanListResp{
		Scans: scans,
	}, nil
}

func HandleScanMetaLabelsReq(req *protos.ScanMetaLabelsReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ScanMetaLabelsResp, error) {
	return nil, errors.New("HandleScanMetaLabelsReq not implemented yet")
}

func HandleScanMetaWriteReq(req *protos.ScanMetaWriteReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ScanMetaWriteResp, error) {
	return nil, errors.New("HandleScanMetaWriteReq not implemented yet")
}

func HandleScanTriggerReImportReq(req *protos.ScanTriggerReImportReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ScanTriggerReImportResp, error) {
	return nil, errors.New("HandleScanTriggerReImportReq not implemented yet")
}

func HandleScanUploadReq(req *protos.ScanUploadReq, s *melody.Session, m *melody.Melody, svcs *services.APIServices) (*protos.ScanUploadResp, error) {
	return nil, errors.New("HandleScanUploadReq not implemented yet")
}
