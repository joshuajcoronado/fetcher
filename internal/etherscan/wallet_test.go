package etherscan

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewWalletFetcher(t *testing.T) {
	apiKey := "test_api_key"
	address := "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"
	baseURL := "https://api.etherscan.io/v2/api"

	fetcher := NewWalletFetcher(apiKey, address, baseURL)

	if fetcher == nil {
		t.Fatal("NewWalletFetcher() returned nil")
	}

	if fetcher.apiKey != apiKey {
		t.Errorf("apiKey = %q, want %q", fetcher.apiKey, apiKey)
	}

	if fetcher.address != address {
		t.Errorf("address = %q, want %q", fetcher.address, address)
	}

	if fetcher.client == nil {
		t.Error("client is nil")
	}
}

func TestWalletFetcher_Key(t *testing.T) {
	address := "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb"
	fetcher := NewWalletFetcher("test_key", address, "http://localhost")

	expectedKey := "fetcher:etherscan:" + address
	if got := fetcher.Key(); got != expectedKey {
		t.Errorf("Key() = %q, want %q", got, expectedKey)
	}
}

func TestWalletFetcher_Fetch_Success(t *testing.T) {
	// Create a mock server
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check which endpoint is being called
		action := r.URL.Query().Get("action")

		if action == "ethprice" {
			// Return mock ETH price response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": {
					"ethbtc": "0.05",
					"ethbtc_timestamp": "1234567890",
					"ethusd": "2000.50",
					"ethusd_timestamp": "1234567890"
				}
			}`))
		} else if action == "balance" {
			// Return mock balance response (1 ETH = 1000000000000000000 wei)
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": "1000000000000000000"
			}`))
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)
	ctx := context.Background()

	value, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}

	// Expected: 1 ETH * $2000.50 = $2000.50
	expected := 2000.50
	if value != expected {
		t.Errorf("Fetch() = %.2f, want %.2f", value, expected)
	}
}

func TestWalletFetcher_Fetch_LargeBalance(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")

		if action == "ethprice" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": {
					"ethusd": "3500.00"
				}
			}`))
		} else if action == "balance" {
			// 100 ETH = 100000000000000000000 wei
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": "100000000000000000000"
			}`))
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)
	ctx := context.Background()

	value, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}

	// Expected: 100 ETH * $3500.00 = $350,000.00
	expected := 350000.00
	if value != expected {
		t.Errorf("Fetch() = %.2f, want %.2f", value, expected)
	}
}

func TestWalletFetcher_Fetch_EthPriceError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error, got nil")
	}
}

func TestWalletFetcher_Fetch_MissingEthPrice(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"status": "1",
			"message": "OK",
			"result": {}
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for missing ETH price, got nil")
	}
}

func TestWalletFetcher_Fetch_BalanceError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")

		if action == "ethprice" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": {
					"ethusd": "2000.00"
				}
			}`))
		} else if action == "balance" {
			// Return error for balance request
			w.WriteHeader(http.StatusInternalServerError)
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for balance request, got nil")
	}
}

func TestWalletFetcher_Fetch_InvalidBalance(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")

		if action == "ethprice" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": {
					"ethusd": "2000.00"
				}
			}`))
		} else if action == "balance" {
			// Return invalid balance
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": "invalid_number"
			}`))
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for invalid balance, got nil")
	}
}

func TestWalletFetcher_Fetch_ZeroBalance(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		action := r.URL.Query().Get("action")

		if action == "ethprice" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": {
					"ethusd": "2000.00"
				}
			}`))
		} else if action == "balance" {
			// Zero balance
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "1",
				"message": "OK",
				"result": "0"
			}`))
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)
	ctx := context.Background()

	value, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}

	if value != 0 {
		t.Errorf("Fetch() = %.2f, want 0.00", value)
	}
}

func TestWalletFetcher_Fetch_ContextCancellation(t *testing.T) {
	// Create a server that doesn't respond quickly
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server will be slow to respond
		<-r.Context().Done()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewWalletFetcher("test_key", "0x123", server.URL)

	// Create a context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for cancelled context, got nil")
	}
}