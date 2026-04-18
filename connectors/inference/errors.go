package inference

import "fmt"

// InferenceErrorCode classifies inference failures.
type InferenceErrorCode string

const (
	ErrRateLimit          InferenceErrorCode = "rate_limit"
	ErrContentPolicy      InferenceErrorCode = "content_policy"
	ErrInvalidRequest     InferenceErrorCode = "invalid_request"
	ErrAuthentication     InferenceErrorCode = "authentication"
	ErrProviderUnavail    InferenceErrorCode = "provider_unavailable"
	ErrModelNotFound      InferenceErrorCode = "model_not_found"
	ErrUnknown            InferenceErrorCode = "unknown"
)

// InferenceError is a normalized error returned by any inference provider.
type InferenceError struct {
	Code       InferenceErrorCode
	Provider   string
	StatusCode int
	Message    string
	Retryable  bool
	Cause      error
}

func (e *InferenceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("inference [%s/%s] %s: %v", e.Provider, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("inference [%s/%s] %s", e.Provider, e.Code, e.Message)
}

func (e *InferenceError) Unwrap() error {
	return e.Cause
}
