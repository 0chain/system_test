package sdk_tests

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
	"github.com/stretchr/testify/require"
)

// SDKTestConfig holds configuration for SDK tests
type SDKTestConfig struct {
	BlockWorker           string `yaml:"block_worker"`
	DefaultTestTimeout    string `yaml:"default_test_case_timeout"`
	SignatureScheme       string `yaml:"signature_scheme"`
	ChainID               string `yaml:"chain_id"`
	MaxTxnQuery           int    `yaml:"max_txn_query"`
	QuerySleepTime        int    `yaml:"query_sleep_time"`
	MinSubmit             int    `yaml:"min_submit"`
	MinConfirmation       int    `yaml:"min_confirmation"`
	TestWalletMnemonics   string `yaml:"test_wallet_mnemonics"`
}

// Shared variables used across SDK tests
var (
	initialisedWallets []json.RawMessage
	walletIdx          int64
	walletMutex        sync.Mutex
	parsedConfig       *SDKTestConfig // initialized in main_test.go
)

// GetParsedConfig returns the parsed configuration
func GetParsedConfig() *SDKTestConfig {
	return parsedConfig
}

// Wallet represents a test wallet
type Wallet struct {
	ClientID        string          `json:"client_id"`
	ClientKey       string          `json:"client_key"`
	Keys            []WalletKeyPair `json:"keys"`
	Mnemonics       string          `json:"mnemonics"`
	Version         string          `json:"version"`
	SignatureScheme string          `json:"signature_scheme,omitempty"`
}

// WalletKeyPair represents a key pair in the wallet
type WalletKeyPair struct {
	PublicKey  string `json:"public_key"`
	PrivateKey string `json:"private_key"`
}

// CreateWallet creates a new wallet for testing
func CreateWallet(t *test.SystemTest) (*Wallet, error) {
	walletMutex.Lock()
	defer walletMutex.Unlock()

	var walletJSON string
	var err error

	// Try to get wallet from pool
	if len(initialisedWallets) > 0 && int(walletIdx) < len(initialisedWallets) {
		walletJSON = string(initialisedWallets[walletIdx])
		walletIdx++
	} else {
		// Create a new wallet offline
		walletJSON, err = zcncore.CreateWalletOffline()
		if err != nil {
			return nil, fmt.Errorf("failed to create wallet: %v", err)
		}
	}

	var wallet Wallet
	err = json.Unmarshal([]byte(walletJSON), &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wallet JSON: %v", err)
	}

	return &wallet, nil
}

// CreateWalletFromMnemonic creates a wallet from a mnemonic phrase
func CreateWalletFromMnemonic(t *test.SystemTest, mnemonic string) (*Wallet, error) {
	walletJSON, err := zcncore.RecoverOfflineWallet(mnemonic)
	if err != nil {
		return nil, fmt.Errorf("failed to recover wallet from mnemonic: %v", err)
	}

	var wallet Wallet
	err = json.Unmarshal([]byte(walletJSON), &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse recovered wallet JSON: %v", err)
	}

	return &wallet, nil
}

// CreateMultiKeyWallet creates a wallet with multiple keys
func CreateMultiKeyWallet(t *test.SystemTest, numKeys int) (*Wallet, error) {
	require.True(t, numKeys > 0, "Number of keys must be greater than 0")
	
	// Create base wallet
	walletJSON, err := zcncore.CreateWalletOffline()
	if err != nil {
		return nil, fmt.Errorf("failed to create wallet: %v", err)
	}

	var wallet Wallet
	err = json.Unmarshal([]byte(walletJSON), &wallet)
	if err != nil {
		return nil, fmt.Errorf("failed to parse wallet JSON: %v", err)
	}

	// For multi-key wallet, we would need SDK support to add additional keys
	// For now, this creates a standard wallet that can be extended
	// when multi-key functionality is implemented in the SDK

	return &wallet, nil
}

// SetWallet sets the wallet as the current wallet for SDK operations
func SetWallet(t *test.SystemTest, wallet *Wallet) error {
	walletJSON, err := json.Marshal(wallet)
	if err != nil {
		return fmt.Errorf("failed to marshal wallet: %v", err)
	}

	// SetWalletInfo requires (walletJSON, signatureScheme, isWASM)
	signatureScheme := wallet.SignatureScheme
	if signatureScheme == "" && parsedConfig != nil {
		signatureScheme = parsedConfig.SignatureScheme
	}

	err = zcncore.SetWalletInfo(string(walletJSON), signatureScheme, false)
	if err != nil {
		return fmt.Errorf("failed to set wallet info: %v", err)
	}

	return nil
}

// GetBalance retrieves the balance for a wallet (simplified version)
func GetBalance(t *test.SystemTest, wallet *Wallet) (int64, error) {
	err := SetWallet(t, wallet)
	if err != nil {
		return 0, err
	}

	// Note: Actual balance retrieval would require callback handlers
	// For now, this is a placeholder that returns 0
	// Real implementation would use zcncore balance retrieval with callbacks
	t.Logf("Balance retrieval requires callback implementation")
	return 0, nil
}

// AssertWalletValid asserts that a wallet is valid and has required fields
func AssertWalletValid(t *test.SystemTest, wallet *Wallet) {
	require.NotNil(t, wallet, "Wallet should not be nil")
	require.NotEmpty(t, wallet.ClientID, "Wallet ClientID should not be empty")
	require.NotEmpty(t, wallet.ClientKey, "Wallet ClientKey should not be empty")
	require.NotEmpty(t, wallet.Keys, "Wallet should have at least one key pair")
	require.NotEmpty(t, wallet.Keys[0].PublicKey, "Public key should not be empty")
	require.NotEmpty(t, wallet.Keys[0].PrivateKey, "Private key should not be empty")
}

// AssertMulti KeyWallet asserts that a wallet has multiple keys
func AssertMultiKeyWallet(t *test.SystemTest, wallet *Wallet, expectedKeyCount int) {
	AssertWalletValid(t, wallet)
	require.Len(t, wallet.Keys, expectedKeyCount, 
		fmt.Sprintf("Wallet should have exactly %d keys", expectedKeyCount))
	
	// Verify each key pair is unique
	publicKeys := make(map[string]bool)
	for i, keyPair := range wallet.Keys {
		require.NotEmpty(t, keyPair.PublicKey, 
			fmt.Sprintf("Key %d public key should not be empty", i))
		require.NotEmpty(t, keyPair.PrivateKey, 
			fmt.Sprintf("Key %d private key should not be empty", i))
		
		// Check for duplicate public keys
		require.False(t, publicKeys[keyPair.PublicKey], 
			fmt.Sprintf("Duplicate public key found at index %d", i))
		publicKeys[keyPair.PublicKey] = true
	}
}
