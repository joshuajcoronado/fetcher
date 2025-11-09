package rentcast

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestNewPropertyFetcher(t *testing.T) {
	apiKey := "test_api_key"
	params := PropertyParams{
		Address:       "123 Main St, Anytown, TX 12345",
		PropertyType:  "Single Family",
		Bedrooms:      3,
		Bathrooms:     2.0,
		SquareFootage: 1500,
	}
	baseURL := "https://api.rentcast.io/v1"

	fetcher := NewPropertyFetcher(apiKey, params, baseURL)

	if fetcher == nil {
		t.Fatal("NewPropertyFetcher() returned nil")
	}

	if fetcher.apiKey != apiKey {
		t.Errorf("apiKey = %q, want %q", fetcher.apiKey, apiKey)
	}

	if fetcher.params.Address != params.Address {
		t.Errorf("params.Address = %q, want %q", fetcher.params.Address, params.Address)
	}

	if fetcher.client == nil {
		t.Error("client is nil")
	}
}

func TestPropertyFetcher_Key(t *testing.T) {
	tests := []struct {
		name        string
		address     string
		expectedKey string
	}{
		{
			name:        "simple address",
			address:     "123 Main St, Anytown, TX 12345",
			expectedKey: "fetcher:rentcast:123_main_st_anytown_tx_12345",
		},
		{
			name:        "address with multiple spaces",
			address:     "5500 Grand Lake Dr, San Antonio, TX 78244",
			expectedKey: "fetcher:rentcast:5500_grand_lake_dr_san_antonio_tx_78244",
		},
		{
			name:        "uppercase address",
			address:     "456 BROADWAY AVE, NEW YORK, NY 10001",
			expectedKey: "fetcher:rentcast:456_broadway_ave_new_york_ny_10001",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			params := PropertyParams{Address: tt.address}
			fetcher := NewPropertyFetcher("test_key", params, "http://localhost")

			if got := fetcher.Key(); got != tt.expectedKey {
				t.Errorf("Key() = %q, want %q", got, tt.expectedKey)
			}
		})
	}
}

func TestPropertyFetcher_Fetch_Success(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify API key header
		if r.Header.Get("X-Api-Key") == "" {
			t.Error("X-Api-Key header not set")
		}

		// Verify endpoint
		if r.URL.Path != "/avm/value" {
			t.Errorf("path = %q, want /avm/value", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"price": 250000.00,
			"priceRangeLow": 230000.00,
			"priceRangeHigh": 270000.00,
			"subjectProperty": {
				"id": "123",
				"formattedAddress": "123 Main St, Anytown, TX 12345",
				"addressLine1": "123 Main St",
				"city": "Anytown",
				"state": "TX",
				"stateFips": "48",
				"zipCode": "12345",
				"county": "Test County",
				"countyFips": "12345",
				"latitude": 30.0,
				"longitude": -95.0,
				"propertyType": "Single Family",
				"bedrooms": 3,
				"bathrooms": 2.0,
				"squareFootage": 1500,
				"lotSize": 5000,
				"yearBuilt": 2000
			},
			"comparables": []
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	params := PropertyParams{
		Address:       "123 Main St, Anytown, TX 12345",
		PropertyType:  "Single Family",
		Bedrooms:      3,
		Bathrooms:     2.0,
		SquareFootage: 1500,
	}

	fetcher := NewPropertyFetcher("test_key", params, server.URL)
	ctx := context.Background()

	value, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}

	expected := 250000.00
	if value != expected {
		t.Errorf("Fetch() = %.2f, want %.2f", value, expected)
	}

	// Verify last response is stored
	if fetcher.lastResponse == nil {
		t.Error("lastResponse is nil")
	} else {
		if fetcher.lastResponse.Price != expected {
			t.Errorf("lastResponse.Price = %.2f, want %.2f", fetcher.lastResponse.Price, expected)
		}
	}
}

func TestPropertyFetcher_Fetch_WithComparables(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"price": 350000.00,
			"priceRangeLow": 330000.00,
			"priceRangeHigh": 370000.00,
			"subjectProperty": {
				"id": "123",
				"formattedAddress": "456 Oak Ave, Anytown, TX 12345",
				"addressLine1": "456 Oak Ave",
				"city": "Anytown",
				"state": "TX",
				"zipCode": "12345",
				"propertyType": "Single Family",
				"bedrooms": 4,
				"bathrooms": 3.0,
				"squareFootage": 2000,
				"lotSize": 7500,
				"yearBuilt": 2010,
				"latitude": 30.5,
				"longitude": -95.5,
				"county": "Test County",
				"countyFips": "12345",
				"stateFips": "48"
			},
			"comparables": [
				{
					"id": "comp1",
					"formattedAddress": "457 Oak Ave, Anytown, TX 12345",
					"price": 340000.00,
					"distance": 0.1,
					"correlation": 0.95,
					"addressLine1": "457 Oak Ave",
					"city": "Anytown",
					"state": "TX",
					"zipCode": "12345",
					"propertyType": "Single Family",
					"bedrooms": 4,
					"bathrooms": 3.0,
					"squareFootage": 1950,
					"lotSize": 7000,
					"yearBuilt": 2009,
					"status": "Sold",
					"listingType": "For Sale",
					"listedDate": "2023-01-01",
					"lastSeenDate": "2023-02-01",
					"daysOnMarket": 30,
					"daysOld": 365,
					"latitude": 30.51,
					"longitude": -95.51,
					"county": "Test County",
					"countyFips": "12345",
					"stateFips": "48"
				}
			]
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	params := PropertyParams{
		Address:       "456 Oak Ave, Anytown, TX 12345",
		PropertyType:  "Single Family",
		Bedrooms:      4,
		Bathrooms:     3.0,
		SquareFootage: 2000,
	}

	fetcher := NewPropertyFetcher("test_key", params, server.URL)
	ctx := context.Background()

	value, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}

	expected := 350000.00
	if value != expected {
		t.Errorf("Fetch() = %.2f, want %.2f", value, expected)
	}

	// Verify comparables are stored
	if len(fetcher.lastResponse.Comparables) != 1 {
		t.Errorf("len(comparables) = %d, want 1", len(fetcher.lastResponse.Comparables))
	}
}

func TestPropertyFetcher_Fetch_HTTPError(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	params := PropertyParams{Address: "123 Main St"}
	fetcher := NewPropertyFetcher("test_key", params, server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error, got nil")
	}
}

func TestPropertyFetcher_Fetch_ZeroPrice(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"price": 0,
			"priceRangeLow": 0,
			"priceRangeHigh": 0
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	params := PropertyParams{Address: "123 Main St"}
	fetcher := NewPropertyFetcher("test_key", params, server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for zero price, got nil")
	}
}

func TestPropertyFetcher_Fetch_MissingPrice(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"priceRangeLow": 200000.00,
			"priceRangeHigh": 300000.00
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	params := PropertyParams{Address: "123 Main St"}
	fetcher := NewPropertyFetcher("test_key", params, server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for missing price, got nil")
	}
}

func TestPropertyFetcher_Fetch_ContextCancellation(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Server will be slow to respond
		<-r.Context().Done()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	params := PropertyParams{Address: "123 Main St"}
	fetcher := NewPropertyFetcher("test_key", params, server.URL)

	// Create a context that is already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := fetcher.Fetch(ctx)
	if err == nil {
		t.Error("Fetch() expected error for cancelled context, got nil")
	}
}

func TestPropertyFetcher_Fetch_VerifyQueryParams(t *testing.T) {
	params := PropertyParams{
		Address:       "789 Elm St, Testville, CA 90210",
		PropertyType:  "Condo",
		Bedrooms:      2,
		Bathrooms:     1.5,
		SquareFootage: 1200,
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify all query parameters
		q := r.URL.Query()

		if got := q.Get("address"); got != params.Address {
			t.Errorf("address = %q, want %q", got, params.Address)
		}
		if got := q.Get("propertyType"); got != params.PropertyType {
			t.Errorf("propertyType = %q, want %q", got, params.PropertyType)
		}
		if got := q.Get("bedrooms"); got != "2" {
			t.Errorf("bedrooms = %q, want 2", got)
		}
		if got := q.Get("bathrooms"); got != "1.5" {
			t.Errorf("bathrooms = %q, want 1.5", got)
		}
		if got := q.Get("squareFootage"); got != "1200" {
			t.Errorf("squareFootage = %q, want 1200", got)
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"price": 180000.00}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher := NewPropertyFetcher("test_key", params, server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}
}

func TestPropertyFetcher_GetLastResponse(t *testing.T) {
	params := PropertyParams{Address: "123 Main St"}
	fetcher := NewPropertyFetcher("test_key", params, "http://localhost")

	// Initially should be nil
	if fetcher.GetLastResponse() != nil {
		t.Error("GetLastResponse() should be nil before Fetch()")
	}

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{
			"price": 300000.00,
			"priceRangeLow": 280000.00,
			"priceRangeHigh": 320000.00
		}`))
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	fetcher = NewPropertyFetcher("test_key", params, server.URL)
	ctx := context.Background()

	_, err := fetcher.Fetch(ctx)
	if err != nil {
		t.Fatalf("Fetch() returned unexpected error: %v", err)
	}

	// After Fetch, should not be nil
	lastResp := fetcher.GetLastResponse()
	if lastResp == nil {
		t.Fatal("GetLastResponse() is nil after Fetch()")
	}

	if lastResp.Price != 300000.00 {
		t.Errorf("GetLastResponse().Price = %.2f, want 300000.00", lastResp.Price)
	}
	if lastResp.PriceRangeLow != 280000.00 {
		t.Errorf("GetLastResponse().PriceRangeLow = %.2f, want 280000.00", lastResp.PriceRangeLow)
	}
	if lastResp.PriceRangeHigh != 320000.00 {
		t.Errorf("GetLastResponse().PriceRangeHigh = %.2f, want 320000.00", lastResp.PriceRangeHigh)
	}
}