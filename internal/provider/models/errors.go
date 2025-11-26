package models

import (
	"errors"
	"fmt"
	"time"
)

// Sentinel errors for common provider failures.
var (
	// Context/Token errors
	ErrContextLengthExceeded = errors.New("context length exceeded")

	// Safety/Content errors
	ErrContentBlocked = errors.New("content blocked by safety filters")

	// Rate limiting errors
	ErrRateLimit     = errors.New("rate limit exceeded")
	ErrQuotaExceeded = errors.New("quota exceeded")

	// Model errors
	ErrInvalidModel = errors.New("invalid model")

	// Authentication errors
	ErrAuthentication   = errors.New("authentication failed")
	ErrPermissionDenied = errors.New("permission denied")

	// Network errors
	ErrNetwork            = errors.New("network error")
	ErrTimeout            = errors.New("request timeout")
	ErrServiceUnavailable = errors.New("service unavailable")

	// Feature errors
	ErrToolCallingNotSupported = errors.New("tool calling not supported")
	ErrStreamingNotSupported   = errors.New("streaming not supported")

	// Request errors
	ErrInvalidRequest = errors.New("invalid request")
)

// ErrorCode represents a provider error code.
type ErrorCode string

const (
	ErrorCodeContextLength  ErrorCode = "context_length_exceeded"
	ErrorCodeContentBlocked ErrorCode = "content_blocked"
	ErrorCodeRateLimit      ErrorCode = "rate_limit"
	ErrorCodeQuota          ErrorCode = "quota_exceeded"
	ErrorCodeInvalidModel   ErrorCode = "invalid_model"
	ErrorCodeAuth           ErrorCode = "authentication_failed"
	ErrorCodePermission     ErrorCode = "permission_denied"
	ErrorCodeNetwork        ErrorCode = "network_error"
	ErrorCodeTimeout        ErrorCode = "timeout"
	ErrorCodeUnavailable    ErrorCode = "service_unavailable"
	ErrorCodeTooling        ErrorCode = "tool_calling_not_supported"
	ErrorCodeStreaming      ErrorCode = "streaming_not_supported"
	ErrorCodeInvalidRequest ErrorCode = "invalid_request"
)

// ProviderError wraps errors with additional context.
type ProviderError struct {
	Code       ErrorCode
	Message    string
	Underlying error
	Retryable  bool
	RetryAfter *time.Duration
}

// Error implements the error interface.
func (e *ProviderError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("%s: %s (%v)", e.Code, e.Message, e.Underlying)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Unwrap returns the underlying error.
func (e *ProviderError) Unwrap() error {
	return e.Underlying
}

// IsRetryable returns true if the error is retryable.
func IsRetryable(err error) bool {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.Retryable
	}
	return false
}

// GetRetryAfter returns the retry-after duration if present.
func GetRetryAfter(err error) *time.Duration {
	var providerErr *ProviderError
	if errors.As(err, &providerErr) {
		return providerErr.RetryAfter
	}
	return nil
}
