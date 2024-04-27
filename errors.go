package nues

import (
	"fmt"
)

type SysErrorData struct {
	Code    int
	Message string
}

type SysError interface {
	Error() string
}

func (e SysErrorData) Error() string {
	return fmt.Sprintf("%d: %s", e.Code, e.Message)
}

func NewError(code int, message string) SysError {
	return SysErrorData{
		Code:    code,
		Message: message,
	}
}

var (
	ErrSystemInternal       = NewError(0, "system error")
	ErrBadCommand           = NewError(1, "can't process your request")
	ErrUserNotAuth          = NewError(2, "not authorized")
	ErrPhoneBadFormat       = NewError(3, "phone number format not supported")
	ErrUserNotActive        = NewError(4, "user not active")
	ErrUserNotVerified      = NewError(5, "user not verified")
	ErrParsingData          = NewError(6, "cannot parse data")
	ErrMissingReuiredFields = NewError(7, "missing required fields")
	ErrProjectionFailed     = NewError(8, "projection failed")
	ErrUpsertFailed         = NewError(8, "upsert failed")
)
