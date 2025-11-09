package fetcher

import (
	"fmt"
)

// ErrorType represents the category of error that occurred during a fetch operation
type ErrorType string

const (
	// ErrorTypeNetwork indicates a network-level error (connection refused, DNS, etc.)
	ErrorTypeNetwork ErrorType = "network"
	// ErrorTypeRateLimit indicates the request was rejected due to rate limiting (HTTP 429)
	ErrorTypeRateLimit ErrorType = "rate_limit"
	// ErrorTypeServer indicates a server error (HTTP 5xx)
	ErrorTypeServer ErrorType = "server"
	// ErrorTypeClient indicates a client error (HTTP 4xx except 429)
	ErrorTypeClient ErrorType = "client"
	// ErrorTypeValidation indicates the response was received but data validation failed
	ErrorTypeValidation ErrorType = "validation"
	// ErrorTypeTimeout indicates the request timed out
	ErrorTypeTimeout ErrorType = "timeout"
	// ErrorTypeUnknown indicates an error of unknown type
	ErrorTypeUnknown ErrorType = "unknown"
)

// FetchError represents a structured error from a fetch operation
type FetchError struct {
	Type       ErrorType
	Retryable  bool
	StatusCode int
	Message    string
	Cause      error
}

// Error implements the error interface
func (e *FetchError) Error() string {
	if e.StatusCode > 0 {
		return fmt.Sprintf("%s error (status %d): %s", e.Type, e.StatusCode, e.Message)
	}
	return fmt.Sprintf("%s error: %s", e.Type, e.Message)
}

// Unwrap implements error unwrapping for errors.Is and errors.As
func (e *FetchError) Unwrap() error {
	return e.Cause
}

// NewNetworkError creates a network error
func NewNetworkError(cause error) *FetchError {
	return &FetchError{
		Type:      ErrorTypeNetwork,
		Retryable: true,
		Message:   "network request failed",
		Cause:     cause,
	}
}

// NewRateLimitError creates a rate limit error
func NewRateLimitError(statusCode int) *FetchError {
	return &FetchError{
		Type:       ErrorTypeRateLimit,
		Retryable:  true,
		StatusCode: statusCode,
		Message:    "rate limit exceeded",
	}
}

// NewServerError creates a server error
func NewServerError(statusCode int) *FetchError {
	return &FetchError{
		Type:       ErrorTypeServer,
		Retryable:  true,
		StatusCode: statusCode,
		Message:    "server returned an error",
	}
}

// NewClientError creates a client error
func NewClientError(statusCode int, message string) *FetchError {
	return &FetchError{
		Type:       ErrorTypeClient,
		Retryable:  false,
		StatusCode: statusCode,
		Message:    message,
	}
}

// NewValidationError creates a validation error
func NewValidationError(message string) *FetchError {
	return &FetchError{
		Type:      ErrorTypeValidation,
		Retryable: false,
		Message:   message,
	}
}

// NewTimeoutError creates a timeout error
func NewTimeoutError(cause error) *FetchError {
	return &FetchError{
		Type:      ErrorTypeTimeout,
		Retryable: true,
		Message:   "request timed out",
		Cause:     cause,
	}
}

// ClassifyHTTPError classifies an HTTP status code into an appropriate FetchError
func ClassifyHTTPError(statusCode int) *FetchError {
	switch {
	case statusCode == 429:
		return NewRateLimitError(statusCode)
	case statusCode >= 500:
		return NewServerError(statusCode)
	case statusCode >= 400:
		return NewClientError(statusCode, fmt.Sprintf("client error: HTTP %d", statusCode))
	default:
		return &FetchError{
			Type:       ErrorTypeUnknown,
			Retryable:  false,
			StatusCode: statusCode,
			Message:    fmt.Sprintf("unexpected status code: %d", statusCode),
		}
	}
}