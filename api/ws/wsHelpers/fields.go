package wsHelpers

import (
	"errors"

	"github.com/pixlise/core/v3/core/errorwithstatus"
)

func CheckStringField(field *string, fieldName string, minLength int, maxLength int) error {
	if field != nil {
		if len(*field) < minLength {
			return errorwithstatus.MakeBadRequestError(errors.New(fieldName + " is too short"))
		}
		if len(*field) > maxLength {
			return errorwithstatus.MakeBadRequestError(errors.New(fieldName + " is too long"))
		}
	}

	return nil
}
