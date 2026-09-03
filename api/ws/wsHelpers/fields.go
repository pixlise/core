package wsHelpers

import (
	"fmt"

	"github.com/pixlise/core/v4/core/errorwithstatus"
)

const IdFieldMaxLength = 32
const Auth0UserIdFieldMaxLength = 32
const DescriptionFieldMaxLength = 1024 * 5
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

func MaskSecretField(s string) string {
	masked := "***"

	stars := len(s) - 3 /* at the end */ - 3 /* We already have 3 */
	for c := 0; c < stars; c++ {
		masked = masked + "*"
	}

	// Append remainder if any
	if len(s) > len(masked) {
		masked = masked + s[len(masked):]
	}

	return masked
}
