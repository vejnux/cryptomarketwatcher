package main

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"time"
)

// CoinGecko API base URL
const baseURL = "https://api.coingecko.com/api/v3"

// Struct for storing cryptocurrency market data
type CryptoData struct {
	ID                string      `json:"id"`
	Symbol            string      `json:"symbol"`
	Name              string      `json:"name"`
	CurrentPrice      interface{} `json:"current_price"`
	MarketCap         interface{} `json:"market_cap"`
	MarketCapRank     interface{} `json:"market_cap_rank"`
	TotalVolume       interface{} `json:"total_volume"`
	High24h           interface{} `json:"high_24h"`
	Low24h            interface{} `json:"low_24h"`
	PriceChange24h    interface{} `json:"price_change_24h"`
	PriceChangePct24h interface{} `json:"price_change_percentage_24h"`
	LastUpdated       string      `json:"last_updated"`
}

// Struct for storing all available coins (to get all symbols)
type CoinList struct {
	ID     string `json:"id"`
	Symbol string `json:"symbol"`
	Name   string `json:"name"`
}

// Fetch all available cryptocurrency symbols from CoinGecko
func fetchAllSymbols() ([]string, error) {
	url := baseURL + "/coins/list"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching symbols: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response body: %v", err)
	}

	// Debugging: Print raw API response (only first 500 characters to avoid spam)
	fmt.Println("Raw API Response:", string(body)[:min(500, len(body))])

	var coinList []CoinList
	err = json.Unmarshal(body, &coinList)
	if err != nil {
		return nil, fmt.Errorf("error unmarshaling JSON: %v", err)
	}

	// Extract IDs of all available coins
	var coinIDs []string
	for _, coin := range coinList {
		coinIDs = append(coinIDs, coin.ID)
	}

	fmt.Println("Total symbols fetched:", len(coinIDs))
	return coinIDs, nil
}

func fetchMarketData(page int) ([]CryptoData, error) {
	url := fmt.Sprintf("https://api.coingecko.com/api/v3/coins/markets?vs_currency=usd&order=market_cap_desc&per_page=250&page=%d", page)

	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 429 {
		log.Println("⚠️ API Rate Limit Hit! Waiting before retrying...")
		time.Sleep(15 * time.Second) // Longer delay
		return fetchMarketData(page) // Retry the same page
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var data []CryptoData
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}

	return data, nil
}

func saveToCSV(data []CryptoData) error {
	// Generate CSV filename
	filename := fmt.Sprintf("market_data_%s.csv", time.Now().Format("20060102_150405"))

	// Open file
	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("error creating CSV file: %v", err)
	}
	defer file.Close()

	writer := csv.NewWriter(file)
	defer writer.Flush()

	// Write headers
	headers := []string{"Timestamp", "ID", "Symbol", "Name", "Current Price", "Market Cap", "Market Cap Rank",
		"Total Volume", "High 24h", "Low 24h", "Price Change 24h", "Price Change % 24h", "Last Updated"}
	writer.Write(headers)

	// Write data
	for _, d := range data {
		writer.Write([]string{
			time.Now().Format("2006-01-02 15:04:05"), // Timestamp
			d.ID, d.Symbol, d.Name,
			fmt.Sprintf("%.2f", d.CurrentPrice),
			fmt.Sprintf("%.0f", d.MarketCap),
			fmt.Sprintf("%d", int(math.Round(d.MarketCapRank.(float64)))),
			fmt.Sprintf("%.0f", d.TotalVolume),
			fmt.Sprintf("%.2f", d.High24h),
			fmt.Sprintf("%.2f", d.Low24h),
			fmt.Sprintf("%.2f", d.PriceChange24h),
			fmt.Sprintf("%.2f", d.PriceChangePct24h),
			d.LastUpdated,
		})
	}

	fmt.Println("✅ Data saved to", filename)
	return nil
}


func main() {
	var allData []CryptoData

	// Fetch first 500 coins (2 pages)
	for page := 1; page <= 2; page++ {
		data, err := fetchMarketData(page)
		if err != nil {
			log.Fatalf("Error fetching market data: %v", err)
		}
		allData = append(allData, data...)
		time.Sleep(10 * time.Second) // Delay between API calls
	}

	if err := saveToCSV(allData); err != nil {
		log.Fatalf("Error saving CSV: %v", err)
	}
}