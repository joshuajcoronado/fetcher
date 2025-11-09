package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// PropertyConfig holds configuration for a property to be valued.
type PropertyConfig struct {
	Address        string  `mapstructure:"address"`
	PropertyType   string  `mapstructure:"property_type"`
	Bedrooms       int     `mapstructure:"bedrooms"`
	Bathrooms      float64 `mapstructure:"bathrooms"`
	SquareFootage  int     `mapstructure:"square_footage"`
}

// Config holds all configuration for the finance fetcher application.
type Config struct {
	// API Keys for various services
	EtherscanAPIKey     string `mapstructure:"etherscan_api_key"`
	AlphavantageAPIKey  string `mapstructure:"alphavantage_api_key"`
	RentcastAPIKey      string `mapstructure:"rentcast_api_key"`
	GuidelineEmail      string `mapstructure:"guideline_email"`
	GuidelinePassword   string `mapstructure:"guideline_password"`

	// Base URLs for API endpoints (configurable for testing)
	EtherscanBaseURL     string `mapstructure:"etherscan_base_url"`
	AlphavantageBaseURL  string `mapstructure:"alphavantage_base_url"`
	RentcastBaseURL      string `mapstructure:"rentcast_base_url"`
	GuidelineBaseURL     string `mapstructure:"guideline_base_url"`

	// Items to fetch
	EthereumWallets []string          `mapstructure:"ethereum_wallets"`
	StockSymbols    []string          `mapstructure:"stock_symbols"`
	Properties      []PropertyConfig  `mapstructure:"properties"`
}

// Load reads configuration from environment variables and optional config file.
// Environment variables take precedence over config file values.
//
// Expected environment variables:
//   - ETHERSCAN_API_KEY
//   - ALPHAVANTAGE_API_KEY
//   - RENTCAST_API_KEY
//   - GUIDELINE_EMAIL
//   - GUIDELINE_PASSWORD
//   - ETHERSCAN_BASE_URL (optional, defaults to production)
//   - ALPHAVANTAGE_BASE_URL (optional, defaults to production)
//   - RENTCAST_BASE_URL (optional, defaults to production)
//   - GUIDELINE_BASE_URL (optional, defaults to production)
func Load() (*Config, error) {
	v := viper.New()

	// Set up environment variable support
	v.SetEnvPrefix("") // No prefix, use full names
	v.AutomaticEnv()

	// Set defaults for base URLs
	v.SetDefault("etherscan_base_url", "https://api.etherscan.io/v2/api")
	v.SetDefault("alphavantage_base_url", "https://www.alphavantage.co/query")
	v.SetDefault("rentcast_base_url", "https://api.rentcast.io/v1")
	v.SetDefault("guideline_base_url", "https://my.guideline.com")

	// Optionally read from config file if it exists
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("$HOME/.financefetcher")

	// Read config file (ignore if not found)
	_ = v.ReadInConfig()

	// Bind environment variables for API keys
	v.BindEnv("etherscan_api_key", "ETHERSCAN_API_KEY")
	v.BindEnv("alphavantage_api_key", "ALPHAVANTAGE_API_KEY")
	v.BindEnv("rentcast_api_key", "RENTCAST_API_KEY")
	v.BindEnv("guideline_email", "GUIDELINE_EMAIL")
	v.BindEnv("guideline_password", "GUIDELINE_PASSWORD")

	// Bind environment variables for base URLs
	v.BindEnv("etherscan_base_url", "ETHERSCAN_BASE_URL")
	v.BindEnv("alphavantage_base_url", "ALPHAVANTAGE_BASE_URL")
	v.BindEnv("rentcast_base_url", "RENTCAST_BASE_URL")
	v.BindEnv("guideline_base_url", "GUIDELINE_BASE_URL")

	// Unmarshal config into struct (handles both simple and complex fields)
	config := &Config{}
	if err := v.Unmarshal(config); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Validate required fields
	var missing []string
	if config.EtherscanAPIKey == "" {
		missing = append(missing, "ETHERSCAN_API_KEY")
	}
	if config.AlphavantageAPIKey == "" {
		missing = append(missing, "ALPHAVANTAGE_API_KEY")
	}
	if config.RentcastAPIKey == "" {
		missing = append(missing, "RENTCAST_API_KEY")
	}
	if config.GuidelineEmail == "" {
		missing = append(missing, "GUIDELINE_EMAIL")
	}
	if config.GuidelinePassword == "" {
		missing = append(missing, "GUIDELINE_PASSWORD")
	}

	if len(missing) > 0 {
		return nil, fmt.Errorf("missing required configuration: %s", strings.Join(missing, ", "))
	}

	return config, nil
}