package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/gosdk/core/transaction"
	"github.com/0chain/gosdk/zcncore"
	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// GetBalanceAndNonce returns balance and nonce for a wallet
func GetBalanceAndNonce(t *test.SystemTest, cliConfigFilename, wallet string) (float64, int64, error) {
	output, err := cliutils.RunCommand(t, "./zwallet getbalance --silent --json "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	if err != nil {
		return 0, 0, err
	}
	var balanceResp = struct {
		ZCN   string `json:"zcn"`
		Nonce int64  `json:"nonce"`
	}{}
	if err := json.Unmarshal([]byte(output[0]), &balanceResp); err != nil {
		return 0, 0, err
	}
	balanceFloat, err := strconv.ParseFloat(balanceResp.ZCN, 64)
	if err != nil {
		return 0, 0, err
	}
	return float64(int(balanceFloat*100)) / 100, balanceResp.Nonce, nil
}

// GetNonceForWallet returns current nonce for a wallet from chain
func GetNonceForWallet(t *test.SystemTest, cliConfigFilename, wallet string) (int64, error) {
	_, nonce, err := GetBalanceAndNonce(t, cliConfigFilename, wallet)
	return nonce, err
}

// EnsureWalletFunded checks if wallet has minimum balance and funds it via faucet if needed
func EnsureWalletFunded(t *test.SystemTest, wallet, cliConfigFilename string, minBalance float64) error {
	balance, _, err := GetBalanceAndNonce(t, cliConfigFilename, wallet)
	if err != nil {
		t.Logf("Wallet %s not found on chain or error getting balance, funding via faucet...", wallet)
		_, err = ExecuteFaucetWithTokensForWallet(t, wallet, cliConfigFilename, 10)
		if err != nil {
			return fmt.Errorf("failed to fund wallet %s: %w", wallet, err)
		}
		return nil
	}
	if balance < minBalance {
		t.Logf("Wallet %s has %.2f ZCN, need %.2f ZCN, funding via faucet...", wallet, balance, minBalance)
		for balance < minBalance {
			_, err = ExecuteFaucetWithTokensForWallet(t, wallet, cliConfigFilename, 10)
			if err != nil {
				return fmt.Errorf("failed to fund wallet %s: %w", wallet, err)
			}
			cliutils.Wait(t, 2*time.Second)
			balance, _, err = GetBalanceAndNonce(t, cliConfigFilename, wallet)
			if err != nil {
				return fmt.Errorf("failed to get balance for wallet %s: %w", wallet, err)
			}
		}
	}
	t.Logf("Wallet %s has sufficient balance: %.2f ZCN", wallet, balance)
	return nil
}

// CreateAndFundWallet creates a wallet and ensures it has minimum balance
func CreateAndFundWallet(t *test.SystemTest, cliConfigFilename string, minBalance float64) error {
	return CreateAndFundWalletForName(t, EscapedTestName(t), cliConfigFilename, minBalance)
}

// CreateAndFundWalletForName creates a named wallet and ensures it has minimum balance
func CreateAndFundWalletForName(t *test.SystemTest, name, cliConfigFilename string, minBalance float64) error {
	_, err := CreateWalletForName(t, cliConfigFilename, name)
	if err != nil {
		return fmt.Errorf("failed to create wallet %s: %w", name, err)
	}
	return EnsureWalletFunded(t, name, cliConfigFilename, minBalance)
}

// ExecuteFaucetWithTokens executes faucet command with given tokens.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func ExecuteFaucetWithTokens(t *test.SystemTest, cliConfigFilename string, tokens float64) ([]string, error) {
	return ExecuteFaucetWithTokensForWallet(t, EscapedTestName(t), cliConfigFilename, tokens)
}

// ExecuteFaucetWithTokensForWallet executes faucet command with given tokens and wallet.
// The faucet gives 1 ZCN per call regardless of requested amount, so we call it
// multiple times to accumulate the needed balance.
func ExecuteFaucetWithTokensForWallet(t *test.SystemTest, wallet, cliConfigFilename string, tokens float64) ([]string, error) {
	numCalls := int(tokens)
	if numCalls < 3 {
		numCalls = 3
	}
	if numCalls > 10 {
		numCalls = 10
	}

	var lastOutput []string
	var lastErr error

	for i := 0; i < numCalls; i++ {
		t.Logf("Executing faucet (%d/%d)...", i+1, numCalls)
		nonce, err := GetNonceForWallet(t, cliConfigFilename, wallet)
		nonceParam := ""
		if err == nil {
			nonceParam = fmt.Sprintf(" --withNonce %d", nonce+1)
		}
		lastOutput, lastErr = cliutils.RunCommand(t, fmt.Sprintf("./zwallet faucet --methodName "+
			"pour --tokens 1 --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s%s",
			wallet,
			cliConfigFilename,
			nonceParam,
		), 3, time.Second*5)
		if lastErr != nil {
			t.Logf("Faucet call %d failed: %v", i+1, lastErr)
		}
	}

	return lastOutput, lastErr
}

func CreateWallet(t *test.SystemTest, cliConfigFilename string, opt ...createWalletOptionFunc) ([]string, error) {
	return CreateWalletForName(t, cliConfigFilename, EscapedTestName(t), opt...)
}

type createWalletOption struct {
	debugLogs bool
}

type createWalletOptionFunc func(*createWalletOption)

func CreateWalletForName(t *test.SystemTest, cliConfigFilename, name string, opts ...createWalletOptionFunc) ([]string, error) {
	t.Logf("creating wallet...")
	regOpt := &createWalletOption{}
	for _, opt := range opts {
		opt(regOpt)
	}

	if regOpt.debugLogs {
		return cliutils.RunCommand(t, "./zwallet create-wallet "+
			"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	}
	output, err := cliutils.RunCommand(t, "./zwallet create-wallet --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	if err != nil {
		return output, err
	}

	output, err = ExecuteFaucetWithTokensForWallet(t, name, cliConfigFilename, 5)
	t.Logf("faucet output: %v", output)
	// Don't fail wallet creation if faucet is empty - the wallet was created successfully
	// The caller can handle faucet errors separately if needed
	if err != nil && strings.Contains(strings.Join(output, "\n"), "faucet has no tokens") {
		t.Logf("Warning: Faucet is empty, but wallet was created successfully")
		return output, nil
	}
	return output, err
}

func GetWalletForName(t *test.SystemTest, cliConfigFilename, name string) (*climodel.Wallet, error) {
	t.Logf("Getting wallet...")
	output, err := cliutils.RunCommand(t, "./zbox getwallet --json --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)

	if err != nil {
		return nil, err
	}

	require.Len(t, output, 1)

	var wallet *climodel.Wallet

	err = json.Unmarshal([]byte(output[0]), &wallet)
	if err != nil {
		t.Errorf("failed to unmarshal the result into wallet")
		return nil, err
	}

	return wallet, err
}

func GetFullWalletForName(t *test.SystemTest, cliConfigFilename, name string) (*model.Wallet, error) {
	t.Logf("Getting wallet...")
	output, err := cliutils.RunCommand(t, "./zbox getwallet --json --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)

	if err != nil {
		return nil, err
	}

	require.Len(t, output, 1)

	var wallet *model.Wallet

	err = json.Unmarshal([]byte(output[0]), &wallet)
	if err != nil {
		t.Errorf("failed to unmarshal the result into wallet")
		return nil, err
	}

	return wallet, err
}

func SetupWalletWithCustomTokens(t *test.SystemTest, configPath string, tokens float64) []string {
	output, err := CreateWallet(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	_, err = ExecuteFaucetWithTokens(t, configPath, tokens)
	require.Nil(t, err, strings.Join(output, "\n"))

	return output
}

func StakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string, retry bool) ([]string, error) {
	t.Log("Staking tokens...")
	nonce, err := GetNonceForWallet(t, cliConfigFilename, wallet)
	nonceParam := ""
	if err == nil {
		nonceParam = fmt.Sprintf(" --withNonce %d", nonce+1)
	}
	cmd := fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s%s", params, wallet, cliConfigFilename, nonceParam)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func UnstakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string) ([]string, error) {
	t.Log("Unlocking tokens from stake pool...")
	nonce, err := GetNonceForWallet(t, cliConfigFilename, wallet)
	nonceParam := ""
	if err == nil {
		nonceParam = fmt.Sprintf(" --withNonce %d", nonce+1)
	}
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s%s", params, wallet, cliConfigFilename, nonceParam), 3, time.Second*2)
}

func UpdateStorageSCConfig(t *test.SystemTest, walletName string, param map[string]string, retry bool) ([]string, error) {
	t.Logf("Updating storage config...")
	p := createKeyValueParams(param)
	nonce, err := GetNonceForWallet(t, configPath, walletName)
	nonceParam := ""
	if err == nil {
		nonceParam = fmt.Sprintf(" --withNonce %d", nonce+1)
	}
	cmd := fmt.Sprintf(
		"./zwallet sc-update-config %s --silent --wallet %s --configDir ./config --config %s%s",
		p,
		walletName+"_wallet.json",
		configPath,
		nonceParam,
	)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func createKeyValueParams(params map[string]string) string {
	keys := "--keys \""
	values := "--values \""
	first := true
	for k, v := range params {
		if first {
			first = false
		} else {
			keys += ","
			values += ","
		}
		keys += " " + k
		values += " " + v
	}
	keys += "\""
	values += "\""
	return keys + " " + values
}

func CollectRewards(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return CollectRewardsForWallet(t, cliConfigFilename, params, EscapedTestName(t), retry)
}

func CollectRewardsForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("collecting rewards...")
	nonce, err := GetNonceForWallet(t, cliConfigFilename, wallet)
	nonceParam := ""
	if err == nil {
		nonceParam = fmt.Sprintf(" --withNonce %d", nonce+1)
	}
	cmd := fmt.Sprintf("./zbox collect-reward %s --silent --wallet %s_wallet.json --configDir ./config --config %s%s", params, wallet, cliConfigFilename, nonceParam)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func GetBalanceZCN(t *test.SystemTest, cliConfigFilename string, walletName ...string) (float64, error) {
	cliutils.Wait(t, 5*time.Second)
	var (
		output []string
		err    error
	)
	if len(walletName) > 0 && walletName[0] != "" {
		output, err = GetBalanceForWalletJSON(t, cliConfigFilename, walletName[0])
		if err != nil {
			return 0, err
		}
	} else {
		output, err = GetBalanceForWalletJSON(t, cliConfigFilename, EscapedTestName(t))
		if err != nil {
			return 0, err
		}
	}

	var balance = struct {
		ZCN string `json:"zcn"`
	}{}

	if err := json.Unmarshal([]byte(output[0]), &balance); err != nil {
		return 0, err
	}

	balanceFloat, err := strconv.ParseFloat(balance.ZCN, 64)
	if err != nil {
		return 0, err
	}

	// round up to 2 decimal places
	return float64(int(balanceFloat*100)) / 100, nil
}

func GetBalanceForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func GetBalanceForWalletJSON(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent --json "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func GetBalanceFromSharders(t *test.SystemTest, clientId string) int64 {
	output, err := getSharders(t, configPath)
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 1)

	// Scan for "MagicBlock Sharders" line (zwallet may print wallet creation messages before it)
	found := false
	for index, line := range output {
		if line == "MagicBlock Sharders" {
			found = true
			output = output[index:]
			break
		}
	}
	require.True(t, found, "MagicBlock Sharders not found in getSharders output: %v", strings.Join(output, "\n"))
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]*climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	// Get base URL for API calls.
	sharderBaseURLs := getAllSharderBaseURLs(sharders)
	res, err := apiGetBalance(t, sharderBaseURLs[0], clientId)
	require.Nil(t, err, "error getting balance")

	if res.StatusCode == 400 {
		return 0
	}
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get balance")
	require.NotNil(t, res.Body, "Balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body")

	var startBalance climodel.Balance
	err = json.Unmarshal(resBody, &startBalance)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

	return startBalance.Balance
}

func getAllSharderBaseURLs(sharders map[string]*climodel.Sharder) []string {
	sharderURLs := make([]string, 0)
	for _, sharder := range sharders {
		sharderURLs = append(sharderURLs, getNodeBaseURL(sharder.Host, sharder.Port))
	}
	return sharderURLs
}

func apiGetBalance(t *test.SystemTest, sharderBaseURL, clientID string) (*http.Response, error) {
	t.Logf("Getting balance for %s...", clientID)
	return http.Get(sharderBaseURL + "/v1/client/get/balance?client_id=" + clientID)
}

// ReadPoolLock locks tokens in the read pool for a wallet using gosdk directly.
// This is needed because zbox CLI does not expose an rp-lock command.
// tokens is in ZCN units (e.g., 0.5 = 0.5 ZCN).
func ReadPoolLock(t *test.SystemTest, cliConfigFilename, walletName string, tokens float64) error {
	t.Logf("Locking %.4f ZCN in read pool for wallet %s...", tokens, walletName)

	walletFile := fmt.Sprintf("./config/%s_wallet.json", walletName)
	configFile := fmt.Sprintf("./config/%s", cliConfigFilename)

	err := InitSDK(walletFile, configFile)
	if err != nil {
		return fmt.Errorf("ReadPoolLock: InitSDK failed: %w", err)
	}

	// 1 ZCN = 10^10 SAS
	amountSAS := uint64(tokens * 1e10)

	hash, _, _, _, err := transaction.SmartContractTxnValue(
		zcncore.StorageSmartContractAddress,
		transaction.SmartContractTxnData{
			Name:      transaction.STORAGESC_READ_POOL_LOCK,
			InputArgs: map[string]string{},
		},
		amountSAS,
		true,
	)
	if err != nil {
		return fmt.Errorf("ReadPoolLock SC txn failed (hash=%s): %w", hash, err)
	}

	t.Logf("ReadPoolLock successful, txn hash: %s", hash)
	return nil
}
