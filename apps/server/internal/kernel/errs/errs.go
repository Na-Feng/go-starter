package errs

import (
	"errors"
	"fmt"
	"net/http"
)

type AppError struct {
	Code       int    `json:"code"`
	HTTPStatus int    `json:"-"`
	Message    string `json:"message"`
	Cause      error  `json:"-"`
}

func (e *AppError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("code=%d msg=%s: %v", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("code=%d msg=%s", e.Code, e.Message)
}

func (e *AppError) Unwrap() error {
	return e.Cause
}

func New(httpStatus, code int, msg string, cause error) *AppError {
	return &AppError{Code: code, HTTPStatus: httpStatus, Message: msg, Cause: cause}
}

func BadRequest(msg string) *AppError      { return New(http.StatusBadRequest, 40001, msg, nil) }
func Unauthorized(msg string) *AppError    { return New(http.StatusUnauthorized, 40101, msg, nil) }
func Forbidden(msg string) *AppError       { return New(http.StatusForbidden, 40301, msg, nil) }
func NotFound(msg string) *AppError        { return New(http.StatusNotFound, 40401, msg, nil) }
func Conflict(msg string) *AppError        { return New(http.StatusConflict, 40901, msg, nil) }
func TooManyRequests(msg string) *AppError { return New(http.StatusTooManyRequests, 42901, msg, nil) }
func Unavailable(msg string, cause error) *AppError {
	return New(http.StatusServiceUnavailable, 50301, msg, cause)
}
func Internal(cause error) *AppError {
	return New(http.StatusInternalServerError, 50000, "internal server error", cause)
}

func From(err error) *AppError {
	if err == nil {
		return nil
	}
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr
	}
	return Internal(err)
}
