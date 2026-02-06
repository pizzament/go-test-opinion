package main

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"os"
	"sync"
	"time"
)

type Market struct {
	MarketID    int    `json:"marketId"`
	MarketTitle string `json:"marketTitle"`
	Status      int    `json:"status"`
	StatusEnum  string `json:"statusEnum"`
	YesTokenID  string `json:"yesTokenId"`
	NoTokenID   string `json:"noTokenId"`
	Volume      string `json:"volume"`
	Volume24h   string `json:"volume24h"`
}

type MarketWithPrices struct {
	Market
	YesBid string `json:"yesBid"`
	YesAsk string `json:"yesAsk"`
	NoBid  string `json:"noBid"`
	NoAsk  string `json:"noAsk"`
}

type Order struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

type OrderbookResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Result struct {
		Market    string  `json:"market"`
		TokenID   string  `json:"tokenId"`
		Timestamp int64   `json:"timestamp"`
		Bids      []Order `json:"bids"`
		Asks      []Order `json:"asks"`
	} `json:"result"`
}

type APIResponse struct {
	Code   int    `json:"code"`
	Msg    string `json:"msg"`
	Result struct {
		Total int      `json:"total"`
		List  []Market `json:"list"`
	} `json:"result"`
}

type MarketDetail struct {
	MarketID         int    `json:"marketId"`
	MarketTitle      string `json:"marketTitle"`
	MarketRules      string `json:"marketRules"`
	Status           int    `json:"status"`
	StatusEnum       string `json:"statusEnum"`
	YesTokenID       string `json:"yesTokenId"`
	NoTokenID        string `json:"noTokenId"`
	YesTokenLabel    string `json:"yesTokenLabel"`
	NoTokenLabel     string `json:"noTokenLabel"`
	Volume           string `json:"volume"`
	Volume24h        string `json:"volume24h"`
	Volume7d         string `json:"volume7d"`
	QuoteToken       string `json:"quoteToken"`
	ChainID          int    `json:"chainId"`
	CreatedTimestamp int64  `json:"createdTimestamp"`
	CutoffTimestamp  int64  `json:"cutoffTimestamp"`
}

type MarketDetailResponse struct {
	Code   int          `json:"code"`
	Msg    string       `json:"msg"`
	Result MarketDetail `json:"result"`
}

const opinionTradeAPI = "https://openapi.opinion.trade/openapi/market"
const orderbookAPI = "https://openapi.opinion.trade/openapi/token/orderbook"

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/api/random-markets", getRandomMarkets)
	http.HandleFunc("/market/", serveMarketDetail)

	port := "8080"
	fmt.Printf("Server starting on http://localhost:%s\n", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func serveHome(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, getHTML())
}

func getRandomMarkets(w http.ResponseWriter, r *http.Request) {
	apiKey := os.Getenv("OPINION_API_KEY")
	if apiKey == "" {
		http.Error(w, "OPINION_API_KEY environment variable not set", http.StatusInternalServerError)
		return
	}

	// Create request with query parameters
	url := fmt.Sprintf("%s?status=activated&limit=20", opinionTradeAPI)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	// Add API key header
	req.Header.Set("apikey", apiKey)

	// Fetch markets from Opinion.trade API
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch markets: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("API returned status %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusInternalServerError)
		return
	}

	var apiResp APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse markets: %v", err), http.StatusInternalServerError)
		return
	}

	if apiResp.Code != 0 {
		http.Error(w, fmt.Sprintf("API error: %s", apiResp.Msg), http.StatusInternalServerError)
		return
	}

	// Select 5 random markets
	randomMarkets := selectRandomMarkets(apiResp.Result.List, 5)

	// Enrich markets with bid/ask prices
	marketsWithPrices := enrichMarketsWithPrices(randomMarkets, apiKey, client)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(marketsWithPrices)
}

func selectRandomMarkets(markets []Market, count int) []Market {
	if len(markets) == 0 {
		return []Market{}
	}

	if len(markets) <= count {
		return markets
	}

	// Fisher-Yates shuffle
	shuffled := make([]Market, len(markets))
	copy(shuffled, markets)

	for i := len(shuffled) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}

	return shuffled[:count]
}

func enrichMarketsWithPrices(markets []Market, apiKey string, client *http.Client) []MarketWithPrices {
	result := make([]MarketWithPrices, len(markets))
	var wg sync.WaitGroup

	for i, market := range markets {
		result[i].Market = market
		wg.Add(2)

		// Fetch Yes token orderbook
		go func(idx int, tokenID string) {
			defer wg.Done()
			bid, ask := fetchBestPrices(tokenID, apiKey, client)
			result[idx].YesBid = bid
			result[idx].YesAsk = ask
		}(i, market.YesTokenID)

		// Fetch No token orderbook
		go func(idx int, tokenID string) {
			defer wg.Done()
			bid, ask := fetchBestPrices(tokenID, apiKey, client)
			result[idx].NoBid = bid
			result[idx].NoAsk = ask
		}(i, market.NoTokenID)
	}

	wg.Wait()
	return result
}

func fetchBestPrices(tokenID, apiKey string, client *http.Client) (bestBid, bestAsk string) {
	if tokenID == "" {
		return "N/A", "N/A"
	}

	url := fmt.Sprintf("%s?token_id=%s", orderbookAPI, tokenID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		log.Printf("Failed to create orderbook request for %s: %v", tokenID, err)
		return "N/A", "N/A"
	}

	req.Header.Set("apikey", apiKey)

	resp, err := client.Do(req)
	if err != nil {
		log.Printf("Failed to fetch orderbook for %s: %v", tokenID, err)
		return "N/A", "N/A"
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("Orderbook API returned status %d for %s", resp.StatusCode, tokenID)
		return "N/A", "N/A"
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("Failed to read orderbook response for %s: %v", tokenID, err)
		return "N/A", "N/A"
	}

	var obResp OrderbookResponse
	if err := json.Unmarshal(body, &obResp); err != nil {
		log.Printf("Failed to parse orderbook for %s: %v", tokenID, err)
		return "N/A", "N/A"
	}

	if obResp.Code != 0 {
		log.Printf("Orderbook API error for %s: %s", tokenID, obResp.Msg)
		return "N/A", "N/A"
	}

	// Get best bid (highest bid price - first in bids array)
	if len(obResp.Result.Bids) > 0 {
		bestBid = obResp.Result.Bids[0].Price
	} else {
		bestBid = "N/A"
	}

	// Get best ask (lowest ask price - first in asks array)
	if len(obResp.Result.Asks) > 0 {
		bestAsk = obResp.Result.Asks[0].Price
	} else {
		bestAsk = "N/A"
	}

	return bestBid, bestAsk
}

func serveMarketDetail(w http.ResponseWriter, r *http.Request) {
	// Extract market ID from URL path
	marketID := r.URL.Path[len("/market/"):]
	if marketID == "" {
		http.Error(w, "Market ID required", http.StatusBadRequest)
		return
	}

	apiKey := os.Getenv("OPINION_API_KEY")
	if apiKey == "" {
		http.Error(w, "OPINION_API_KEY environment variable not set", http.StatusInternalServerError)
		return
	}

	// Fetch market details
	url := fmt.Sprintf("%s/%s", opinionTradeAPI, marketID)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to create request: %v", err), http.StatusInternalServerError)
		return
	}

	req.Header.Set("apikey", apiKey)

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to fetch market details: %v", err), http.StatusInternalServerError)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		http.Error(w, fmt.Sprintf("API returned status %d", resp.StatusCode), http.StatusInternalServerError)
		return
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to read response: %v", err), http.StatusInternalServerError)
		return
	}

	var marketResp MarketDetailResponse
	if err := json.Unmarshal(body, &marketResp); err != nil {
		http.Error(w, fmt.Sprintf("Failed to parse market details: %v", err), http.StatusInternalServerError)
		return
	}

	if marketResp.Code != 0 {
		http.Error(w, fmt.Sprintf("API error: %s", marketResp.Msg), http.StatusInternalServerError)
		return
	}

	// Fetch orderbook prices for Yes and No tokens concurrently
	var yesBid, yesAsk, noBid, noAsk string
	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		yesBid, yesAsk = fetchBestPrices(marketResp.Result.YesTokenID, apiKey, client)
	}()
	go func() {
		defer wg.Done()
		noBid, noAsk = fetchBestPrices(marketResp.Result.NoTokenID, apiKey, client)
	}()
	wg.Wait()

	// Serve the market detail page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprint(w, getMarketDetailHTML(marketResp.Result, yesBid, yesAsk, noBid, noAsk))
}

func getHTML() string {
	return `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>Random Markets - Opinion.trade</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        h1 {
            color: white;
            text-align: center;
            margin-bottom: 30px;
            font-size: 2.5rem;
            text-shadow: 2px 2px 4px rgba(0,0,0,0.2);
        }

        .controls {
            text-align: center;
            margin-bottom: 30px;
        }

        button {
            background: white;
            color: #667eea;
            border: none;
            padding: 15px 30px;
            font-size: 1.1rem;
            border-radius: 25px;
            cursor: pointer;
            box-shadow: 0 4px 6px rgba(0,0,0,0.1);
            transition: transform 0.2s, box-shadow 0.2s;
            font-weight: 600;
        }

        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 6px 12px rgba(0,0,0,0.15);
        }

        button:active {
            transform: translateY(0);
        }

        button:disabled {
            opacity: 0.6;
            cursor: not-allowed;
        }

        .markets-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 20px;
        }

        .market-link {
            text-decoration: none;
            color: inherit;
            display: block;
        }

        .market-card {
            background: white;
            border-radius: 15px;
            padding: 25px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            transition: transform 0.3s, box-shadow 0.3s;
            animation: fadeIn 0.5s ease-in;
            cursor: pointer;
        }

        @keyframes fadeIn {
            from {
                opacity: 0;
                transform: translateY(20px);
            }
            to {
                opacity: 1;
                transform: translateY(0);
            }
        }

        .market-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 15px 40px rgba(0,0,0,0.3);
        }

        .market-question {
            font-size: 1.2rem;
            font-weight: 700;
            color: #2d3748;
            margin-bottom: 10px;
            line-height: 1.4;
        }

        .market-description {
            color: #718096;
            margin-bottom: 15px;
            line-height: 1.6;
            font-size: 0.95rem;
        }

        .market-stats {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 10px;
            margin-top: 15px;
            padding-top: 15px;
            border-top: 2px solid #e2e8f0;
        }

        .stat {
            text-align: center;
        }

        .stat-label {
            font-size: 0.75rem;
            color: #a0aec0;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 5px;
        }

        .stat-value {
            font-size: 1.3rem;
            font-weight: 700;
        }

        .stat-value.yes {
            color: #48bb78;
        }

        .stat-value.no {
            color: #f56565;
        }

        .market-meta {
            margin-top: 15px;
            font-size: 0.85rem;
            color: #a0aec0;
        }

        .loading {
            text-align: center;
            color: white;
            font-size: 1.2rem;
            padding: 40px;
        }

        .error {
            background: #fed7d7;
            color: #c53030;
            padding: 20px;
            border-radius: 10px;
            text-align: center;
            margin: 20px 0;
        }

        .spinner {
            border: 3px solid rgba(255,255,255,0.3);
            border-top: 3px solid white;
            border-radius: 50%;
            width: 40px;
            height: 40px;
            animation: spin 1s linear infinite;
            margin: 0 auto;
        }

        @keyframes spin {
            0% { transform: rotate(0deg); }
            100% { transform: rotate(360deg); }
        }
    </style>
</head>
<body>
    <div class="container">
        <h1>🎲 Random Markets from Opinion.trade</h1>

        <div class="controls">
            <button id="refreshBtn" onclick="loadMarkets()">🔄 Load Random Markets</button>
        </div>

        <div id="marketsContainer"></div>
    </div>

    <script>
        async function loadMarkets() {
            const container = document.getElementById('marketsContainer');
            const btn = document.getElementById('refreshBtn');

            btn.disabled = true;
            container.innerHTML = '<div class="loading"><div class="spinner"></div><p style="margin-top: 20px;">Loading markets...</p></div>';

            try {
                const response = await fetch('/api/random-markets');
                if (!response.ok) {
                    throw new Error('Failed to fetch markets');
                }

                const markets = await response.json();
                displayMarkets(markets);
            } catch (error) {
                container.innerHTML = '<div class="error">❌ Error loading markets: ' + error.message + '</div>';
            } finally {
                btn.disabled = false;
            }
        }

        function displayMarkets(markets) {
            const container = document.getElementById('marketsContainer');

            if (!markets || markets.length === 0) {
                container.innerHTML = '<div class="error">No markets found</div>';
                return;
            }

            const html = '<div class="markets-grid">' +
                markets.map(market =>
                    '<a href="/market/' + market.marketId + '" class="market-link">' +
                        '<div class="market-card">' +
                            '<div class="market-question">' + escapeHtml(market.marketTitle) + '</div>' +
                            '<div class="market-description">Market ID: ' + market.marketId + ' • Status: ' + market.statusEnum + '</div>' +
                            '<div class="market-stats">' +
                                '<div class="stat">' +
                                    '<div class="stat-label">Yes Bid</div>' +
                                    '<div class="stat-value yes">' + formatPrice(market.yesBid) + '</div>' +
                                '</div>' +
                                '<div class="stat">' +
                                    '<div class="stat-label">Yes Ask</div>' +
                                    '<div class="stat-value yes">' + formatPrice(market.yesAsk) + '</div>' +
                                '</div>' +
                            '</div>' +
                            '<div class="market-stats" style="margin-top: 10px;">' +
                                '<div class="stat">' +
                                    '<div class="stat-label">No Bid</div>' +
                                    '<div class="stat-value no">' + formatPrice(market.noBid) + '</div>' +
                                '</div>' +
                                '<div class="stat">' +
                                    '<div class="stat-label">No Ask</div>' +
                                    '<div class="stat-value no">' + formatPrice(market.noAsk) + '</div>' +
                                '</div>' +
                            '</div>' +
                            '<div class="market-meta">💰 Volume: $' + formatVolume(market.volume) + ' (24h: $' + formatVolume(market.volume24h) + ')</div>' +
                        '</div>' +
                    '</a>'
                ).join('') +
                '</div>';

            container.innerHTML = html;
        }

        function formatPrice(price) {
            if (!price || price === 'N/A') return 'N/A';
            const num = parseFloat(price);
            if (isNaN(num)) return price;
            return (num * 100).toFixed(1) + '%';
        }

        function formatVolume(volume) {
            if (!volume) return '0';
            const num = parseFloat(volume);
            if (num >= 1000000) return (num / 1000000).toFixed(2) + 'M';
            if (num >= 1000) return (num / 1000).toFixed(2) + 'K';
            return num.toFixed(2);
        }

        function shortenAddress(addr) {
            if (!addr || addr.length < 12) return addr;
            return addr.substring(0, 6) + '...' + addr.substring(addr.length - 4);
        }

        function escapeHtml(text) {
            const div = document.createElement('div');
            div.textContent = text;
            return div.innerHTML;
        }

        // Load markets on page load
        loadMarkets();
    </script>
</body>
</html>`
}

func getMarketDetailHTML(market MarketDetail, yesBid, yesAsk, noBid, noAsk string) string {
	createdTime := time.Unix(market.CreatedTimestamp/1000, 0).Format("Jan 2, 2006 3:04 PM")
	cutoffTime := ""
	if market.CutoffTimestamp > 0 {
		cutoffTime = time.Unix(market.CutoffTimestamp/1000, 0).Format("Jan 2, 2006 3:04 PM")
	}

	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s - Opinion.trade</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, Oxygen, Ubuntu, Cantarell, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            padding: 20px;
        }

        .container {
            max-width: 900px;
            margin: 0 auto;
        }

        .back-button {
            display: inline-block;
            background: white;
            color: #667eea;
            padding: 10px 20px;
            border-radius: 20px;
            text-decoration: none;
            margin-bottom: 20px;
            font-weight: 600;
            transition: transform 0.2s;
        }

        .back-button:hover {
            transform: translateY(-2px);
        }

        .detail-card {
            background: white;
            border-radius: 20px;
            padding: 40px;
            box-shadow: 0 10px 40px rgba(0,0,0,0.3);
            margin-bottom: 20px;
        }

        h1 {
            color: #2d3748;
            font-size: 2rem;
            margin-bottom: 20px;
            line-height: 1.3;
        }

        .status-badge {
            display: inline-block;
            background: #48bb78;
            color: white;
            padding: 6px 16px;
            border-radius: 20px;
            font-size: 0.85rem;
            font-weight: 600;
            margin-bottom: 20px;
        }

        .market-rules {
            background: #f7fafc;
            padding: 20px;
            border-radius: 10px;
            margin: 20px 0;
            border-left: 4px solid #667eea;
        }

        .market-rules h3 {
            color: #2d3748;
            margin-bottom: 10px;
            font-size: 1.1rem;
        }

        .market-rules p {
            color: #4a5568;
            line-height: 1.6;
            white-space: pre-wrap;
        }

        .stats-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 20px;
            margin: 30px 0;
        }

        .stat-box {
            background: #f7fafc;
            padding: 20px;
            border-radius: 10px;
            text-align: center;
        }

        .stat-label {
            font-size: 0.85rem;
            color: #718096;
            text-transform: uppercase;
            letter-spacing: 0.5px;
            margin-bottom: 8px;
        }

        .stat-value {
            font-size: 1.5rem;
            font-weight: 700;
            color: #2d3748;
        }

        .prices-section {
            margin: 30px 0;
        }

        .prices-section h2 {
            color: #2d3748;
            margin-bottom: 20px;
            font-size: 1.5rem;
        }

        .token-prices {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 20px;
            margin-bottom: 20px;
        }

        .token-card {
            background: #f7fafc;
            padding: 25px;
            border-radius: 15px;
            border: 2px solid #e2e8f0;
        }

        .token-card.yes {
            border-color: #48bb78;
        }

        .token-card.no {
            border-color: #f56565;
        }

        .token-label {
            font-size: 1.2rem;
            font-weight: 700;
            margin-bottom: 15px;
            display: flex;
            align-items: center;
        }

        .token-label.yes {
            color: #48bb78;
        }

        .token-label.no {
            color: #f56565;
        }

        .price-row {
            display: flex;
            justify-content: space-between;
            margin: 10px 0;
            padding: 10px;
            background: white;
            border-radius: 8px;
        }

        .price-label {
            font-weight: 600;
            color: #4a5568;
        }

        .price-value {
            font-weight: 700;
            font-size: 1.1rem;
        }

        .price-value.yes {
            color: #48bb78;
        }

        .price-value.no {
            color: #f56565;
        }

        .token-id {
            font-size: 0.75rem;
            color: #a0aec0;
            word-break: break-all;
            margin-top: 10px;
            font-family: monospace;
        }

        .metadata {
            display: grid;
            grid-template-columns: 1fr 1fr;
            gap: 15px;
            margin: 20px 0;
        }

        .meta-item {
            padding: 15px;
            background: #f7fafc;
            border-radius: 10px;
        }

        .meta-label {
            font-size: 0.85rem;
            color: #718096;
            margin-bottom: 5px;
        }

        .meta-value {
            color: #2d3748;
            font-weight: 600;
        }

        @media (max-width: 768px) {
            .token-prices {
                grid-template-columns: 1fr;
            }

            .stats-grid {
                grid-template-columns: 1fr;
            }

            .metadata {
                grid-template-columns: 1fr;
            }

            .detail-card {
                padding: 25px;
            }

            h1 {
                font-size: 1.5rem;
            }
        }
    </style>
</head>
<body>
    <div class="container">
        <a href="/" class="back-button">← Back to Markets</a>

        <div class="detail-card">
            <div class="status-badge">%s</div>
            <h1>%s</h1>

            <div class="stats-grid">
                <div class="stat-box">
                    <div class="stat-label">Total Volume</div>
                    <div class="stat-value">$%s</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">24h Volume</div>
                    <div class="stat-value">$%s</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">7d Volume</div>
                    <div class="stat-value">$%s</div>
                </div>
                <div class="stat-box">
                    <div class="stat-label">Market ID</div>
                    <div class="stat-value">#%d</div>
                </div>
            </div>

            %s

            <div class="prices-section">
                <h2>Current Prices</h2>
                <div class="token-prices">
                    <div class="token-card yes">
                        <div class="token-label yes">✓ %s</div>
                        <div class="price-row">
                            <span class="price-label">Bid Price</span>
                            <span class="price-value yes">%s</span>
                        </div>
                        <div class="price-row">
                            <span class="price-label">Ask Price</span>
                            <span class="price-value yes">%s</span>
                        </div>
                        <div class="token-id">Token: %s</div>
                    </div>

                    <div class="token-card no">
                        <div class="token-label no">✗ %s</div>
                        <div class="price-row">
                            <span class="price-label">Bid Price</span>
                            <span class="price-value no">%s</span>
                        </div>
                        <div class="price-row">
                            <span class="price-label">Ask Price</span>
                            <span class="price-value no">%s</span>
                        </div>
                        <div class="token-id">Token: %s</div>
                    </div>
                </div>
            </div>

            <div class="metadata">
                <div class="meta-item">
                    <div class="meta-label">Created</div>
                    <div class="meta-value">%s</div>
                </div>
                %s
                <div class="meta-item">
                    <div class="meta-label">Chain ID</div>
                    <div class="meta-value">%d</div>
                </div>
                <div class="meta-item">
                    <div class="meta-label">Quote Token</div>
                    <div class="meta-value">%s</div>
                </div>
            </div>
        </div>
    </div>
</body>
</html>`,
		market.MarketTitle,
		market.StatusEnum,
		market.MarketTitle,
		formatVolumeGo(market.Volume),
		formatVolumeGo(market.Volume24h),
		formatVolumeGo(market.Volume7d),
		market.MarketID,
		getMarketRulesHTML(market.MarketRules),
		market.YesTokenLabel,
		formatPriceGo(yesBid),
		formatPriceGo(yesAsk),
		market.YesTokenID,
		market.NoTokenLabel,
		formatPriceGo(noBid),
		formatPriceGo(noAsk),
		market.NoTokenID,
		createdTime,
		getCutoffHTML(cutoffTime),
		market.ChainID,
		market.QuoteToken,
	)
}

func getMarketRulesHTML(rules string) string {
	if rules == "" {
		return ""
	}
	return fmt.Sprintf(`
            <div class="market-rules">
                <h3>Market Rules</h3>
                <p>%s</p>
            </div>`, rules)
}

func getCutoffHTML(cutoffTime string) string {
	if cutoffTime == "" {
		return ""
	}
	return fmt.Sprintf(`
                <div class="meta-item">
                    <div class="meta-label">Cutoff Time</div>
                    <div class="meta-value">%s</div>
                </div>`, cutoffTime)
}

func formatPriceGo(price string) string {
	if price == "" || price == "N/A" {
		return "N/A"
	}
	// Parse the price and convert to percentage
	var p float64
	fmt.Sscanf(price, "%f", &p)
	return fmt.Sprintf("%.1f%%", p*100)
}

func formatVolumeGo(volume string) string {
	if volume == "" {
		return "0"
	}
	var v float64
	fmt.Sscanf(volume, "%f", &v)
	if v >= 1000000 {
		return fmt.Sprintf("%.2fM", v/1000000)
	}
	if v >= 1000 {
		return fmt.Sprintf("%.2fK", v/1000)
	}
	return fmt.Sprintf("%.2f", v)
}
