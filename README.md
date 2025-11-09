# Finance Fetcher

A concurrent Go application that fetches financial data from multiple sources and outputs results with Redis-compatible keys.

## Architecture

The application follows a clean, interface-based architecture with concurrent workers:

```
┌─────────────────────────────────────────────────────────┐
│                      Main Program                        │
│  - Loads config                                          │
│  - Creates fetcher instances                             │
│  - Passes to Coordinator                                 │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                     Coordinator                          │
│  - Spawns goroutine per fetcher                         │
│  - Collects results via channel                          │
│  - Prints to stdout (ready for Redis integration)       │
└─────────────────────────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          ▼                 ▼                 ▼
    ┌──────────┐      ┌──────────┐      ┌──────────┐
    │ Fetcher  │      │ Fetcher  │      │ Fetcher  │
    │   #1     │      │   #2     │      │   #N     │
    └──────────┘      └──────────┘      └──────────┘
```

### Core Interface

```go
type Fetcher interface {
    Fetch(ctx context.Context) (float64, error)
    Key() string  // Redis-compatible key
}
```

### Supported Data Sources

1. **Etherscan** - Ethereum wallet balances in USD
   - Fetches ETH/USD price
   - Fetches wallet balance in wei
   - Calculates USD value
   - Key format: `fetcher:etherscan:{address}`

2. **AlphaVantage** - Stock prices
   - Real-time stock quotes
   - Key format: `fetcher:alphavantage:{ticker}`

3. **Rentcast** - Property valuations
   - Automated valuation models (AVM)
   - Includes price ranges and comparables
   - Key format: `fetcher:rentcast:{address_stub}`

4. **Guideline** - Retirement account balances (planned, not yet implemented)
   - Key format: `fetcher:guideline:{user_id_stub}`

## Configuration

Configuration is managed via Viper, supporting both environment variables and config files.

### Environment Variables

Create a `.env` file (see `.env.example`):

```bash
# Required
ETHERSCAN_API_KEY=your_key
ALPHAVANTAGE_API_KEY=your_key
RENTCAST_API_KEY=your_key
GUIDELINE_EMAIL=your_email
GUIDELINE_PASSWORD=your_password

# Optional (for testing/mocking)
ETHERSCAN_BASE_URL=https://api.etherscan.io/v2/api
ALPHAVANTAGE_BASE_URL=https://www.alphavantage.co/query
RENTCAST_BASE_URL=https://api.rentcast.io/v1
GUIDELINE_BASE_URL=https://my.guideline.com
```

## Usage

```bash
# Set up environment variables
cp .env.example .env
# Edit .env with your API keys

# Build
go build

# Run
./financefetcher
```

### Example Output

```
Fetching financial data from multiple sources...
================================================
fetcher:etherscan:0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb: $713842.91
fetcher:alphavantage:AAPL: $178.23
fetcher:alphavantage:GOOGL: $142.56
fetcher:alphavantage:MSFT: $378.91
fetcher:rentcast:5500_grand_lake_dr_san_antonio_tx_78244: $250000.00
================================================
All fetches completed!
```

## Project Structure

```
financefetcher/
├── main.go                           # Application entry point
├── go.mod                            # Go module definition
├── internal/
│   ├── config/
│   │   └── config.go                 # Viper-based configuration
│   ├── fetcher/
│   │   ├── fetcher.go                # Core Fetcher interface
│   │   └── result.go                 # Result type for channels
│   ├── coordinator/
│   │   └── coordinator.go            # Orchestrates concurrent fetchers
│   ├── etherscan/
│   │   └── wallet.go                 # Ethereum wallet balance fetcher
│   ├── alphavantage/
│   │   └── stock.go                  # Stock price fetcher
│   └── rentcast/
│       └── property.go               # Property valuation fetcher
```

## Design Decisions

### Concurrency Pattern

- Each fetcher runs in its own goroutine
- Results are sent to a shared channel
- Coordinator collects and processes results as they arrive
- Context-based cancellation for graceful shutdown

### Redis Key Format

Hierarchical keys using `:` separator for easy namespacing:
- `fetcher:{source}:{identifier}`
- Examples: `fetcher:etherscan:0x123...`, `fetcher:alphavantage:AAPL`

### Configuration

- Viper for flexible config management
- Environment variables with sensible defaults
- Base URLs configurable for testing/mocking

### HTTP Client

- Uses `resty.dev/v3` for all HTTP requests
- Clean API, built-in retry support
- Automatic JSON marshaling/unmarshaling

## Future Enhancements

- [ ] Redis integration (replace stdout with Redis SET commands)
- [ ] Guideline fetcher implementation (requires browser automation)
- [ ] Configurable TTL for cache entries
- [ ] Retry logic with exponential backoff
- [ ] Metrics and observability
- [ ] Dynamic fetcher configuration from file/API
- [ ] Rate limiting per API source