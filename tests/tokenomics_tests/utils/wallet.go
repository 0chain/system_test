package utils

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/model"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// ExecuteFaucetWithTokens executes faucet command with given tokens.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func ExecuteFaucetWithTokens(t *test.SystemTest, cliConfigFilename string, tokens float64) ([]string, error) {
	return ExecuteFaucetWithTokensForWallet(t, EscapedTestName(t), cliConfigFilename, tokens)
}

// ExecuteFaucetWithTokensForWallet executes faucet command with given tokens and wallet.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func ExecuteFaucetWithTokensForWallet(t *test.SystemTest, wallet, cliConfigFilename string, tokens float64) ([]string, error) {
	t.Logf("Executing faucet...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet faucet --methodName "+
		"pour --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
		tokens,
		wallet,
		cliConfigFilename,
	), 3, time.Second*5)
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
	cmd := fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func UnstakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string) ([]string, error) {
	t.Log("Unlocking tokens from stake pool...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}

func UpdateStorageSCConfig(t *test.SystemTest, walletName string, param map[string]string, retry bool) ([]string, error) {
	t.Logf("Updating storage config...")
	p := createKeyValueParams(param)
	cmd := fmt.Sprintf(
		"./zwallet sc-update-config %s --silent --wallet %s --configDir ./config --config %s",
		p,
		walletName+"_wallet.json",
		configPath,
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
	cmd := fmt.Sprintf("./zbox collect-reward %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
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
