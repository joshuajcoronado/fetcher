package fetcher

import (
	"log/slog"
	"time"

	"resty.dev/v3"
)

const (
	// Default retry configuration
	defaultRetryCount       = 3
	defaultRetryWaitTime    = 1 * time.Second
	defaultRetryMaxWaitTime = 10 * time.Second
)

// NewHTTPClient creates a new HTTP client with retry logic and exponential backoff
func NewHTTPClient(baseURL string) *resty.Client {
	client := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Accept", "application/json").
		SetRetryCount(defaultRetryCount).
		SetRetryWaitTime(defaultRetryWaitTime).
		SetRetryMaxWaitTime(defaultRetryMaxWaitTime).
		AddRetryConditions(retryCondition).
		AddRetryHooks(retryHook)

	return client
}

// retryCondition determines whether a request should be retried based on the response and error
func retryCondition(r *resty.Response, err error) bool {
	// Retry on network errors
	if err != nil {
		return true
	}

	// Retry on server errors (5xx)
	if r.StatusCode() >= 500 {
		return true
	}

	// Retry on rate limit (429)
	if r.StatusCode() == 429 {
		return true
	}

	// Retry on request timeout (408)
	if r.StatusCode() == 408 {
		return true
	}

	// Don't retry on client errors (4xx except 429)
	if r.StatusCode() >= 400 && r.StatusCode() < 500 {
		return false
	}

	return false
}

// retryHook logs retry attempts for observability
func retryHook(r *resty.Response, err error) {
	if err != nil {
		slog.Debug("retrying request due to error",
			"url", r.Request.URL,
			"attempt", r.Request.Attempt,
			"error", err.Error())
		return
	}

	slog.Debug("retrying request due to status code",
		"url", r.Request.URL,
		"attempt", r.Request.Attempt,
		"status_code", r.StatusCode())
}
