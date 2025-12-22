package multisig

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	"github.com/brewgator/lightning-node-tools/internal/db"
	"github.com/btcsuite/btcd/btcec/v2"
	"github.com/btcsuite/btcd/btcutil"
	"github.com/btcsuite/btcd/btcutil/hdkeychain"
	"github.com/btcsuite/btcd/chaincfg"
	"github.com/btcsuite/btcd/txscript"
)

// MultisigService handles multisig wallet operations
type MultisigService struct {
	db      *db.Database
	network *chaincfg.Params
}

// NewMultisigService creates a new multisig service
func NewMultisigService(database *db.Database) *MultisigService {
	return &MultisigService{
		db:      database,
		network: &chaincfg.MainNetParams, // Default to mainnet
	}
}

// ImportWalletConfig imports a wallet config JSON file into the database
func (ms *MultisigService) ImportWalletConfig(configJSON []byte) (*db.MultisigWallet, error) {
	var config db.WalletConfig
	if err := json.Unmarshal(configJSON, &config); err != nil {
		return nil, fmt.Errorf("failed to parse wallet config: %w", err)
	}

	// Validate the config
	if err := ms.validateWalletConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid wallet config: %w", err)
	}

	// Check if wallet with this UUID already exists
	existing, err := ms.db.GetMultisigWalletByUUID(config.UUID)
	if err != nil {
		return nil, fmt.Errorf("failed to check for existing wallet: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("wallet with UUID %s already exists", config.UUID)
	}

	// Convert config to database format
	wallet := &db.MultisigWallet{
		Name:                 config.Name,
		UUID:                 config.UUID,
		AddressType:          config.AddressType,
		Network:              config.Network,
		RequiredSigners:      config.Quorum.RequiredSigners,
		TotalSigners:         config.Quorum.TotalSigners,
		StartingAddressIndex: config.StartingAddressIndex,
		NextAddressIndex:     config.StartingAddressIndex,
		Active:               true,
		ExtendedPublicKeys:   make([]db.MultisigExtendedPubKey, len(config.ExtendedPublicKeys)),
	}

	for i, key := range config.ExtendedPublicKeys {
		wallet.ExtendedPublicKeys[i] = db.MultisigExtendedPubKey{
			Name:        key.Name,
			XPub:        key.XPub,
			BIP32Path:   key.BIP32Path,
			Fingerprint: key.XFP,
			KeyIndex:    i,
		}
	}

	// Insert wallet into database
	insertedWallet, err := ms.db.InsertMultisigWallet(wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to insert wallet: %w", err)
	}

	return insertedWallet, nil
}

// validateWalletConfig validates a wallet configuration
func (ms *MultisigService) validateWalletConfig(config *db.WalletConfig) error {
	if config.Name == "" {
		return fmt.Errorf("wallet name is required")
	}
	if config.UUID == "" {
		return fmt.Errorf("wallet UUID is required")
	}
	if config.AddressType != "P2SH" && config.AddressType != "P2WSH" {
		return fmt.Errorf("unsupported address type: %s", config.AddressType)
	}
	if config.Network != "mainnet" && config.Network != "testnet" {
		return fmt.Errorf("unsupported network: %s", config.Network)
	}
	if config.Quorum.RequiredSigners <= 0 || config.Quorum.TotalSigners <= 0 {
		return fmt.Errorf("invalid quorum configuration")
	}
	if config.Quorum.RequiredSigners > config.Quorum.TotalSigners {
		return fmt.Errorf("required signers cannot exceed total signers")
	}
	if len(config.ExtendedPublicKeys) != config.Quorum.TotalSigners {
		return fmt.Errorf("number of extended public keys must match total signers")
	}

	// Validate each extended public key
	for i, key := range config.ExtendedPublicKeys {
		if key.Name == "" {
			return fmt.Errorf("extended public key %d: name is required", i)
		}
		if key.XPub == "" {
			return fmt.Errorf("extended public key %d: xpub is required", i)
		}
		if key.BIP32Path == "" {
			return fmt.Errorf("extended public key %d: BIP32 path is required", i)
		}

		// Validate xpub format
		if _, err := hdkeychain.NewKeyFromString(key.XPub); err != nil {
			return fmt.Errorf("extended public key %d: invalid xpub format: %w", i, err)
		}
	}

	return nil
}

// DeriveAddress derives a multisig address at a specific index
func (ms *MultisigService) DeriveAddress(wallet *db.MultisigWallet, addressIndex int) (*db.MultisigAddress, error) {
	// Set network parameters
	var netParams *chaincfg.Params
	if wallet.Network == "testnet" {
		netParams = &chaincfg.TestNet3Params
	} else {
		netParams = &chaincfg.MainNetParams
	}

	// Derive public keys for each extended public key
	pubKeys := make([]*btcec.PublicKey, len(wallet.ExtendedPublicKeys))
	for i, extKey := range wallet.ExtendedPublicKeys {
		// Parse the extended public key
		masterKey, err := hdkeychain.NewKeyFromString(extKey.XPub)
		if err != nil {
			return nil, fmt.Errorf("failed to parse extended public key %d: %w", i, err)
		}

		// Derive the child key at the specified index
		// For most multisig wallets, we derive m/0/addressIndex
		childKey, err := masterKey.Derive(0)
		if err != nil {
			return nil, fmt.Errorf("failed to derive child key (0) for key %d: %w", i, err)
		}

		finalKey, err := childKey.Derive(uint32(addressIndex))
		if err != nil {
			return nil, fmt.Errorf("failed to derive final key (%d) for key %d: %w", addressIndex, i, err)
		}

		// Get the public key
		pubKey, err := finalKey.ECPubKey()
		if err != nil {
			return nil, fmt.Errorf("failed to get public key for key %d: %w", i, err)
		}

		pubKeys[i] = pubKey
	}

	// Sort public keys (required for deterministic address generation)
	sort.Slice(pubKeys, func(i, j int) bool {
		return pubKeys[i].SerializeCompressed()[0] < pubKeys[j].SerializeCompressed()[0] ||
			(pubKeys[i].SerializeCompressed()[0] == pubKeys[j].SerializeCompressed()[0] &&
				hex.EncodeToString(pubKeys[i].SerializeCompressed()) < hex.EncodeToString(pubKeys[j].SerializeCompressed()))
	})

	// Convert public keys to address format
	pubKeyAddrs := make([]*btcutil.AddressPubKey, len(pubKeys))
	for i, pubKey := range pubKeys {
		pubKeyAddr, err := btcutil.NewAddressPubKey(pubKey.SerializeCompressed(), netParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create address from public key %d: %w", i, err)
		}
		pubKeyAddrs[i] = pubKeyAddr
	}

	// Create multisig redeem script
	redeemScript, err := txscript.MultiSigScript(pubKeyAddrs, wallet.RequiredSigners)
	if err != nil {
		return nil, fmt.Errorf("failed to create multisig script: %w", err)
	}

	var address string
	var scriptType string

	switch wallet.AddressType {
	case "P2SH":
		// Create P2SH address
		scriptAddr, err := btcutil.NewAddressScriptHash(redeemScript, netParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create P2SH address: %w", err)
		}
		address = scriptAddr.EncodeAddress()
		scriptType = "P2SH"

	case "P2WSH":
		// Create P2WSH address
		scriptHash := sha256.Sum256(redeemScript)
		witnessAddr, err := btcutil.NewAddressWitnessScriptHash(scriptHash[:], netParams)
		if err != nil {
			return nil, fmt.Errorf("failed to create P2WSH address: %w", err)
		}
		address = witnessAddr.EncodeAddress()
		scriptType = "P2WSH"

	default:
		return nil, fmt.Errorf("unsupported address type: %s", wallet.AddressType)
	}

	// Create MultisigAddress struct
	multisigAddr := &db.MultisigAddress{
		WalletID:     wallet.ID,
		Address:      address,
		AddressIndex: addressIndex,
		ScriptType:   scriptType,
		RedeemScript: hex.EncodeToString(redeemScript),
		Active:       true,
	}

	return multisigAddr, nil
}

// GenerateAddresses generates a range of addresses for a multisig wallet
func (ms *MultisigService) GenerateAddresses(walletID int64, count int) ([]db.MultisigAddress, error) {
	// Get the wallet
	wallet, err := ms.db.GetMultisigWalletByID(walletID)
	if err != nil {
		return nil, fmt.Errorf("failed to get wallet: %w", err)
	}
	if wallet == nil {
		return nil, fmt.Errorf("wallet not found")
	}

	var addresses []db.MultisigAddress
	startIndex := wallet.NextAddressIndex

	for i := 0; i < count; i++ {
		addressIndex := startIndex + i

		// Check if this address already exists
		existingAddresses, err := ms.db.GetMultisigAddressesByWalletID(walletID)
		if err != nil {
			return nil, fmt.Errorf("failed to get existing addresses: %w", err)
		}

		exists := false
		for _, existing := range existingAddresses {
			if existing.AddressIndex == addressIndex {
				exists = true
				addresses = append(addresses, existing)
				break
			}
		}

		if !exists {
			// Derive the address
			addr, err := ms.DeriveAddress(wallet, addressIndex)
			if err != nil {
				return nil, fmt.Errorf("failed to derive address at index %d: %w", addressIndex, err)
			}

			// Insert into database
			insertedAddr, err := ms.db.InsertMultisigAddress(addr)
			if err != nil {
				return nil, fmt.Errorf("failed to insert address: %w", err)
			}

			addresses = append(addresses, *insertedAddr)
		}
	}

	// Update next address index
	err = ms.db.UpdateMultisigWalletNextAddressIndex(walletID, startIndex+count)
	if err != nil {
		return nil, fmt.Errorf("failed to update next address index: %w", err)
	}

	return addresses, nil
}

// GetWalletAddresses gets all addresses for a wallet
func (ms *MultisigService) GetWalletAddresses(walletID int64) ([]db.MultisigAddress, error) {
	return ms.db.GetMultisigAddressesByWalletID(walletID)
}

// GetWallets gets all active multisig wallets
func (ms *MultisigService) GetWallets() ([]db.MultisigWallet, error) {
	return ms.db.GetMultisigWallets()
}

// GetWalletByID gets a specific wallet by ID
func (ms *MultisigService) GetWalletByID(id int64) (*db.MultisigWallet, error) {
	return ms.db.GetMultisigWalletByID(id)
}

// DeleteWallet marks a wallet as inactive
func (ms *MultisigService) DeleteWallet(id int64) error {
	return ms.db.DeleteMultisigWallet(id)
}

// UpdateLastScanned updates the last scanned timestamp for a wallet
func (ms *MultisigService) UpdateLastScanned(id int64) error {
	return ms.db.UpdateMultisigWalletLastScanned(id, time.Now())
}

// GetAddressesForTracking returns all multisig addresses as regular onchain addresses for tracking
func (ms *MultisigService) GetAddressesForTracking() ([]db.OnchainAddress, error) {
	wallets, err := ms.db.GetMultisigWallets()
	if err != nil {
		return nil, fmt.Errorf("failed to get multisig wallets: %w", err)
	}

	var addresses []db.OnchainAddress
	for _, wallet := range wallets {
		multisigAddresses, err := ms.db.GetMultisigAddressesByWalletID(wallet.ID)
		if err != nil {
			return nil, fmt.Errorf("failed to get addresses for wallet %d: %w", wallet.ID, err)
		}

		for _, addr := range multisigAddresses {
			onchainAddr := db.OnchainAddress{
				ID:      addr.ID,
				Address: addr.Address,
				Label:   fmt.Sprintf("%s [%d/%d] - Index %d", wallet.Name, wallet.RequiredSigners, wallet.TotalSigners, addr.AddressIndex),
				Active:  addr.Active,
			}
			addresses = append(addresses, onchainAddr)
		}
	}

	return addresses, nil
}
