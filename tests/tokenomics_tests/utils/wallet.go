package utils

import (
	"encoding/json"
	"fmt" //nolint:goimports
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"strings"
	"time"
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
	noPourAndReadPool bool
	debugLogs         bool
}

type createWalletOptionFunc func(*createWalletOption)

func CreateWalletForName(t *test.SystemTest, cliConfigFilename, name string, opts ...createWalletOptionFunc) ([]string, error) {
	t.Logf("creating wallet...")
	regOpt := &createWalletOption{}
	for _, opt := range opts {
		opt(regOpt)
	}

	if regOpt.noPourAndReadPool {
		return cliutils.RunCommand(t, "./zwallet create-wallet --silent "+
			"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	}

	if regOpt.debugLogs {
		return cliutils.RunCommand(t, "./zwallet create-wallet "+
			"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	}

	output, err := ExecuteFaucetWithTokensForWallet(t, name, cliConfigFilename, 5)
	t.Logf("faucet output: %v", output)
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
