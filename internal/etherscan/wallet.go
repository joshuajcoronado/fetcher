package etherscan

import (
	"context"
	"fmt"
	"math/big"
	"strconv"

	"resty.dev/v3"
)

const (
	weiPerEth = 1e18
)

// EthPriceResponse represents the Etherscan API response for ETH price
type EthPriceResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  struct {
		EthBTC          string `json:"ethbtc"`
		EthBTCTimestamp string `json:"ethbtc_timestamp"`
		EthUSD          string `json:"ethusd"`
		EthUSDTimestamp string `json:"ethusd_timestamp"`
	} `json:"result"`
}

// BalanceResponse represents the Etherscan API response for account balance
type BalanceResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
	Result  string `json:"result"` // Balance in wei as a string
}

// WalletFetcher fetches an Ethereum wallet balance in USD
type WalletFetcher struct {
	apiKey  string
	address string
	client  *resty.Client
}

// NewWalletFetcher creates a new wallet balance fetcher
func NewWalletFetcher(apiKey, address, baseURL string) *WalletFetcher {
	client := resty.New().
		SetBaseURL(baseURL).
		SetHeader("Accept", "application/json")

	return &WalletFetcher{
		apiKey:  apiKey,
		address: address,
		client:  client,
	}
}

// fetchEthPrice gets the current ETH/USD price
func (f *WalletFetcher) fetchEthPrice(ctx context.Context) (float64, error) {
	var result EthPriceResponse

	resp, err := f.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"chainid": "1",
			"module":  "stats",
			"action":  "ethprice",
			"apikey":  f.apiKey,
		}).
		SetResult(&result).
		Get("")

	if err != nil {
		return 0, fmt.Errorf("failed to fetch ETH price: %w", err)
	}

	if !resp.IsSuccess() {
		return 0, fmt.Errorf("etherscan API returned status %d", resp.StatusCode())
	}

	if result.Result.EthUSD == "" {
		return 0, fmt.Errorf("ETH price not found in response")
	}

	price, err := strconv.ParseFloat(result.Result.EthUSD, 64)
	if err != nil {
		return 0, fmt.Errorf("failed to parse ETH price: %w", err)
	}

	return price, nil
}

// Fetch retrieves the wallet balance in USD
func (f *WalletFetcher) Fetch(ctx context.Context) (float64, error) {
	// First, get the current ETH/USD price
	ethUSD, err := f.fetchEthPrice(ctx)
	if err != nil {
		return 0, err
	}

	// Then get the wallet balance in wei
	var balanceResult BalanceResponse

	resp, err := f.client.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"chainid": "1",
			"module":  "account",
			"action":  "balance",
			"address": f.address,
			"tag":     "latest",
			"apikey":  f.apiKey,
		}).
		SetResult(&balanceResult).
		Get("")

	if err != nil {
		return 0, fmt.Errorf("failed to fetch wallet balance: %w", err)
	}

	if !resp.IsSuccess() {
		return 0, fmt.Errorf("etherscan API returned status %d", resp.StatusCode())
	}

	if balanceResult.Result == "" {
		return 0, fmt.Errorf("balance not found in response")
	}

	// Convert wei (string) to big.Int, then to ETH (float64)
	weiBalance := new(big.Int)
	weiBalance, ok := weiBalance.SetString(balanceResult.Result, 10)
	if !ok {
		return 0, fmt.Errorf("failed to parse balance: %s", balanceResult.Result)
	}

	// Convert wei to ETH: divide by 10^18
	ethBalance := new(big.Float).SetInt(weiBalance)
	ethBalance.Quo(ethBalance, big.NewFloat(weiPerEth))

	// Convert to float64
	ethFloat, _ := ethBalance.Float64()

	// Calculate USD value
	usdValue := ethFloat * ethUSD

	return usdValue, nil
}

// Key returns the Redis key for this fetcher
func (f *WalletFetcher) Key() string {
	return fmt.Sprintf("fetcher:etherscan:%s", f.address)
}