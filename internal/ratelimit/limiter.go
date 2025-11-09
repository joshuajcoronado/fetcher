package ratelimit

import (
	"context"
	"os"
	"sync"

	"golang.org/x/time/rate"
)

// API represents the different external APIs we interact with
type API string

const (
	// APIEtherscan represents the Etherscan API
	APIEtherscan API = "etherscan"
	// APIAlphaVantage represents the AlphaVantage API
	APIAlphaVantage API = "alphavantage"
	// APIRentcast represents the Rentcast API
	APIRentcast API = "rentcast"
)

// Limiter manages rate limits for different APIs
type Limiter struct {
	limiters map[API]*rate.Limiter
	mu       sync.RWMutex
}

var (
	instance *Limiter
	once     sync.Once
)

// GetLimiter returns the singleton rate limiter instance
func GetLimiter() *Limiter {
	once.Do(func() {
		instance = &Limiter{
			limiters: make(map[API]*rate.Limiter),
		}
		instance.initLimiters()
	})
	return instance
}

// initLimiters initializes rate limiters for each API with conservative defaults
func (l *Limiter) initLimiters() {
	// In test mode, use unlimited rate limits to avoid slowing down tests
	// Check for GO_TESTING environment variable or if we're running tests
	if os.Getenv("GO_TESTING") == "1" || isTestMode() {
		// Use rate.Inf for unlimited rate limiting in tests
		l.limiters[APIEtherscan] = rate.NewLimiter(rate.Inf, 1)
		l.limiters[APIAlphaVantage] = rate.NewLimiter(rate.Inf, 1)
		l.limiters[APIRentcast] = rate.NewLimiter(rate.Inf, 1)
		return
	}

	// Production rate limits
	// Etherscan: 4 requests per second (conservative, actual limit may be higher)
	l.limiters[APIEtherscan] = rate.NewLimiter(rate.Limit(4), 1)

	// AlphaVantage: 5 requests per minute on free tier = 1 request every 12 seconds
	// We use a rate of 1/12 requests per second
	l.limiters[APIAlphaVantage] = rate.NewLimiter(rate.Limit(1.0/12.0), 1)

	// Rentcast: 10 requests per second (conservative estimate)
	l.limiters[APIRentcast] = rate.NewLimiter(rate.Limit(10), 1)
}

// isTestMode checks if we're running in test mode
func isTestMode() bool {
	// Check if the test binary is running by looking for test-related arguments
	for _, arg := range os.Args {
		if len(arg) > 6 && arg[:6] == "-test." {
			return true
		}
	}
	return false
}

// Wait blocks until the rate limiter permits an event for the given API
// It returns an error if the context is canceled before the event can proceed
func (l *Limiter) Wait(ctx context.Context, api API) error {
	l.mu.RLock()
	limiter, exists := l.limiters[api]
	l.mu.RUnlock()

	if !exists {
		// If no limiter exists for this API, allow the request without limiting
		return nil
	}

	return limiter.Wait(ctx)
}

// Allow reports whether an event for the given API may happen now
func (l *Limiter) Allow(api API) bool {
	l.mu.RLock()
	limiter, exists := l.limiters[api]
	l.mu.RUnlock()

	if !exists {
		// If no limiter exists for this API, allow the request
		return true
	}

	return limiter.Allow()
}