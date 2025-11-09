package testutil

import (
	"context"
	"financefetcher/internal/fetcher"
)

// MockFetcher is a mock implementation of the Fetcher interface for testing
type MockFetcher struct {
	FetchFunc func(ctx context.Context) (float64, error)
	KeyFunc   func() string
}

// Fetch implements the Fetcher interface
func (m *MockFetcher) Fetch(ctx context.Context) (float64, error) {
	if m.FetchFunc != nil {
		return m.FetchFunc(ctx)
	}
	return 0, nil
}

// Key implements the Fetcher interface
func (m *MockFetcher) Key() string {
	if m.KeyFunc != nil {
		return m.KeyFunc()
	}
	return "mock:key"
}

// NewMockFetcher creates a simple mock fetcher with predefined values
func NewMockFetcher(key string, value float64, err error) fetcher.Fetcher {
	return &MockFetcher{
		FetchFunc: func(ctx context.Context) (float64, error) {
			return value, err
		},
		KeyFunc: func() string {
			return key
		},
	}
}