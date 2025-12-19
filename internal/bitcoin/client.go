package bitcoin

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strconv"
)

// Client represents a Bitcoin Core RPC client
type Client struct{}

// NewClient creates a new Bitcoin Core client
func NewClient() (*Client, error) {
	// Test Bitcoin Core connectivity (without wallet)
	cmd := exec.Command("bitcoin-cli", "getblockchaininfo")
	_, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Bitcoin Core: %w", err)
	}
	return &Client{}, nil
}

// RunBitcoinCLI executes bitcoin-cli commands and returns the output.
//
// Security Note: While this function doesn't currently accept user input directly,
// callers must ensure that any user-provided data (e.g., addresses, labels) is
// properly validated before being passed to bitcoin-cli. The current codebase
// validates addresses using ValidateAddress() before import, and all numeric
// parameters are type-safe. For future enhancements, consider:
//   - Using a structured RPC client library instead of command-line execution
//   - Adding explicit sanitization for string parameters
//   - Implementing allowlists for command names
func RunBitcoinCLI(args ...string) ([]byte, error) {
	// Add wallet parameter for our tracking wallet
	fullArgs := []string{"-rpcwallet=tracker_watchonly"}
	fullArgs = append(fullArgs, args...)

	cmd := exec.Command("bitcoin-cli", fullArgs...)
	output, err := cmd.Output()
	if err != nil {
		// If there's an error, try to get stderr for more details
		if exitError, ok := err.(*exec.ExitError); ok {
			// Include stderr in the error message
			return nil, fmt.Errorf("bitcoin-cli command failed: %v, stderr: %s", err, string(exitError.Stderr))
		}
		return nil, fmt.Errorf("bitcoin-cli command failed: %v", err)
	}
	return output, nil
}

// GetBlockchainInfo retrieves general blockchain information
func (c *Client) GetBlockchainInfo() (*BlockchainInfo, error) {
	output, err := RunBitcoinCLI("getblockchaininfo")
	if err != nil {
		return nil, err
	}

	var info BlockchainInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// GetAddressBalance gets the current balance for a specific address by summing UTXOs
// Note: This requires the address to be imported as watch-only
func (c *Client) GetAddressBalance(address string) (int64, error) {
	// Import address as watch-only if not already imported
	err := c.ImportAddress(address)
	if err != nil {
		// Import might fail if address is already imported, which is fine
		// We'll continue and try to get the balance anyway
	}

	// Get UTXOs to calculate actual spendable balance
	utxos, err := c.GetAddressUTXOs(address)
	if err != nil {
		return 0, err
	}

	// Sum up UTXO values to get current balance (convert BTC to satoshis)
	var balanceBTC float64
	for _, utxo := range utxos {
		balanceBTC += utxo.Amount
	}

	// Convert BTC to satoshis
	balanceSatoshis := int64(balanceBTC * 100000000)
	return balanceSatoshis, nil
}

// ImportAddress imports an address as watch-only using descriptors
func (c *Client) ImportAddress(address string) error {
	// Get the descriptor with checksum for this address
	descriptorInfo, err := c.GetDescriptorInfo(address)
	if err != nil {
		return fmt.Errorf("failed to get descriptor info: %w", err)
	}

	// Import using descriptors with full history (timestamp: 0 forces rescan from genesis)
	descriptorJSON := fmt.Sprintf(`[{"desc":"%s","timestamp":0,"watchonly":true}]`, descriptorInfo.Descriptor)
	_, err = RunBitcoinCLI("importdescriptors", descriptorJSON)
	return err
}

// GetDescriptorInfo gets descriptor information for an address
func (c *Client) GetDescriptorInfo(address string) (*DescriptorInfo, error) {
	// Use non-wallet command to get descriptor info
	cmd := exec.Command("bitcoin-cli", "getdescriptorinfo", fmt.Sprintf("addr(%s)", address))
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var info DescriptorInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// GetAddressUTXOs gets unspent transaction outputs for an address
func (c *Client) GetAddressUTXOs(address string) ([]UTXO, error) {
	// First import the address if not already imported
	err := c.ImportAddress(address)
	if err != nil {
		// Continue even if import fails (might already be imported)
	}

	// Get unspent outputs
	output, err := RunBitcoinCLI("listunspent", "0", "9999999", fmt.Sprintf("[\"%s\"]", address))
	if err != nil {
		return nil, err
	}

	var utxos []UTXO
	if err := json.Unmarshal(output, &utxos); err != nil {
		return nil, err
	}

	return utxos, nil
}

// GetAddressTransactions gets transaction history for an address
func (c *Client) GetAddressTransactions(address string) ([]AddressTransaction, error) {
	// This requires Bitcoin Core with txindex=1
	// Use listtransactions to get transactions involving this address
	output, err := RunBitcoinCLI("listtransactions", "*", "1000", "0", "true")
	if err != nil {
		return nil, err
	}

	var allTxs []AddressTransaction
	if err := json.Unmarshal(output, &allTxs); err != nil {
		return nil, err
	}

	// Filter transactions for this specific address
	var addressTxs []AddressTransaction
	for _, tx := range allTxs {
		if tx.Address == address {
			addressTxs = append(addressTxs, tx)
		}
	}

	return addressTxs, nil
}

// ValidateAddress checks if an address is valid
func (c *Client) ValidateAddress(address string) (*AddressValidation, error) {
	output, err := RunBitcoinCLI("validateaddress", address)
	if err != nil {
		return nil, err
	}

	var validation AddressValidation
	if err := json.Unmarshal(output, &validation); err != nil {
		return nil, err
	}

	return &validation, nil
}

// GetAddressInfo gets detailed information about an address (Bitcoin Core 0.17+)
func (c *Client) GetAddressInfo(address string) (*AddressInfo, error) {
	output, err := RunBitcoinCLI("getaddressinfo", address)
	if err != nil {
		return nil, err
	}

	var info AddressInfo
	if err := json.Unmarshal(output, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// RescanBlockchain rescans the blockchain for transactions affecting watched addresses
// This is useful after importing new addresses
func (c *Client) RescanBlockchain(startHeight int64) error {
	startHeightStr := strconv.FormatInt(startHeight, 10)
	_, err := RunBitcoinCLI("rescanblockchain", startHeightStr)
	return err
}
