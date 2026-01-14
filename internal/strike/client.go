package strike

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

const (
	// DefaultBaseURL is the Strike API base URL
	DefaultBaseURL = "https://api.strike.me/v1"
)

// Client is a Strike API client
type Client struct {
	apiKey  string
	baseURL string
	client  *http.Client
}

// BalanceResponse represents a single currency balance from Strike API
type BalanceResponse struct {
	Currency  string `json:"currency"`
	Available string `json:"available"`
	Total     string `json:"total"`
	Pending   string `json:"pending"`
	Reserved  string `json:"reserved"`
	Outgoing  string `json:"outgoing"`
	Current   string `json:"current"`
}

// BalanceDetail represents detailed balance information with parsed amounts
type BalanceDetail struct {
	Currency  string
	Available int64 // In smallest unit (sats for BTC, cents for fiat)
	Total     int64
	Pending   int64
	Reserved  int64
	Timestamp time.Time
}

// NewClient creates a new Strike API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: DefaultBaseURL,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// GetAccountBalance fetches current account balance from Strike API
func (c *Client) GetAccountBalance() ([]BalanceDetail, error) {
	url := fmt.Sprintf("%s/balances", c.baseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", c.apiKey))
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("strike API returned status %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var balances []BalanceResponse
	if err := json.Unmarshal(body, &balances); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Convert to BalanceDetail with proper unit conversion
	details := make([]BalanceDetail, 0, len(balances))
	timestamp := time.Now()

	for _, bal := range balances {
		detail := BalanceDetail{
			Currency:  bal.Currency,
			Timestamp: timestamp,
		}

		// Convert string amounts to integers in smallest units
		// For BTC: convert to satoshis
		// For fiat: convert to cents
		var err error
		detail.Available, err = parseAmountToSmallestUnit(bal.Available, bal.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to parse available for %s: %w", bal.Currency, err)
		}

		detail.Total, err = parseAmountToSmallestUnit(bal.Total, bal.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to parse total for %s: %w", bal.Currency, err)
		}

		detail.Pending, err = parseAmountToSmallestUnit(bal.Pending, bal.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pending for %s: %w", bal.Currency, err)
		}

		detail.Reserved, err = parseAmountToSmallestUnit(bal.Reserved, bal.Currency)
		if err != nil {
			return nil, fmt.Errorf("failed to parse reserved for %s: %w", bal.Currency, err)
		}

		details = append(details, detail)
	}

	return details, nil
}

// parseAmountToSmallestUnit converts Strike API string amounts to smallest units
// For BTC: converts to satoshis (multiply by 100,000,000)
// For fiat: converts to cents (multiply by 100)
func parseAmountToSmallestUnit(amountStr, currency string) (int64, error) {
	if amountStr == "" || amountStr == "0" {
		return 0, nil
	}

	// Parse as float first
	amount, err := strconv.ParseFloat(amountStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid amount format: %w", err)
	}

	// Convert to smallest unit based on currency
	var multiplier float64
	if currency == "BTC" {
		multiplier = 100_000_000 // Convert BTC to satoshis
	} else {
		multiplier = 100 // Convert fiat to cents
	}

	// Round to nearest integer
	smallestUnit := int64(amount*multiplier + 0.5)
	return smallestUnit, nil
}
