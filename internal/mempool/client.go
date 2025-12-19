package mempool

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"
)

// Client represents a Mempool.space API client
type Client struct {
	baseURL    string
	httpClient *http.Client
	limiter    *RateLimiter
}

// NewClient creates a new Mempool.space API client
func NewClient(baseURL string) *Client {
	if baseURL == "" {
		baseURL = "https://mempool.space/api"
	}
	
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		limiter: NewRateLimiter(10, time.Minute), // 10 requests per minute
	}
}

// GetAddressStats gets comprehensive address statistics
func (c *Client) GetAddressStats(address string) (*AddressStats, error) {
	if err := c.limiter.Wait(); err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/address/%s", c.baseURL, address)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch address stats: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mempool API error %d: %s", resp.StatusCode, string(body))
	}
	
	var stats AddressStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode address stats: %w", err)
	}
	
	return &stats, nil
}

// GetAddressUTXOs gets unspent transaction outputs for an address
func (c *Client) GetAddressUTXOs(address string) ([]UTXO, error) {
	if err := c.limiter.Wait(); err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/address/%s/utxo", c.baseURL, address)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch address UTXOs: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mempool API error %d: %s", resp.StatusCode, string(body))
	}
	
	var utxos []UTXO
	if err := json.NewDecoder(resp.Body).Decode(&utxos); err != nil {
		return nil, fmt.Errorf("failed to decode UTXOs: %w", err)
	}
	
	return utxos, nil
}

// GetAddressTransactions gets transaction history for an address
func (c *Client) GetAddressTransactions(address string, after string) ([]Transaction, error) {
	if err := c.limiter.Wait(); err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/address/%s/txs", c.baseURL, address)
	if after != "" {
		url += "?after_txid=" + after
	}
	
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch address transactions: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mempool API error %d: %s", resp.StatusCode, string(body))
	}
	
	var transactions []Transaction
	if err := json.NewDecoder(resp.Body).Decode(&transactions); err != nil {
		return nil, fmt.Errorf("failed to decode transactions: %w", err)
	}
	
	return transactions, nil
}

// GetChainTips gets current blockchain information
func (c *Client) GetChainTips() (*ChainTips, error) {
	if err := c.limiter.Wait(); err != nil {
		return nil, err
	}
	
	url := fmt.Sprintf("%s/blocks/tip/height", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch chain tips: %w", err)
	}
	defer resp.Body.Close()
	
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("mempool API error %d: %s", resp.StatusCode, string(body))
	}
	
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}
	
	height, err := strconv.ParseInt(string(body), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("failed to parse block height: %w", err)
	}
	
	return &ChainTips{
		Height:    height,
		Timestamp: time.Now(),
	}, nil
}

// CalculateAddressBalance calculates total balance from UTXOs
func (c *Client) CalculateAddressBalance(address string) (int64, int64, error) {
	utxos, err := c.GetAddressUTXOs(address)
	if err != nil {
		return 0, 0, err
	}
	
	var balance int64
	for _, utxo := range utxos {
		balance += utxo.Value
	}
	
	return balance, int64(len(utxos)), nil
}

// ValidateAddress checks if an address is valid (basic check)
func (c *Client) ValidateAddress(address string) bool {
	// Basic validation - try to fetch address stats
	_, err := c.GetAddressStats(address)
	return err == nil
}