package alphavantage

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"financefetcher/internal/fetcher"
	"financefetcher/internal/ratelimit"

	"resty.dev/v3"
)

// GlobalQuoteResponse represents the AlphaVantage API response for stock quotes
type GlobalQuoteResponse struct {
	GlobalQuote struct {
		Symbol           string `json:"01. symbol"`
		Open             string `json:"02. open"`
		High             string `json:"03. high"`
		Low              string `json:"04. low"`
		Price            string `json:"05. price"`
		Volume           string `json:"06. volume"`
		LatestTradingDay string `json:"07. latest trading day"`
		PreviousClose    string `json:"08. previous close"`
		Change           string `json:"09. change"`
		ChangePercent    string `json:"10. change percent"`
	} `json:"Global Quote"`
}

// StockFetcher fetches stock prices from AlphaVantage
type StockFetcher struct {
	apiKey string
	ticker string
	client *resty.Client
}

// NewStockFetcher creates a new stock price fetcher
func NewStockFetcher(apiKey, ticker, baseURL string) *StockFetcher {
	client := fetcher.NewHTTPClient(baseURL)

	return &StockFetcher{
		apiKey: apiKey,
		ticker: ticker,
		client: client,
	}
}

// Fetch retrieves the current stock price
func (f *StockFetcher) Fetch(ctx context.Context) (float64, error) {
	// Apply rate limiting
	limiter := ratelimit.GetLimiter()
	if err := limiter.Wait(ctx, ratelimit.APIAlphaVantage); err != nil {
		return 0, fetcher.NewTimeoutError(err)
	}

	slog.Debug("fetching stock price from AlphaVantage", "ticker", f.ticker)

	var result GlobalQuoteResponse

	resp, err := f.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"apikey":   f.apiKey,
			"function": "GLOBAL_QUOTE",
			"symbol":   f.ticker,
		}).
		SetResult(&result).
		Get("")

	if err != nil {
		return 0, fetcher.NewNetworkError(err)
	}

	if !resp.IsSuccess() {
		fetchErr := fetcher.ClassifyHTTPError(resp.StatusCode())
		return 0, fmt.Errorf("failed to fetch stock price for %s: %w", f.ticker, fetchErr)
	}

	if result.GlobalQuote.Price == "" {
		return 0, fetcher.NewValidationError(fmt.Sprintf("price not found in response for %s", f.ticker))
	}

	price, err := strconv.ParseFloat(result.GlobalQuote.Price, 64)
	if err != nil {
		return 0, fetcher.NewValidationError(fmt.Sprintf("failed to parse stock price: %v", err))
	}

	return price, nil
}

// Key returns the Redis key for this fetcher
func (f *StockFetcher) Key() string {
	return fmt.Sprintf("fetcher:alphavantage:%s", f.ticker)
}