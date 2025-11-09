package coordinator

import (
	"context"
	"errors"
	"testing"
	"time"

	"financefetcher/internal/fetcher"
	"financefetcher/internal/testutil"
)

func TestNew(t *testing.T) {
	fetchers := []fetcher.Fetcher{
		testutil.NewMockFetcher("test:key1", 100.0, nil),
		testutil.NewMockFetcher("test:key2", 200.0, nil),
	}

	coord := New(fetchers)
	if coord == nil {
		t.Fatal("New() returned nil")
	}

	if len(coord.fetchers) != len(fetchers) {
		t.Errorf("New() created coordinator with %d fetchers, want %d", len(coord.fetchers), len(fetchers))
	}
}

func TestRun_Success(t *testing.T) {
	fetchers := []fetcher.Fetcher{
		testutil.NewMockFetcher("test:key1", 100.50, nil),
		testutil.NewMockFetcher("test:key2", 200.75, nil),
		testutil.NewMockFetcher("test:key3", 300.25, nil),
	}

	coord := New(fetchers)
	ctx := context.Background()

	// Run should complete without error
	err := coord.Run(ctx)
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
}

func TestRun_WithErrors(t *testing.T) {
	testErr := errors.New("fetch failed")

	fetchers := []fetcher.Fetcher{
		testutil.NewMockFetcher("test:key1", 100.0, nil),
		testutil.NewMockFetcher("test:key2", 0, testErr),
		testutil.NewMockFetcher("test:key3", 300.0, nil),
	}

	coord := New(fetchers)
	ctx := context.Background()

	// Run should complete without error even if some fetchers fail
	// (errors are reported per-fetcher, not at coordinator level)
	err := coord.Run(ctx)
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
}

func TestRun_NoFetchers(t *testing.T) {
	coord := New([]fetcher.Fetcher{})
	ctx := context.Background()

	err := coord.Run(ctx)
	if err == nil {
		t.Error("Run() expected error for no fetchers, got nil")
	}

	expectedErrMsg := "no fetchers configured"
	if err.Error() != expectedErrMsg {
		t.Errorf("Run() error = %q, want %q", err.Error(), expectedErrMsg)
	}
}

func TestRun_ContextCancellation(t *testing.T) {
	// Create a slow fetcher that will be cancelled
	slowFetcher := &testutil.MockFetcher{
		FetchFunc: func(ctx context.Context) (float64, error) {
			select {
			case <-ctx.Done():
				return 0, ctx.Err()
			case <-time.After(5 * time.Second):
				return 100.0, nil
			}
		},
		KeyFunc: func() string {
			return "test:slow"
		},
	}

	fetchers := []fetcher.Fetcher{slowFetcher}
	coord := New(fetchers)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Run should complete even with context cancellation
	// The fetcher will return a context error
	err := coord.Run(ctx)
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}
}

func TestRun_ConcurrentExecution(t *testing.T) {
	// Create fetchers that track execution order
	executionOrder := make(chan string, 3)

	fetcher1 := &testutil.MockFetcher{
		FetchFunc: func(ctx context.Context) (float64, error) {
			time.Sleep(50 * time.Millisecond)
			executionOrder <- "fetcher1"
			return 100.0, nil
		},
		KeyFunc: func() string {
			return "test:key1"
		},
	}

	fetcher2 := &testutil.MockFetcher{
		FetchFunc: func(ctx context.Context) (float64, error) {
			time.Sleep(30 * time.Millisecond)
			executionOrder <- "fetcher2"
			return 200.0, nil
		},
		KeyFunc: func() string {
			return "test:key2"
		},
	}

	fetcher3 := &testutil.MockFetcher{
		FetchFunc: func(ctx context.Context) (float64, error) {
			time.Sleep(10 * time.Millisecond)
			executionOrder <- "fetcher3"
			return 300.0, nil
		},
		KeyFunc: func() string {
			return "test:key3"
		},
	}

	fetchers := []fetcher.Fetcher{fetcher1, fetcher2, fetcher3}
	coord := New(fetchers)
	ctx := context.Background()

	err := coord.Run(ctx)
	if err != nil {
		t.Errorf("Run() returned unexpected error: %v", err)
	}

	close(executionOrder)

	// Verify all fetchers executed
	count := 0
	for range executionOrder {
		count++
	}

	if count != 3 {
		t.Errorf("Expected 3 fetchers to execute, got %d", count)
	}

	// Note: We don't check the order because concurrent execution
	// means fetcher3 (fastest) should complete first, demonstrating concurrency
}