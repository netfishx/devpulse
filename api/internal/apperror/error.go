package apperror

import (
	"fmt"
	"net/http"
)

// AppError is the unified error type returned by service layer.
// Handler layer does NOT inspect it — the error middleware handles formatting.
type AppError struct {
	Code   int    `json:"-"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
	Err    error  `json:"-"`
}

func (e *AppError) Error() string { return e.Detail }
func (e *AppError) Unwrap() error { return e.Err }

func BadRequest(detail string) *AppError {
	return &AppError{Code: http.StatusBadRequest, Title: "Bad Request", Detail: detail}
}

func Unauthorized(detail string) *AppError {
	return &AppError{Code: http.StatusUnauthorized, Title: "Unauthorized", Detail: detail}
}

func Conflict(detail string) *AppError {
	return &AppError{Code: http.StatusConflict, Title: "Conflict", Detail: detail}
}

func NotFound(detail string) *AppError {
	return &AppError{Code: http.StatusNotFound, Title: "Not Found", Detail: detail}
}

func Internal(err error) *AppError {
	return &AppError{
		Code:   http.StatusInternalServerError,
		Title:  "Internal Server Error",
		Detail: "an unexpected error occurred",
		Err:    err,
	}
}

func Internalf(format string, args ...any) *AppError {
	return Internal(fmt.Errorf(format, args...))
}
