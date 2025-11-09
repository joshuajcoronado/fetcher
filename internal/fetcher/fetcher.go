package fetcher

import "context"

// Fetcher is the core interface that all data fetchers must implement.
// Each fetcher knows how to retrieve a specific piece of financial data
// and provides a Redis-compatible key for caching/storage.
type Fetcher interface {
	// Fetch retrieves the financial data and returns it as a float64.
	// Returns an error if the fetch operation fails.
	Fetch(ctx context.Context) (float64, error)

	// Key returns a Redis-compatible hierarchical key for this fetcher.
	// Format: fetcher:{source}:{identifier}
	// Examples:
	//   - fetcher:etherscan:eth_usd
	//   - fetcher:etherscan:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb
	//   - fetcher:alphavantage:AAPL
	//   - fetcher:rentcast:123_main_st_anytown
	Key() string
}