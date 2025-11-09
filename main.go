package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"financefetcher/internal/alphavantage"
	"financefetcher/internal/config"
	"financefetcher/internal/coordinator"
	"financefetcher/internal/etherscan"
	"financefetcher/internal/fetcher"
	"financefetcher/internal/rentcast"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Create context with cancellation for graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle interrupt signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sigChan
		fmt.Println("\nReceived interrupt signal, shutting down...")
		cancel()
	}()

	// Create fetchers dynamically from configuration
	var fetchers []fetcher.Fetcher

	// Create Ethereum wallet fetchers
	for _, wallet := range cfg.EthereumWallets {
		fetchers = append(fetchers, etherscan.NewWalletFetcher(
			cfg.EtherscanAPIKey,
			wallet,
			cfg.EtherscanBaseURL,
		))
	}

	// Create stock fetchers
	for _, symbol := range cfg.StockSymbols {
		fetchers = append(fetchers, alphavantage.NewStockFetcher(
			cfg.AlphavantageAPIKey,
			symbol,
			cfg.AlphavantageBaseURL,
		))
	}

	// Create property fetchers
	for _, prop := range cfg.Properties {
		fetchers = append(fetchers, rentcast.NewPropertyFetcher(
			cfg.RentcastAPIKey,
			rentcast.PropertyParams{
				Address:       prop.Address,
				PropertyType:  prop.PropertyType,
				Bedrooms:      prop.Bedrooms,
				Bathrooms:     prop.Bathrooms,
				SquareFootage: prop.SquareFootage,
			},
			cfg.RentcastBaseURL,
		))
	}

	// Create coordinator
	coord := coordinator.New(fetchers)

	// Add timeout to prevent hanging indefinitely
	fetchCtx, fetchCancel := context.WithTimeout(ctx, 30*time.Second)
	defer fetchCancel()

	// Run all fetchers concurrently
	fmt.Println("Fetching financial data from multiple sources...")
	fmt.Println("================================================")
	if err := coord.Run(fetchCtx); err != nil {
		log.Fatalf("Coordinator failed: %v", err)
	}

	fmt.Println("================================================")
	fmt.Println("All fetches completed!")
}
