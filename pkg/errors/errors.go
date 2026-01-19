package errors

import "net/http"

// AppError is a custom error type that includes an HTTP status code
type AppError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

func (e *AppError) Error() string {
	return e.Message
}

// NewAppError creates a new AppError
func NewAppError(code int, message string) *AppError {
	return &AppError{
		Code:    code,
		Message: message,
	}
}

// Common errors
var (
	ErrInvalidRequest = NewAppError(http.StatusBadRequest, "Invalid request parameters")
	ErrUnauthorized   = NewAppError(http.StatusUnauthorized, "Unauthorized access")
	ErrForbidden      = NewAppError(http.StatusForbidden, "Access denied")
	ErrNotFound       = NewAppError(http.StatusNotFound, "Resource not found")
	ErrInternalServer = NewAppError(http.StatusInternalServerError, "Internal server error")
	ErrRateLimit      = NewAppError(http.StatusTooManyRequests, "Rate limit exceeded")
)

// Helper functions to create specific errors
func BadRequest(msg string) *AppError {
	return NewAppError(http.StatusBadRequest, msg)
}

func NotFound(msg string) *AppError {
	return NewAppError(http.StatusNotFound, msg)
}

func Unauthorized(msg string) *AppError {
	return NewAppError(http.StatusUnauthorized, msg)
}

func Forbidden(msg string) *AppError {
	return NewAppError(http.StatusForbidden, msg)
}

func Internal(msg string) *AppError {
	return NewAppError(http.StatusInternalServerError, msg)
}
