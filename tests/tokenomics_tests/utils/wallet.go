package utils

import (
	"encoding/json"
	"fmt"
	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"io"
	"os"
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

func registerWalletAndLockReadTokens(t *test.SystemTest, cliConfigFilename string) error {
	_, err := RegisterWalletForName(t, cliConfigFilename, EscapedTestName(t))
	if err != nil {
		return err
	}
	var tokens float64 = 2
	_, err = ExecuteFaucetWithTokens(t, cliConfigFilename, tokens)
	if err != nil {
		return err
	}

	// Lock half the tokens for read pool
	readPoolParams := CreateParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	_, err = readPoolLock(t, cliConfigFilename, readPoolParams, true)

	return err
}

func RegisterWallet(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return RegisterWalletForName(t, cliConfigFilename, EscapedTestName(t))
}

func RegisterWalletForName(t *test.SystemTest, cliConfigFilename, name string) ([]string, error) {
	t.Logf("Registering wallet...")
	return cliutils.RunCommand(t, "./zbox register --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func registerWalletForNameAndLockReadTokens(t *test.SystemTest, cliConfigFilename, name string) {
	var tokens = 2.0
	registerWalletWithTokens(t, cliConfigFilename, name, tokens)
	readPoolParams := CreateParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	_, err := readPoolLockWithWallet(t, name, cliConfigFilename, readPoolParams, true)
	require.NoErrorf(t, err, "error occurred when locking read pool for %s", name)
}

func registerWalletWithTokens(t *test.SystemTest, cliConfigFilename, name string, tokens float64) {
	_, err := RegisterWalletForName(t, cliConfigFilename, name)
	require.NoErrorf(t, err, "register wallet %s", name)
	_, err = ExecuteFaucetWithTokensForWallet(t, name, cliConfigFilename, tokens)
	require.NoErrorf(t, err, "get tokens for wallet %s", name)
}

func getBalance(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	return getBalanceForWallet(t, cliConfigFilename, EscapedTestName(t))
}

func ensureZeroBalance(t *test.SystemTest, output []string, err error) {
	if err != nil {
		require.Len(t, output, 1)
		require.Equal(t, "Failed to get balance:", output[0])
		return
	}
	require.Equal(t, "Balance: 0 SAS (0.00 USD)", output[0])
}

func getBalanceForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func GetWallet(t *test.SystemTest, cliConfigFilename string) (*climodel.Wallet, error) {
	return GetWalletForName(t, cliConfigFilename, EscapedTestName(t))
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

func verifyTransaction(t *test.SystemTest, cliConfigFilename, txn string) ([]string, error) {
	t.Logf("Verifying transaction...")
	return cliutils.RunCommand(t, "./zwallet verify --silent --wallet "+EscapedTestName(t)+""+
		"_wallet.json"+" --hash "+txn+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func SetupWalletWithCustomTokens(t *test.SystemTest, configPath string, tokens float64) []string {
	output, err := RegisterWallet(t, configPath)
	require.Nil(t, err, strings.Join(output, "\n"))

	ExecuteFaucetWithTokens(t, configPath, tokens)
	require.Nil(t, err, strings.Join(output, "\n"))

	return output
}

func stakeTokens(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Staking tokens...")
	cmd := fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
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

func unstakeTokens(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Unlocking tokens from stake pool...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, EscapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func UnstakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string) ([]string, error) {
	t.Log("Unlocking tokens from stake pool...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second*2)
}

func ReadWalletFile(t *test.SystemTest, file string) *climodel.WalletFile {
	wallet := &climodel.WalletFile{}

	f, err := os.Open(file)
	require.Nil(t, err, "wallet file %s not found", file)

	ownerWalletBytes, err := io.ReadAll(f)
	require.Nil(t, err, "error reading wallet file %s", file)

	err = json.Unmarshal(ownerWalletBytes, wallet)
	require.Nil(t, err, "error marshaling wallet content")

	return wallet
}
