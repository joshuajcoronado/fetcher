package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"financefetcher/internal/alphavantage"
	"financefetcher/internal/coordinator"
	"financefetcher/internal/etherscan"
	"financefetcher/internal/fetcher"
	"financefetcher/internal/rentcast"
)

// TestIntegration_AllFetchers tests the full flow with all fetchers using mock HTTP servers
func TestIntegration_AllFetchers(t *testing.T) {
	// Create mock Etherscan server
	etherscanServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if action == "ethprice" {
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": {
					"ethusd": "2500.00"
				}
			}`))
		} else if action == "balance" {
			// 10 ETH = 10000000000000000000 wei
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": "10000000000000000000"
			}`))
		}
	}))
	defer etherscanServer.Close()

	// Create mock AlphaVantage server
	alphavantageServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		symbol := r.URL.Query().Get("symbol")
		price := "100.00"

		switch symbol {
		case "AAPL":
			price = "178.23"
		case "GOOGL":
			price = "142.56"
		case "MSFT":
			price = "378.91"
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "` + symbol + `",
				"05. price": "` + price + `"
			}
		}`))
	}))
	defer alphavantageServer.Close()

	// Create mock Rentcast server
	rentcastServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"price": 250000.00,
			"priceRangeLow": 230000.00,
			"priceRangeHigh": 270000.00,
			"subjectProperty": {
				"id": "123",
				"formattedAddress": "5500 Grand Lake Dr, San Antonio, TX 78244",
				"addressLine1": "5500 Grand Lake Dr",
				"city": "San Antonio",
				"state": "TX",
				"zipCode": "78244",
				"propertyType": "Single Family",
				"bedrooms": 3,
				"bathrooms": 2.0,
				"squareFootage": 1878,
				"lotSize": 5000,
				"yearBuilt": 2000,
				"latitude": 29.0,
				"longitude": -98.0,
				"county": "Bexar",
				"countyFips": "48029",
				"stateFips": "48"
			},
			"comparables": []
		}`))
	}))
	defer rentcastServer.Close()

	// Create fetchers using mock servers
	fetchers := []fetcher.Fetcher{
		etherscan.NewWalletFetcher(
			"test_etherscan_key",
			"0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
			etherscanServer.URL,
		),
		alphavantage.NewStockFetcher(
			"test_alphavantage_key",
			"AAPL",
			alphavantageServer.URL,
		),
		alphavantage.NewStockFetcher(
			"test_alphavantage_key",
			"GOOGL",
			alphavantageServer.URL,
		),
		alphavantage.NewStockFetcher(
			"test_alphavantage_key",
			"MSFT",
			alphavantageServer.URL,
		),
		rentcast.NewPropertyFetcher(
			"test_rentcast_key",
			rentcast.PropertyParams{
				Address:       "5500 Grand Lake Dr, San Antonio, TX 78244",
				PropertyType:  "Single Family",
				Bedrooms:      3,
				Bathrooms:     2,
				SquareFootage: 1878,
			},
			rentcastServer.URL,
		),
	}

	// Create coordinator and run
	coord := coordinator.New(fetchers)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := coord.Run(ctx)
	if err != nil {
		t.Fatalf("coordinator.Run() failed: %v", err)
	}
}

// TestIntegration_ConcurrentFetching tests that fetchers run concurrently
func TestIntegration_ConcurrentFetching(t *testing.T) {
	// Create a server that introduces delays
	slowServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Each request takes 100ms
		time.Sleep(100 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "TEST",
				"05. price": "100.00"
			}
		}`))
	}))
	defer slowServer.Close()

	// Create multiple fetchers
	numFetchers := 5
	fetchers := make([]fetcher.Fetcher, numFetchers)
	for i := 0; i < numFetchers; i++ {
		fetchers[i] = alphavantage.NewStockFetcher(
			"test_key",
			"TEST",
			slowServer.URL,
		)
	}

	// Create coordinator and run
	coord := coordinator.New(fetchers)
	ctx := context.Background()

	start := time.Now()
	err := coord.Run(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("coordinator.Run() failed: %v", err)
	}

	// If fetchers ran sequentially, it would take 500ms (5 * 100ms)
	// If concurrent, should be closer to 100ms
	// We'll check that it's less than 300ms to account for overhead
	if duration > 300*time.Millisecond {
		t.Errorf("Fetchers likely ran sequentially. Duration: %v (expected < 300ms)", duration)
	}
}

// TestIntegration_PartialFailures tests that the system handles partial failures gracefully
func TestIntegration_PartialFailures(t *testing.T) {
	requestCount := 0

	// Create a server that fails for some requests
	mixedServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++

		// First request succeeds, second fails, third succeeds
		if requestCount%2 == 0 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "TEST",
				"05. price": "100.00"
			}
		}`))
	}))
	defer mixedServer.Close()

	// Create multiple fetchers
	fetchers := []fetcher.Fetcher{
		alphavantage.NewStockFetcher("test_key", "TEST1", mixedServer.URL),
		alphavantage.NewStockFetcher("test_key", "TEST2", mixedServer.URL),
		alphavantage.NewStockFetcher("test_key", "TEST3", mixedServer.URL),
	}

	// Create coordinator and run
	coord := coordinator.New(fetchers)
	ctx := context.Background()

	// Run should complete without error even if some fetchers fail
	err := coord.Run(ctx)
	if err != nil {
		t.Fatalf("coordinator.Run() failed: %v", err)
	}
}

// TestIntegration_ContextTimeout tests that context timeout is respected
func TestIntegration_ContextTimeout(t *testing.T) {
	// Create a server that never responds
	hangingServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Wait for context cancellation
		<-r.Context().Done()
	}))
	defer hangingServer.Close()

	fetchers := []fetcher.Fetcher{
		alphavantage.NewStockFetcher("test_key", "TEST", hangingServer.URL),
	}

	coord := coordinator.New(fetchers)

	// Create context with very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	start := time.Now()
	err := coord.Run(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("coordinator.Run() failed: %v", err)
	}

	// Should complete quickly due to timeout, not hang forever
	if duration > 200*time.Millisecond {
		t.Errorf("Context timeout not respected. Duration: %v", duration)
	}
}

// TestIntegration_RealWorldScenario simulates a real-world usage scenario
func TestIntegration_RealWorldScenario(t *testing.T) {
	// This test simulates fetching data for a user's portfolio:
	// - 1 crypto wallet
	// - 3 stocks
	// - 1 property

	// Create realistic mock servers
	etherscanServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate realistic response time
		time.Sleep(50 * time.Millisecond)

		action := r.URL.Query().Get("action")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)

		if action == "ethprice" {
			w.Write([]byte(`{"status": "1", "message": "OK", "result": {"ethusd": "3200.00"}}`))
		} else {
			w.Write([]byte(`{"status": "1", "message": "OK", "result": "5000000000000000000"}`)) // 5 ETH
		}
	}))
	defer etherscanServer.Close()

	stockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(30 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"Global Quote": {"05. price": "150.00"}}`))
	}))
	defer stockServer.Close()

	propertyServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"price": 450000.00}`))
	}))
	defer propertyServer.Close()

	// Create realistic portfolio
	fetchers := []fetcher.Fetcher{
		etherscan.NewWalletFetcher("key", "0xabc", etherscanServer.URL),
		alphavantage.NewStockFetcher("key", "AAPL", stockServer.URL),
		alphavantage.NewStockFetcher("key", "GOOGL", stockServer.URL),
		alphavantage.NewStockFetcher("key", "MSFT", stockServer.URL),
		rentcast.NewPropertyFetcher("key", rentcast.PropertyParams{Address: "123 Main St"}, propertyServer.URL),
	}

	coord := coordinator.New(fetchers)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	err := coord.Run(ctx)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("coordinator.Run() failed: %v", err)
	}

	// With concurrency, should complete in ~100ms (slowest fetcher)
	// rather than ~260ms (sum of all delays)
	t.Logf("Portfolio fetch completed in %v", duration)

	// Verify it completed reasonably quickly
	if duration > 500*time.Millisecond {
		t.Errorf("Fetch took too long: %v (expected < 500ms with concurrency)", duration)
	}
}