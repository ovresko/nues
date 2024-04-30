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
	ErrSystemInternal   = NewError(0, "system error")
	ErrBadCommand       = NewError(1, "can't process your request")
	ErrUserNotAuth      = NewError(2, "not authorized")
	ErrParsingData      = NewError(3, "cannot parse data")
	ErrProjectionFailed = NewError(4, "projection failed")
	ErrUpsertFailed     = NewError(5, "upsert failed")
	ErrPhoneBadFormat   = NewError(6, "phone format not supported")
)
