# Random Markets UI - Opinion.trade

A simple Go web application that displays 5 random prediction markets from Opinion.trade with a beautiful, modern UI.

## Features

- Fetches markets from the Opinion.trade API
- Displays 5 randomly selected markets
- Modern, responsive UI with smooth animations
- Real-time market data including:
  - Market question and description
  - Yes/No prices (displayed as percentages)
  - End date
  - Trading volume
- Refresh button to load new random markets

## Prerequisites

- Go 1.16 or higher

## Installation

1. Clone the repository:
```bash
git clone https://github.com/pizzament/go-test-opinion.git
cd go-test-opinion
```

2. No additional dependencies needed (uses only Go standard library)

## Usage

1. Run the application:
```bash
go run main.go
```

2. Open your browser and navigate to:
```
http://localhost:8080
```

3. Click the "Load Random Markets" button to fetch and display 5 random markets

## How It Works

- The Go server runs on port 8080
- `/` - Serves the HTML UI
- `/api/random-markets` - API endpoint that fetches markets from Opinion.trade and returns 5 random ones

The application uses the Fisher-Yates shuffle algorithm to randomly select 5 markets from all available markets.

## API Integration

The app fetches data from:
```
https://api.opinion.trade/v1/markets
```

**Note:** The actual API endpoint may differ. If you encounter issues, please check the Opinion.trade API documentation at https://docs.opinion.trade/ for the correct endpoint and data structure.

## Building

To build the application:
```bash
go build -o random-markets
./random-markets
```

## Screenshots

The UI features:
- Gradient purple background
- Card-based layout for each market
- Color-coded Yes (green) and No (red) prices
- Hover effects and smooth animations
- Responsive design that works on mobile and desktop

## License

MIT
