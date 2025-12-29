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

const opinionTradeAPI = "https://openapi.opinion.trade/openapi/market"
const orderbookAPI = "https://openapi.opinion.trade/openapi/token/orderbook"

func main() {
	rand.Seed(time.Now().UnixNano())

	http.HandleFunc("/", serveHome)
	http.HandleFunc("/api/random-markets", getRandomMarkets)

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

        .market-card {
            background: white;
            border-radius: 15px;
            padding: 25px;
            box-shadow: 0 10px 30px rgba(0,0,0,0.2);
            transition: transform 0.3s, box-shadow 0.3s;
            animation: fadeIn 0.5s ease-in;
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
                    '</div>'
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
