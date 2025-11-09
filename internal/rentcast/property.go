package rentcast

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"financefetcher/internal/fetcher"
	"financefetcher/internal/ratelimit"

	"resty.dev/v3"
)

// SubjectProperty represents the property being valued
type SubjectProperty struct {
	ID               string   `json:"id"`
	FormattedAddress string   `json:"formattedAddress"`
	AddressLine1     string   `json:"addressLine1"`
	AddressLine2     *string  `json:"addressLine2"`
	City             string   `json:"city"`
	State            string   `json:"state"`
	StateFips        string   `json:"stateFips"`
	ZipCode          string   `json:"zipCode"`
	County           string   `json:"county"`
	CountyFips       string   `json:"countyFips"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	PropertyType     string   `json:"propertyType"`
	Bedrooms         int      `json:"bedrooms"`
	Bathrooms        float64  `json:"bathrooms"`
	SquareFootage    int      `json:"squareFootage"`
	LotSize          int      `json:"lotSize"`
	YearBuilt        int      `json:"yearBuilt"`
	LastSaleDate     *string  `json:"lastSaleDate"`
	LastSalePrice    *float64 `json:"lastSalePrice"`
}

// Comparable represents a comparable property
type Comparable struct {
	ID               string   `json:"id"`
	FormattedAddress string   `json:"formattedAddress"`
	AddressLine1     string   `json:"addressLine1"`
	AddressLine2     *string  `json:"addressLine2"`
	City             string   `json:"city"`
	State            string   `json:"state"`
	StateFips        string   `json:"stateFips"`
	ZipCode          string   `json:"zipCode"`
	County           string   `json:"county"`
	CountyFips       string   `json:"countyFips"`
	Latitude         float64  `json:"latitude"`
	Longitude        float64  `json:"longitude"`
	PropertyType     string   `json:"propertyType"`
	Bedrooms         int      `json:"bedrooms"`
	Bathrooms        float64  `json:"bathrooms"`
	SquareFootage    int      `json:"squareFootage"`
	LotSize          int      `json:"lotSize"`
	YearBuilt        int      `json:"yearBuilt"`
	Status           string   `json:"status"`
	Price            float64  `json:"price"`
	ListingType      string   `json:"listingType"`
	ListedDate       string   `json:"listedDate"`
	RemovedDate      *string  `json:"removedDate"`
	LastSeenDate     string   `json:"lastSeenDate"`
	DaysOnMarket     int      `json:"daysOnMarket"`
	Distance         float64  `json:"distance"`
	DaysOld          int      `json:"daysOld"`
	Correlation      float64  `json:"correlation"`
}

// PropertyValueResponse represents the Rentcast API response for property valuations
type PropertyValueResponse struct {
	Price           float64         `json:"price"`
	PriceRangeLow   float64         `json:"priceRangeLow"`
	PriceRangeHigh  float64         `json:"priceRangeHigh"`
	SubjectProperty SubjectProperty `json:"subjectProperty"`
	Comparables     []Comparable    `json:"comparables"`
}

// PropertyParams holds the parameters needed for a property valuation request
type PropertyParams struct {
	Address       string
	PropertyType  string
	Bedrooms      int
	Bathrooms     float64
	SquareFootage int
}

// PropertyFetcher fetches property valuations from Rentcast
type PropertyFetcher struct {
	apiKey         string
	params         PropertyParams
	client         *resty.Client
	lastResponse   *PropertyValueResponse
}

// NewPropertyFetcher creates a new property valuation fetcher
func NewPropertyFetcher(apiKey string, params PropertyParams, baseURL string) *PropertyFetcher {
	client := fetcher.NewHTTPClient(baseURL)
	client.SetHeader("X-Api-Key", apiKey)

	return &PropertyFetcher{
		apiKey: apiKey,
		params: params,
		client: client,
	}
}

// Fetch retrieves the property valuation
func (f *PropertyFetcher) Fetch(ctx context.Context) (float64, error) {
	// Apply rate limiting
	limiter := ratelimit.GetLimiter()
	if err := limiter.Wait(ctx, ratelimit.APIRentcast); err != nil {
		return 0, fetcher.NewTimeoutError(err)
	}

	slog.Debug("fetching property valuation from Rentcast", "address", f.params.Address)

	var result PropertyValueResponse

	resp, err := f.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"address":       f.params.Address,
			"propertyType":  f.params.PropertyType,
			"bedrooms":      fmt.Sprintf("%d", f.params.Bedrooms),
			"bathrooms":     fmt.Sprintf("%.1f", f.params.Bathrooms),
			"squareFootage": fmt.Sprintf("%d", f.params.SquareFootage),
		}).
		SetResult(&result).
		Get("/avm/value")

	if err != nil {
		return 0, fetcher.NewNetworkError(err)
	}

	if !resp.IsSuccess() {
		fetchErr := fetcher.ClassifyHTTPError(resp.StatusCode())
		return 0, fmt.Errorf("failed to fetch property valuation for %s: %w", f.params.Address, fetchErr)
	}

	if result.Price == 0 {
		return 0, fetcher.NewValidationError(fmt.Sprintf("price not found in response for %s", f.params.Address))
	}

	// Store the full response for later access
	f.lastResponse = &result

	return result.Price, nil
}

// GetLastResponse returns the last full API response
func (f *PropertyFetcher) GetLastResponse() *PropertyValueResponse {
	return f.lastResponse
}

// Key returns the Redis key for this fetcher
// Creates a stub from the address by replacing spaces with underscores and lowercasing
func (f *PropertyFetcher) Key() string {
	addressStub := strings.ToLower(strings.ReplaceAll(f.params.Address, " ", "_"))
	addressStub = strings.ReplaceAll(addressStub, ",", "")
	return fmt.Sprintf("fetcher:rentcast:%s", addressStub)
}