package alphavantage

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewStockFetcher(t *testing.T) {
	apiKey := "test_api_key"
	ticker := "AAPL"
	baseURL := "https://www.alphavantage.co/query"

	fetcher := NewStockFetcher(apiKey, ticker, baseURL)

	if fetcher == nil {
		t.Fatal("NewStockFetcher() returned nil")
	}

	if fetcher.apiKey != apiKey {
		t.Errorf("apiKey = %q, want %q", fetcher.apiKey, apiKey)
	}

	if fetcher.ticker != ticker {
		t.Errorf("ticker = %q, want %q", fetcher.ticker, ticker)
	}

	if fetcher.client == nil {
		t.Error("client is nil")
	}
}

func TestStockFetcher_Key(t *testing.T) {
	tests := []struct {
		ticker      string
		expectedKey string
	}{
		{"AAPL", "fetcher:alphavantage:AAPL"},
		{"GOOGL", "fetcher:alphavantage:GOOGL"},
		{"MSFT", "fetcher:alphavantage:MSFT"},
	}

	for _, tt := range tests {
		t.Run(tt.ticker, func(t *testing.T) {
			fetcher := NewStockFetcher("test_key", tt.ticker, "http://localhost")
			if got := fetcher.Key(); got != tt.expectedKey {
				t.Errorf("Key() = %q, want %q", got, tt.expectedKey)
			}
		})
	}
}

func TestStockFetcher_Fetch_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify query parameters
		if r.URL.Query().Get("function") != "GLOBAL_QUOTE" {
			t.Errorf("function = %q, want GLOBAL_QUOTE", r.URL.Query().Get("function"))
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "AAPL",
				"02. open": "175.50",
				"03. high": "178.75",
				"04. low": "174.25",
				"05. price": "178.23",
				"06. volume": "50000000",
				"07. latest trading day": "2024-01-15",
				"08. previous close": "176.50",
				"09. change": "1.73",
				"10. change percent": "0.98%"
			}
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher("test_key", "AAPL", server.URL)
	ctx := context.Background()

	value, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}

	expected := 178.23
	if value != expected {
		t.Errorf("Fetch() = %.2f, want %.2f", value, expected)
	}
}

func TestStockFetcher_Fetch_DifferentStocks(t *testing.T) {
	tests := []struct {
		ticker string
		price  string
		want   float64
	}{
		{"AAPL", "178.23", 178.23},
		{"GOOGL", "142.56", 142.56},
		{"MSFT", "378.91", 378.91},
		{"TSLA", "250.00", 250.00},
	}

	for _, tt := range tests {
		t.Run(tt.ticker, func(t *testing.T) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"Global Quote": {
						"01. symbol": "` + tt.ticker + `",
						"05. price": "` + tt.price + `"
					}
				}`))
			})

			server := httptest.NewServer(handler)
			defer server.Close()

			fetcher := NewStockFetcher("test_key", tt.ticker, server.URL)
			ctx := context.Background()

			value, err := fetcher.Fetch(ctx)
			if err != nil {
				t.Fatalf("Fetch() returned unexpected error: %v", err)
			}

			if value != tt.want {
				t.Errorf("Fetch() = %.2f, want %.2f", value, tt.want)
			}
		})
	}
}

func TestStockFetcher_Fetch_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher("test_key", "AAPL", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error, got nil")
	}
}

func TestStockFetcher_Fetch_MissingPrice(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "AAPL"
			}
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher("test_key", "AAPL", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for missing price, got nil")
	}

	expectedErrMsg := "validation error: price not found in response for AAPL"
	if err.Error() != expectedErrMsg {
		t.Errorf("Fetch() error = %q, want %q", err.Error(), expectedErrMsg)
	}
}

func TestStockFetcher_Fetch_InvalidPrice(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "AAPL",
				"05. price": "invalid_number"
			}
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher("test_key", "AAPL", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for invalid price, got nil")
	}
}

func TestStockFetcher_Fetch_EmptyResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher("test_key", "AAPL", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for empty response, got nil")
	}
}

func TestStockFetcher_Fetch_RateLimitResponse(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Note": "Thank you for using Alpha Vantage! Our standard API call frequency is 5 calls per minute."
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher("test_key", "AAPL", server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for rate limit response, got nil")
	}
}

func TestStockFetcher_Fetch_ContextCancellation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server will be slow to respond
		<-r.Context().Done()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher("test_key", "AAPL", server.URL)

	// Create a context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for cancelled context, got nil")
	}
}

func TestStockFetcher_Fetch_VerifyQueryParams(t *testing.T) {
	apiKey := "test_api_key_123"
	ticker := "GOOGL"

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all query parameters
		if got := r.URL.Query().Get("apikey"); got != apiKey {
			t.Errorf("apikey = %q, want %q", got, apiKey)
		}
		if got := r.URL.Query().Get("function"); got != "GLOBAL_QUOTE" {
			t.Errorf("function = %q, want GLOBAL_QUOTE", got)
		}
		if got := r.URL.Query().Get("symbol"); got != ticker {
			t.Errorf("symbol = %q, want %q", got, ticker)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"Global Quote": {
				"01. symbol": "GOOGL",
				"05. price": "142.56"
			}
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewStockFetcher(apiKey, ticker, server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}
}