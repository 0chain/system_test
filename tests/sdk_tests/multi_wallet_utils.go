package sdk_tests

import (
	"encoding/json"
	"fmt"

	"github.com/0chain/gosdk/core/client"
	"github.com/0chain/gosdk/core/zcncrypto"
	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/util/test"
)

// ========== Multi-Wallet Helper Functions ==========

// MinerSCNodes list of nodes registered to the miner smart contract
type MinerSCNodes struct {
	Nodes []Node `json:"Nodes"`
}

type Node struct {
	Miner     Miner `json:"simple_miner"`
}

type Miner struct {
	ID string `json:"id"`
}

// GetMinerList returns a list of active miners
func GetMinerList(t *test.SystemTest) ([]string, error) {
	blob, err := zcncore.GetMiners(true, false, 20, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to get miners: %v", err)
	}

	var nodes MinerSCNodes
	err = json.Unmarshal(blob, &nodes)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal miners: %v", err)
	}

	if len(nodes.Nodes) == 0 {
		return nil, fmt.Errorf("no active miners found")
	}

	var miners []string
	for _, node := range nodes.Nodes {
		miners = append(miners, node.Miner.ID)
	}
	return miners, nil
}

// LoadWalletsFromPool loads a specified number of wallets from the pool
func LoadWalletsFromPool(count int) ([]*Wallet, error) {
	walletMutex.Lock()
	defer walletMutex.Unlock()

	if len(initialisedWallets) == 0 {
		return nil, fmt.Errorf("no wallets available in pool")
	}

	if count > len(initialisedWallets) {
		count = len(initialisedWallets)
	}

	wallets := make([]*Wallet, 0, count)
	for i := 0; i < count; i++ {
		var wallet Wallet
		err := json.Unmarshal(initialisedWallets[i], &wallet)
		if err != nil {
			return nil, fmt.Errorf("failed to parse wallet %d: %v", i, err)
		}
		wallets = append(wallets, &wallet)
	}

	return wallets, nil
}

// GetWalletPublicKey returns the public key of a wallet
func GetWalletPublicKey(wallet *Wallet) string {
	if wallet == nil || len(wallet.Keys) == 0 {
		return ""
	}
	return wallet.Keys[0].PublicKey
}

// GetWalletJSON converts a wallet to JSON string
func GetWalletJSON(wallet *Wallet) (string, error) {
	walletJSON, err := json.Marshal(wallet)
	if err != nil {
		return "", fmt.Errorf("failed to marshal wallet: %v", err)
	}
	return string(walletJSON), nil
}

// ConvertToZcncryptoWallet converts our Wallet type to zcncrypto.Wallet
func ConvertToZcncryptoWallet(wallet *Wallet) (*zcncrypto.Wallet, error) {
	walletJSON, err := GetWalletJSON(wallet)
	if err != nil {
		return nil, err
	}

	var zcnWallet zcncrypto.Wallet
	err = json.Unmarshal([]byte(walletJSON), &zcnWallet)
	if err != nil {
		return nil, fmt.Errorf("failed to convert to zcncrypto.Wallet: %v", err)
	}

	return &zcnWallet, nil
}

// AddWalletToSDK adds a wallet to the SDK's wallet pool
func AddWalletToSDK(t *test.SystemTest, wallet *Wallet) error {
	zcnWallet, err := ConvertToZcncryptoWallet(wallet)
	if err != nil {
		return err
	}

	client.AddWallet(*zcnWallet)
	t.Logf("Added wallet %s to SDK", wallet.ClientID)
	return nil
}

// RemoveWalletFromSDK removes a wallet from the SDK's wallet pool by ClientID
func RemoveWalletFromSDK(t *test.SystemTest, clientID string) {
	client.RemoveWallet(clientID)
	t.Logf("Removed wallet with clientID %s from SDK", clientID)
}

// GetWalletByKeyFromSDK retrieves a wallet from SDK by key (supports both ClientID and PublicKey)
func GetWalletByKeyFromSDK(key string) *zcncrypto.Wallet {
	return client.GetWalletByKey(key)
}

// SendTokens sends tokens from one wallet to another
func SendTokens(t *test.SystemTest, toClientID string, tokens uint64, desc string, key ...string) (string, error) {
	hash, _, _, _, err := zcncore.Send(toClientID, tokens, desc, key...)
	if err != nil {
		return "", fmt.Errorf("failed to send tokens: %v", err)
	}
	t.Logf("Sent %d tokens to %s, hash: %s", tokens, toClientID, hash)
	return hash, nil
}

// StakeToProvider stakes tokens to a provider (miner/sharder)
func StakeToProvider(t *test.SystemTest, providerID string, providerType zcncore.Provider, amount uint64, key ...string) (string, error) {
	hash, _, _, _, err := zcncore.MinerSCLock(providerID, providerType, amount, key...)
	if err != nil {
		return "", fmt.Errorf("failed to stake to provider: %v", err)
	}
	t.Logf("Staked %d tokens to provider %s, hash: %s", amount, providerID, hash)
	return hash, nil
}

// UnstakeFromProvider unstakes tokens from a provider
func UnstakeFromProvider(t *test.SystemTest, providerID string, providerType zcncore.Provider, key ...string) (string, error) {
	hash, _, _, _, err := zcncore.MinerSCUnlock(providerID, providerType, key...)
	if err != nil {
		return "", fmt.Errorf("failed to unstake from provider: %v", err)
	}
	t.Logf("Unstaked from provider %s, hash: %s", providerID, hash)
	return hash, nil
}


