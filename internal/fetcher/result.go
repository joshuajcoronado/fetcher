package fetcher

// Result represents the outcome of a fetch operation.
// It's designed to be sent through channels from worker goroutines
// to a coordinator that processes and stores the results.
type Result struct {
	// Key is the Redis-compatible hierarchical key for this data point
	Key string

	// Value is the fetched financial data (price, balance, valuation, etc.)
	Value float64

	// Error contains any error that occurred during the fetch operation.
	// If Error is not nil, Value should be considered invalid.
	Error error
}