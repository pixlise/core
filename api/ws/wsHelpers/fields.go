package wsHelpers

import (
	"fmt"

	"github.com/pixlise/core/v3/core/errorwithstatus"
)

const IdFieldMaxLength = 16
const Auth0UserIdFieldMaxLength = 32
const DescriptionFieldMaxLength = 300
const SourceCodeMaxLength = 1024 * 1024 * 5 // Trying to be very generous here, but maybe this is not enough?
const TagListMaxLength = 100

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
	} else if minLength > 0 {
		return errorwithstatus.MakeBadRequestError(fmt.Errorf(`%v must contain at least %v items`, fieldName, minLength))
	}

	return nil
}
