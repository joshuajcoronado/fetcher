package coordinator

import (
	"context"
	"fmt"
	"sync"

	"financefetcher/internal/fetcher"
)

// Coordinator manages concurrent fetchers and aggregates results
type Coordinator struct {
	fetchers []fetcher.Fetcher
}

// New creates a new Coordinator with the given fetchers
func New(fetchers []fetcher.Fetcher) *Coordinator {
	return &Coordinator{
		fetchers: fetchers,
	}
}

// Run executes all fetchers concurrently and prints results to stdout
// Each fetcher runs in its own goroutine and sends results to a shared channel
// Results are printed as they arrive in the format:
//   - Success: "KEY: $VALUE"
//   - Error: "KEY: ERROR - error message"
func (c *Coordinator) Run(ctx context.Context) error {
	if len(c.fetchers) == 0 {
		return fmt.Errorf("no fetchers configured")
	}

	// Create a channel for collecting results
	resultChan := make(chan fetcher.Result, len(c.fetchers))

	// WaitGroup to track all worker goroutines
	var wg sync.WaitGroup

	// Launch a goroutine for each fetcher
	for _, f := range c.fetchers {
		wg.Add(1)
		go func(ft fetcher.Fetcher) {
			defer wg.Done()

			// Execute the fetch operation
			value, err := ft.Fetch(ctx)

			// Send result to the channel
			resultChan <- fetcher.Result{
				Key:   ft.Key(),
				Value: value,
				Error: err,
			}
		}(f)
	}

	// Close the result channel when all workers are done
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect and print results as they arrive
	for result := range resultChan {
		if result.Error != nil {
			fmt.Printf("%s: ERROR - %v\n", result.Key, result.Error)
		} else {
			fmt.Printf("%s: $%.2f\n", result.Key, result.Value)
		}
	}

	return nil
}