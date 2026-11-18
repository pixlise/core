package endpoints

import (
	"errors"
	"fmt"
	"io"
	"path"
	"strconv"
	"strings"

	queryparams "github.com/pixlise/core/v4/api/endpoints/query-params"
	apiRouter "github.com/pixlise/core/v4/api/router"
	"github.com/pixlise/core/v4/api/ws/wsHelpers"
	"github.com/pixlise/core/v4/core/errorwithstatus"
	"github.com/pixlise/core/v4/core/fileaccess"
	"github.com/pixlise/core/v4/core/utils"
	protos "github.com/pixlise/core/v4/generated-protos"
)

// Allows uploading a file, either as one message or in multiple chunks with the possibility of resuming from
// a previously uploaded point. Expects query parameters:
// scan=<scanid>&filename=<filename-to-save>&totalBytes=<total-bytes-in-file>&writePos=<where-to-write>
// If totalBytes is defined, writePos also needs to be present. If neither are present, the PUT is interpreted
// as uploading the entire file in a single message
// The PUT body contains the data (entire file, or a chunk in case of multi-chunk uploads). If the PUT body is
// empty, the message is treated as a "resume check" request - and the response will specify how many bytes
// of the file have been received up to this point
// Response is a JSON structure, example:
//  {
//    bytesReceived: 123,
//    completed: true
//  }

func PutFileData(params apiRouter.ApiHandlerGenericParams) error {
	if !params.UserInfo.Permissions["EDIT_SCAN"] {
		return errorwithstatus.MakeBadRequestError(errors.New("PutFileData not allowed"))
	}

	// User might be impersonating someone, check this
	ctx := params.Request.Context()
	userId, err := checkImpersonation(ctx, params.Svcs, params.UserInfo.UserID)
	if err != nil {
		return fmt.Errorf("Failed to determine user id impersonation status: %v", err)
	}

	// Check access to each associated scan. The user should already have a web socket open by this point, so we can
	// look to see if there is a cached copy of their user group membership. If we don't find one, we stop
	memberOfGroupIds, viewerOfGroupIds, err := wsHelpers.GetCachedUserGroupAccess(userId, params.Svcs.MongoDB)
	if err != nil {
		// User is probably not logged in
		return errorwithstatus.MakeBadRequestError(errors.New("User has no group membership, can't determine permissions"))
	}

	scanId, ok := params.PathParams[queryparams.ScanIdentifier]
	if !ok {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Missing %v", queryparams.ScanIdentifier))
	}

	// NOTE: We allow blank for a scan id as some file uploads may be for importing a scan so we wouldn't have a scan id to begin with!
	// BUT:  If we do get a scan id, we make sure the user has access to it and ensure that the file name (as stored in our upload area)
	//       is prefixed with the scan id, eg how we store images
	scanId = fileaccess.MakeValidObjectName(scanId, false)
	if l := len(scanId); /*l <= 0 ||*/ l >= wsHelpers.IdFieldMaxLength {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid ScanID: %v, must be between 1 and %v characters long", scanId, wsHelpers.IdFieldMaxLength))
	}

	fileName, ok := params.PathParams[queryparams.FileNameIdentifier]
	if !ok {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Missing %v", queryparams.FileNameIdentifier))
	}

	if l := len(fileName); l <= 0 || l >= 100 {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid %v: \"%v\"", queryparams.FileNameIdentifier, fileName))
	}

	saveName := fileName

	// If we have a scan id provided, ensure we have access to it and ensure it prefixes the file name!
	if len(scanId) > 0 {
		// Check that user has access to this scan
		_, err = wsHelpers.CheckObjectAccessForUser(false, scanId, protos.ObjectType_OT_SCAN, userId, memberOfGroupIds, viewerOfGroupIds, params.Svcs.MongoDB)
		if err != nil {
			return err
		}

		if !strings.HasPrefix(saveName, scanId+"/") {
			saveName = path.Join(scanId, saveName)
		}
	}

	// Also need the byte size and bytes so far. If these aren't specified we just treat it like a single request upload of an entire file
	fileByteSizeStr, ok := params.PathParams[queryparams.FileTotalBytes]
	bytesUploadedSoFarStr, ok2 := params.PathParams[queryparams.BytesUploadedSoFar]

	if ok != ok2 {
		// Need either both or neither defined
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Must have both or neither of [%v, %v] defined", queryparams.FileTotalBytes, queryparams.BytesUploadedSoFar))
	}

	// Read in body
	bodyData, err := io.ReadAll(params.Request.Body)
	if err != nil {
		return err
	}

	if !ok && !ok2 {
		// Treat it as an upload-in-one message. Must contain data
		if len(bodyData) <= 0 {
			return errorwithstatus.MakeBadRequestError(fmt.Errorf("Uploading without [%v, %v] set can't contain empty file data (body)", queryparams.FileTotalBytes, queryparams.BytesUploadedSoFar))
		}

		_ /*sz*/, completed, err := params.Svcs.MultiPartUpload.StartUpload(saveName, uint64(len(bodyData)))
		if err != nil {
			return errorwithstatus.MakeBadRequestError(err)
		}

		if completed {
			// We already had it, just report no errors
			return nil
		}

		// Upload the chunk
		sz, completed, err := params.Svcs.MultiPartUpload.UploadPart(saveName, 0, bodyData)
		if err != nil {
			return errorwithstatus.MakeBadRequestError(err)
		}

		// Return the response
		resp := &protos.UploadHttpResponseBody{
			BytesReceived: sz,
			Completed:     completed,
		}
		utils.SendProtoJSON(params.Writer, resp)
		if !completed {
			params.Svcs.Log.Errorf("File \"%v\" uploaded as one chunk but not marked complete. Size: %v", saveName, sz)
			//return errorwithstatus.MakeStatusError(http.StatusInternalServerError, errors.New("File uploaded as one chunk but not marked complete"))
		}
		return nil
	}

	// If either are not numbers, stop here
	fileByteSize, err := strconv.Atoi(fileByteSizeStr)
	if err != nil {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to read %v: %v", queryparams.FileTotalBytes, err))
	} else if fileByteSize < 0 {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid %v: %v", queryparams.FileTotalBytes, err))
	}

	bytesUploadedSoFar, err := strconv.Atoi(bytesUploadedSoFarStr)
	if err != nil {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Failed to read %v: %v", queryparams.BytesUploadedSoFar, err))
	} else if bytesUploadedSoFar < 0 {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf("Invalid %v: %v", queryparams.BytesUploadedSoFar, err))
	}

	// If the data size is empty, we're being asked to start the upload and return the resume position
	if len(bodyData) <= 0 {
		sz, completed, err := params.Svcs.MultiPartUpload.StartUpload(saveName, uint64(fileByteSize))
		if err != nil {
			return errorwithstatus.MakeBadRequestError(err)
		}

		// Return completed flag and size to resume from (if not completed)
		resp := &protos.UploadHttpResponseBody{
			BytesReceived: sz,
			Completed:     completed,
		}
		utils.SendProtoJSON(params.Writer, resp)
		return nil
	}

	// Last case, we got a chunk of data, save it and continue
	sz, completed, err := params.Svcs.MultiPartUpload.UploadPart(saveName, uint64(bytesUploadedSoFar), bodyData)
	if err != nil {
		return errorwithstatus.MakeBadRequestError(err)
	}

	resp := &protos.UploadHttpResponseBody{
		BytesReceived: sz,
		Completed:     completed,
	}

	utils.SendProtoJSON(params.Writer, resp)
	return nil
}
