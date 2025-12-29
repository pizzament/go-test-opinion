# Random Markets UI - Opinion.trade

A simple Go web application that displays 5 random prediction markets from Opinion.trade with a beautiful, modern UI.

## Features

- Fetches markets from the official Opinion.trade Open API
- Displays 5 randomly selected markets
- Clickable market cards that link to detailed market pages
- Modern, responsive UI with smooth animations
- Real-time market data including:
  - Market title and status
  - Bid/Ask prices for Yes and No tokens (displayed as percentages)
  - Total volume and 24h volume
  - Market ID
- Detailed market view showing:
  - Complete market rules and description
  - Volume statistics (total, 24h, 7d)
  - Full orderbook prices for both Yes and No tokens
  - Token IDs and labels
  - Market metadata (created time, cutoff time, chain ID, quote token)
- Concurrent orderbook fetching for optimal performance
- Refresh button to load new random markets

## Prerequisites

- Go 1.16 or higher
- Opinion.trade API key (apply at https://docs.opinion.trade/)

## Installation

1. Clone the repository:
```bash
git clone https://github.com/pizzament/go-test-opinion.git
cd go-test-opinion
```

2. No additional dependencies needed (uses only Go standard library)

## Setup

1. Get your API key from Opinion.trade by filling out the application form at https://docs.opinion.trade/developer-guide/opinion-open-api/authentication

2. Set the API key as an environment variable:
```bash
export OPINION_API_KEY=your_api_key_here
```

## Usage

1. Run the application:
```bash
OPINION_API_KEY=your_api_key_here go run main.go
```

Or if you've already exported the environment variable:
```bash
go run main.go
```

2. Open your browser and navigate to:
```
http://localhost:8080
```

3. Click the "Load Random Markets" button to fetch and display 5 random activated markets
4. Click on any market card to view detailed information about that market

## How It Works

- The Go server runs on port 8080
- `/` - Serves the main HTML UI with random markets grid
- `/api/random-markets` - API endpoint that fetches markets from Opinion.trade and returns 5 random ones
- `/market/{id}` - Market detail page showing comprehensive information about a specific market

The application uses the Fisher-Yates shuffle algorithm to randomly select 5 markets from all available markets.

## API Integration

The app uses the official Opinion.trade Open API:

**Markets Endpoint:** `GET https://openapi.opinion.trade/openapi/market`

**Query Parameters:**
- `status=activated` - Only fetch active markets
- `limit=20` - Fetch up to 20 markets per request

**Market Detail Endpoint:** `GET https://openapi.opinion.trade/openapi/market/{marketId}`

Returns comprehensive information about a specific market including:
- Market rules and description
- Volume statistics (total, 24h, 7d)
- Token IDs and labels
- Timestamps (created, cutoff)
- Chain and quote token information

**Orderbook Endpoint:** `GET https://openapi.opinion.trade/openapi/token/orderbook?token_id={tokenId}`

Returns bid/ask prices for each token (Yes and No outcomes)

**Authentication:** Requires `apikey` header with your Opinion.trade API key

For more information, see the [Opinion.trade API documentation](https://docs.opinion.trade/developer-guide/opinion-open-api/overview).

## Building

To build the application:
```bash
go build -o random-markets
OPINION_API_KEY=your_api_key_here ./random-markets
```

## UI Features

The UI features:
- Gradient purple background
- Card-based layout for each market
- Color-coded bid/ask prices:
  - Green for Yes token prices (bid/ask)
  - Red for No token prices (bid/ask)
  - Prices displayed as percentages
- Volume statistics for total and 24h trading activity
- Hover effects and smooth animations
- Responsive design that works on mobile and desktop
- Loading spinner and error handling

## License

MIT
