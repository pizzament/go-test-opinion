# Random Markets UI - Opinion.trade

A simple Go web application that displays 5 random prediction markets from Opinion.trade with a beautiful, modern UI.

## Features

- Fetches markets from the official Opinion.trade Open API
- Displays 5 randomly selected markets
- Modern, responsive UI with smooth animations
- Real-time market data including:
  - Market title and status
  - Total volume and 24h volume
  - Yes/No token addresses
  - Market ID
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

## How It Works

- The Go server runs on port 8080
- `/` - Serves the HTML UI
- `/api/random-markets` - API endpoint that fetches markets from Opinion.trade and returns 5 random ones

The application uses the Fisher-Yates shuffle algorithm to randomly select 5 markets from all available markets.

## API Integration

The app uses the official Opinion.trade Open API:

**Endpoint:** `GET https://openapi.opinion.trade/openapi/market`

**Query Parameters:**
- `status=activated` - Only fetch active markets
- `limit=20` - Fetch up to 20 markets per request

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
- Color-coded volume statistics (green for total, red for 24h)
- Hover effects and smooth animations
- Responsive design that works on mobile and desktop
- Loading spinner and error handling

## License

MIT
