package wsHelpers

import (
	"fmt"

	"github.com/pixlise/core/v3/core/errorwithstatus"
)

const IdFieldMaxLength = 16
const Auth0UserIdFieldMaxLength = 32

func CheckStringField(field *string, fieldName string, minLength int, maxLength int) error {
	if field != nil {
		if len(*field) < minLength {
			return errorwithstatus.MakeBadRequestError(fmt.Errorf(`%v is too short`, fieldName))
		}
		if len(*field) > maxLength {
			return errorwithstatus.MakeBadRequestError(fmt.Errorf(`%v is too long`, fieldName))
		}
	}

	return nil
}

func CheckFieldLength[T any](field []T, fieldName string, minLength int, maxLength int) error {
	if field != nil {
		if len(field) < minLength {
			return errorwithstatus.MakeBadRequestError(fmt.Errorf(`%v is too short`, fieldName))
		}
		if len(field) > maxLength {
			return errorwithstatus.MakeBadRequestError(fmt.Errorf(`%v is too long`, fieldName))
		}
	}

	return nil
}
