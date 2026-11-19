package endpoints

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pixlise/core/v4/api/dbCollections"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/services"
	"github.com/pixlise/core/v4/api/services/servicesMock"
	wsHandler "github.com/pixlise/core/v4/api/ws/handlers"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/idgen"
	"github.com/pixlise/core/v4/core/jwtparser"
	"github.com/pixlise/core/v4/core/logger"
	"github.com/pixlise/core/v4/core/timestamper"
	"github.com/pixlise/core/v4/core/wstestlib"
	protos "github.com/pixlise/core/v4/generated-protos"
	"go.mongodb.org/mongo-driver/bson"
)

const scanFileTestScanId = "scanfiletest1"

func errStatus(err error) int {
	if err == nil {
		return http.StatusOK
	}
	if e, ok := err.(errorwithstatus.Error); ok {
		return e.Status()
	}
	return http.StatusInternalServerError
}

func putScanFileForTest(svcs *services.APIServices, user jwtparser.JWTUserInfo, scanId string, fileName string, body []byte) (*httptest.ResponseRecorder, error) {
	req := httptest.NewRequest("PUT", "/scan-files", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/octet-stream")

	resp := httptest.NewRecorder()
	err := PutScanFile(apiRouter.ApiHandlerGenericParams{
		Svcs:       svcs,
		UserInfo:   user,
		PathParams: map[string]string{ScanIdentifier: scanId, FileNameIdentifier: fileName},
		Writer:     resp,
		Request:    req,
	})
	return resp, err
}

func getScanFileForTest(svcs *services.APIServices, user jwtparser.JWTUserInfo, scanId string, fileId string) (*httptest.ResponseRecorder, error) {
	resp := httptest.NewRecorder()
	err := GetScanFile(apiRouter.ApiHandlerGenericParams{
		Svcs:       svcs,
		UserInfo:   user,
		PathParams: map[string]string{ScanIdentifier: scanId, ScanFileIdentifier: fileId},
		Writer:     resp,
		Request:    httptest.NewRequest("GET", "/scan-files/download/"+scanId+"/"+fileId, nil),
	})
	return resp, err
}

func Test_ScanFiles(t *testing.T) {
	db := wstestlib.GetDB()
	ctx := context.TODO()

	editor := jwtparser.JWTUserInfo{Name: "Ed Itor", UserID: "auth0|scanfile-editor", Permissions: map[string]bool{"EDIT_SCAN": true}}
	viewer := jwtparser.JWTUserInfo{Name: "Vi Ewer", UserID: "auth0|scanfile-viewer", Permissions: map[string]bool{"EDIT_SCAN": true}}
	stranger := jwtparser.JWTUserInfo{Name: "Stranger", UserID: "auth0|scanfile-stranger", Permissions: map[string]bool{"EDIT_SCAN": true}}

	db.Collection(dbCollections.ScanFilesName).Drop(ctx)
	db.Collection(dbCollections.OwnershipName).DeleteOne(ctx, bson.M{"_id": scanFileTestScanId})

	log := &logger.NullLogger{}
	sessUsers := map[string]wsHelpers.HandlerContext{}

	storageRoot := t.TempDir()
	svcs := servicesMock.MakeMockSvcsWithFS(storageRoot, &idgen.IDGen{}, nil)
	svcs.MongoDB = db
	svcs.TimeStamper = &timestamper.UnixTimeNowStamper{}
	svcs.ScanFilesFS = fileaccess.MakeFSAccessS3Simulator(storageRoot)
	svcs.Config.ScanFilesBucket = "scan-files-bucket"

	for _, user := range []jwtparser.JWTUserInfo{editor, viewer, stranger} {
		db.Collection(dbCollections.UsersName).DeleteOne(ctx, bson.M{"_id": user.UserID})
		sessUser, err := wsHelpers.CreateDBUser("sess-"+user.UserID, user, db, "", log)
		if err != nil {
			t.Fatalf("Failed to create user %v: %v", user.UserID, err)
		}
		sessUsers[user.UserID] = wsHelpers.HandlerContext{SessUser: *sessUser, Svcs: &svcs}
	}

	_, err := db.Collection(dbCollections.OwnershipName).InsertOne(ctx, &protos.OwnershipItem{
		Id:         scanFileTestScanId,
		ObjectType: protos.ObjectType_OT_SCAN,
		Editors:    &protos.UserGroupList{UserIds: []string{editor.UserID}},
		Viewers:    &protos.UserGroupList{UserIds: []string{viewer.UserID}},
	})
	if err != nil {
		t.Fatal(err)
	}

	content := []byte("%PDF-1.7 pretend this is a pdf")

	if _, err := putScanFileForTest(&svcs, viewer, scanFileTestScanId, "report.pdf", content); errStatus(err) != http.StatusUnauthorized {
		t.Errorf("Expected viewer upload to be unauthorised, got: %v", err)
	}
	if _, err := putScanFileForTest(&svcs, stranger, scanFileTestScanId, "report.pdf", content); errStatus(err) != http.StatusUnauthorized {
		t.Errorf("Expected stranger upload to be unauthorised, got: %v", err)
	}
	noPerm := jwtparser.JWTUserInfo{UserID: editor.UserID, Permissions: map[string]bool{}}
	if _, err := putScanFileForTest(&svcs, noPerm, scanFileTestScanId, "report.pdf", content); errStatus(err) != http.StatusBadRequest {
		t.Errorf("Expected upload without EDIT_SCAN to fail, got: %v", err)
	}
	if _, err := putScanFileForTest(&svcs, editor, scanFileTestScanId, "empty.pdf", []byte{}); errStatus(err) != http.StatusBadRequest {
		t.Errorf("Expected empty upload to fail, got: %v", err)
	}
	if _, err := putScanFileForTest(&svcs, editor, "no-such-scan", "report.pdf", content); errStatus(err) != http.StatusNotFound {
		t.Errorf("Expected upload to unknown scan to be not found, got: %v", err)
	}

	resp, err := putScanFileForTest(&svcs, editor, scanFileTestScanId, "../../sneaky/report.pdf", content)
	if err != nil {
		t.Fatalf("Upload failed: %v", err)
	}
	if !strings.Contains(resp.Body.String(), `"name":"report.pdf"`) || strings.Contains(resp.Body.String(), "storagePath") {
		t.Errorf("Unexpected upload response: %v", resp.Body.String())
	}

	if _, err := putScanFileForTest(&svcs, editor, scanFileTestScanId, "report.pdf", content); errStatus(err) != http.StatusConflict {
		t.Errorf("Expected duplicate upload to conflict, got: %v", err)
	}

	stored := []string{}
	filepath.Walk(storageRoot, func(p string, info os.FileInfo, err error) error {
		if err == nil && !info.IsDir() {
			rel, _ := filepath.Rel(storageRoot, p)
			stored = append(stored, filepath.ToSlash(rel))
		}
		return nil
	})
	if len(stored) != 1 || !strings.HasPrefix(stored[0], "scan-files-bucket/Scan-Files/"+scanFileTestScanId+"/") || !strings.HasSuffix(stored[0], "/report.pdf") {
		t.Errorf("Unexpected stored files: %v", stored)
	}

	if _, err := wsHandler.HandleScanFileListReq(&protos.ScanFileListReq{ScanId: scanFileTestScanId}, sessUsers[stranger.UserID]); errStatus(err) != http.StatusUnauthorized {
		t.Errorf("Expected stranger list to be unauthorised, got: %v", err)
	}

	listResp, err := wsHandler.HandleScanFileListReq(&protos.ScanFileListReq{ScanId: scanFileTestScanId}, sessUsers[viewer.UserID])
	if err != nil || len(listResp.Files) != 1 {
		t.Fatalf("List failed: %v, %+v", err, listResp)
	}

	file := listResp.Files[0]
	if file.Name != "report.pdf" || file.SizeBytes != uint64(len(content)) || file.ContentType != "application/pdf" || file.StoragePath != "" ||
		file.CreatorUserId != editor.UserID || file.Creator == nil || file.Creator.Name != editor.Name {
		t.Errorf("Unexpected listed file: %+v", file)
	}

	resp, err = getScanFileForTest(&svcs, viewer, scanFileTestScanId, file.Id)
	if err != nil || !bytes.Equal(resp.Body.Bytes(), content) {
		t.Errorf("Viewer download failed: %v, body: %v", err, resp.Body.String())
	}
	if _, err := getScanFileForTest(&svcs, stranger, scanFileTestScanId, file.Id); errStatus(err) != http.StatusUnauthorized {
		t.Errorf("Expected stranger download to be unauthorised, got: %v", err)
	}
	if _, err := getScanFileForTest(&svcs, viewer, scanFileTestScanId, "no-such-file"); errStatus(err) != http.StatusNotFound {
		t.Errorf("Expected unknown file download to be not found, got: %v", err)
	}

	rename := &protos.ScanFileWriteReq{Id: file.Id, Name: "final report.pdf", Description: "The final one"}
	if _, err := wsHandler.HandleScanFileWriteReq(rename, sessUsers[viewer.UserID]); errStatus(err) != http.StatusUnauthorized {
		t.Errorf("Expected viewer rename to be unauthorised, got: %v", err)
	}

	writeResp, err := wsHandler.HandleScanFileWriteReq(rename, sessUsers[editor.UserID])
	if err != nil || writeResp.File.Name != "final report.pdf" || writeResp.File.Description != "The final one" {
		t.Errorf("Rename failed: %v, %+v", err, writeResp)
	}

	resp, err = getScanFileForTest(&svcs, viewer, scanFileTestScanId, file.Id)
	if err != nil || !bytes.Equal(resp.Body.Bytes(), content) {
		t.Errorf("Download after rename failed: %v", err)
	}

	if _, err := wsHandler.HandleScanFileDeleteReq(&protos.ScanFileDeleteReq{Id: file.Id}, sessUsers[viewer.UserID]); errStatus(err) != http.StatusUnauthorized {
		t.Errorf("Expected viewer delete to be unauthorised, got: %v", err)
	}
	if _, err := wsHandler.HandleScanFileDeleteReq(&protos.ScanFileDeleteReq{Id: file.Id}, sessUsers[editor.UserID]); err != nil {
		t.Errorf("Delete failed: %v", err)
	}
	if _, err := getScanFileForTest(&svcs, viewer, scanFileTestScanId, file.Id); errStatus(err) != http.StatusNotFound {
		t.Errorf("Expected deleted file download to be not found, got: %v", err)
	}

	for _, name := range []string{"a.docx", "b.pptx"} {
		if _, err := putScanFileForTest(&svcs, editor, scanFileTestScanId, name, content); err != nil {
			t.Fatalf("Upload of %v failed: %v", name, err)
		}
	}
	if err := wsHelpers.DeleteAllScanFiles(ctx, scanFileTestScanId, &svcs); err != nil {
		t.Errorf("DeleteAllScanFiles failed: %v", err)
	}
	listResp, err = wsHandler.HandleScanFileListReq(&protos.ScanFileListReq{ScanId: scanFileTestScanId}, sessUsers[editor.UserID])
	if err != nil || len(listResp.Files) != 0 {
		t.Errorf("Expected no files after deleting all: %v, %+v", err, listResp)
	}

	remaining, _ := svcs.ScanFilesFS.ListObjects("scan-files-bucket", "")
	if len(remaining) != 0 {
		t.Errorf("Expected storage to be empty, found: %v", remaining)
	}
}
