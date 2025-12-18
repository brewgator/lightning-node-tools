package utils

import (
	"crypto/sha256"
	"fmt"
	"regexp"
	"strings"
)

// FormatSats formats satoshi amounts in a human-readable way
// Uses the more precise format suitable for both tools
func FormatSats(amount int64) string {
	if amount >= 100000000 {
		// Show in BTC for amounts >= 1 BTC
		return fmt.Sprintf("%.8f BTC", float64(amount)/100000000)
	} else if amount >= 1000000 {
		// Show in millions for amounts >= 1M sats
		return fmt.Sprintf("%.2fM sats", float64(amount)/1000000)
	} else if amount >= 1000 {
		// Show in thousands for amounts >= 1K sats
		return fmt.Sprintf("%.1fK sats", float64(amount)/1000)
	}
	return fmt.Sprintf("%d sats", amount)
}

// FormatSatsCompact formats satoshi amounts in a compact way for tables
// Used by channel-manager for cleaner table display
func FormatSatsCompact(amount int64) string {
	if amount >= 100000000 {
		// Show in BTC for amounts >= 1 BTC
		return fmt.Sprintf("%.3f BTC", float64(amount)/100000000)
	} else if amount >= 1000000 {
		// Show in millions for amounts >= 1M sats
		return fmt.Sprintf("%.1fM", float64(amount)/1000000)
	} else if amount >= 1000 {
		// Show in thousands for amounts >= 1K sats
		return fmt.Sprintf("%.0fK", float64(amount)/1000)
	}
	return fmt.Sprintf("%d", amount)
}

// Base58 alphabet used in Bitcoin addresses
const base58Alphabet = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz"

// ValidateBitcoinAddress validates a Bitcoin address (Legacy, SegWit, and Taproot)
func ValidateBitcoinAddress(address string) bool {
	if len(address) == 0 {
		return false
	}

	// Bech32 addresses (SegWit v0 and Taproot v1)
	if strings.HasPrefix(address, "bc1") || strings.HasPrefix(address, "tb1") {
		return validateBech32Address(address)
	}

	// Legacy P2PKH and P2SH addresses
	if address[0] == '1' || address[0] == '3' || address[0] == 'm' || address[0] == 'n' || address[0] == '2' {
		return validateLegacyAddress(address)
	}

	return false
}

// validateBech32Address validates Bech32 encoded addresses (SegWit and Taproot)
func validateBech32Address(address string) bool {
	// Basic length and format checks
	if len(address) < 14 || len(address) > 74 {
		return false
	}

	// Check for valid Bech32 characters
	validChars := regexp.MustCompile(`^[a-z0-9]+$`)
	if !validChars.MatchString(address) {
		return false
	}

	// Split human readable part and data
	lastOne := strings.LastIndex(address, "1")
	if lastOne < 1 {
		return false
	}

	hrp := address[:lastOne]
	data := address[lastOne+1:]

	// Validate human readable part (bc or tb for mainnet/testnet)
	if hrp != "bc" && hrp != "tb" {
		return false
	}

	// Basic data length check
	if len(data) < 6 {
		return false
	}

	// For now, we'll do basic validation. Full Bech32 checksum validation
	// would require implementing the full Bech32 algorithm
	return true
}

// validateLegacyAddress validates Legacy P2PKH and P2SH addresses
func validateLegacyAddress(address string) bool {
	// Length check
	if len(address) < 26 || len(address) > 35 {
		return false
	}

	// Check all characters are valid Base58
	for _, char := range address {
		if !strings.ContainsRune(base58Alphabet, char) {
			return false
		}
	}

	// Decode Base58Check
	decoded, err := base58Decode(address)
	if err != nil {
		return false
	}

	// Should be 25 bytes (21 payload + 4 checksum)
	if len(decoded) != 25 {
		return false
	}

	// Verify checksum
	payload := decoded[:21]
	checksum := decoded[21:]

	hash1 := sha256.Sum256(payload)
	hash2 := sha256.Sum256(hash1[:])

	for i := 0; i < 4; i++ {
		if checksum[i] != hash2[i] {
			return false
		}
	}

	// Check version byte
	version := decoded[0]
	switch version {
	case 0x00, 0x05: // Mainnet P2PKH, P2SH
		return true
	case 0x6f, 0xc4: // Testnet P2PKH, P2SH
		return true
	default:
		return false
	}
}

// base58Decode decodes a Base58 string
func base58Decode(input string) ([]byte, error) {
	if len(input) == 0 {
		return nil, fmt.Errorf("empty input")
	}

	// Count leading zeros
	zeros := 0
	for i := 0; i < len(input) && input[i] == '1'; i++ {
		zeros++
	}

	// Convert to big integer
	result := make([]byte, 0, len(input))

	for _, char := range input {
		index := strings.IndexRune(base58Alphabet, char)
		if index == -1 {
			return nil, fmt.Errorf("invalid character: %c", char)
		}

		// Multiply result by 58 and add index
		carry := index
		for i := len(result) - 1; i >= 0; i-- {
			carry += int(result[i]) * 58
			result[i] = byte(carry & 0xff)
			carry >>= 8
		}

		for carry > 0 {
			result = append([]byte{byte(carry & 0xff)}, result...)
			carry >>= 8
		}
	}

	// Add leading zeros
	for i := 0; i < zeros; i++ {
		result = append([]byte{0}, result...)
	}

	return result, nil
}

// ValidateXPub validates an extended public key (xpub, ypub, zpub)
func ValidateXPub(xpub string) bool {
	if len(xpub) == 0 {
		return false
	}

	// Check prefix
	validPrefixes := []string{"xpub", "ypub", "zpub", "tpub", "upub", "vpub"}
	validPrefix := false
	for _, prefix := range validPrefixes {
		if strings.HasPrefix(xpub, prefix) {
			validPrefix = true
			break
		}
	}

	if !validPrefix {
		return false
	}

	// Should be approximately 111 characters for most xpubs
	if len(xpub) < 100 || len(xpub) > 120 {
		return false
	}

	// Check all characters are valid Base58
	for _, char := range xpub {
		if !strings.ContainsRune(base58Alphabet, char) {
			return false
		}
	}

	// Basic validation - more detailed checksum validation would require
	// a more robust Base58Check implementation
	return true
}
