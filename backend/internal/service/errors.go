package service

import "errors"

type ErrorCode string

const (
	ErrorCodeValidation ErrorCode = "validation"
	ErrorCodeNotFound   ErrorCode = "not_found"
	ErrorCodeForbidden  ErrorCode = "forbidden"
	ErrorCodeConflict   ErrorCode = "conflict"
	ErrorCodeDependency ErrorCode = "dependency"
	ErrorCodeInternal   ErrorCode = "internal"
)

type Error struct {
	Code    ErrorCode
	Message string
	Cause   error
}

func NewError(code ErrorCode, message string) *Error {
	return &Error{Code: code, Message: message}
}

func WrapError(code ErrorCode, message string, cause error) *Error {
	return &Error{Code: code, Message: message, Cause: cause}
}

func (err *Error) Error() string {
	if err == nil {
		return ""
	}
	if err.Cause == nil {
		return err.Message
	}
	return err.Message + ": " + err.Cause.Error()
}

func (err *Error) Unwrap() error {
	if err == nil {
		return nil
	}
	return err.Cause
}

func Code(err error) ErrorCode {
	var serviceErr *Error
	if errors.As(err, &serviceErr) {
		return serviceErr.Code
	}
	return ErrorCodeInternal
}

func SafeMessage(err error) string {
	var serviceErr *Error
	if errors.As(err, &serviceErr) && serviceErr.Message != "" {
		return serviceErr.Message
	}
	return "internal server error"
}
