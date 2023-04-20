package cli_tests

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const (
	defaultInitFaucetTokens = 5.0
)

func registerWalletAndLockReadTokens(t *test.SystemTest, cliConfigFilename string) error {
	_, err := registerWalletForName(t, cliConfigFilename, escapedTestName(t))
	if err != nil {
		return err
	}
	var tokens float64 = 2
	_, err = executeFaucetWithTokens(t, cliConfigFilename, tokens)
	if err != nil {
		return err
	}

	// Lock half the tokens for read pool
	readPoolParams := createParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	_, err = readPoolLock(t, cliConfigFilename, readPoolParams, true)

	return err
}

func registerWallet(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	return registerWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func registerWalletForName(t *test.SystemTest, cliConfigFilename, name string) ([]string, error) {
	t.Logf("Registering wallet...")

	output, err := cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet create-wallet --silent " +
		"--wallet " + name + " --configDir ./config --config " + cliConfigFilename))
	if err != nil {
		return nil, err
	}

	output, err = executeFaucetWithTokensForWallet(t, name, cliConfigFilename, defaultInitFaucetTokens)
	if err != nil {
		return nil, err
	}
	t.Logf("faucet output: %v", output)

	return []string{""}, nil
}

func registerWalletForNameAndLockReadTokens(t *test.SystemTest, cliConfigFilename, name string) {
	var tokens = 2.0
	registerWalletWithTokens(t, cliConfigFilename, name, tokens)
	readPoolParams := createParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	_, err := readPoolLockWithWallet(t, name, cliConfigFilename, readPoolParams, true)
	require.NoErrorf(t, err, "error occurred when locking read pool for %s", name)
}

func registerWalletWithTokens(t *test.SystemTest, cliConfigFilename, name string, tokens float64) {
	_, err := registerWalletForName(t, cliConfigFilename, name)
	require.NoErrorf(t, err, "register wallet %s", name)
	_, err = executeFaucetWithTokensForWallet(t, name, cliConfigFilename, tokens)
	require.NoErrorf(t, err, "get tokens for wallet %s", name)
}

func getBalance(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	return getBalanceForWallet(t, cliConfigFilename, escapedTestName(t))
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
	fmt.Println("./zwallet getbalance --silent " +
		"--wallet " + wallet + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)

	return cliutils.RunCommand(t, "./zwallet getbalance --silent "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getWallet(t *test.SystemTest, cliConfigFilename string) (*climodel.Wallet, error) {
	return getWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func getWalletForName(t *test.SystemTest, cliConfigFilename, name string) (*climodel.Wallet, error) {
	t.Logf("Getting wallet...")

	fmt.Println("name: ", name, " cliConfigFilename: ", cliConfigFilename)

	output, err := cliutils.RunCommand(t, "./zbox getwallet --json --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)

	if err != nil {
		return nil, err
	}

	fmt.Println("output: ", output)

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
	return cliutils.RunCommand(t, "./zwallet verify --silent --wallet "+escapedTestName(t)+""+
		"_wallet.json"+" --hash "+txn+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getBalanceZCN(t *test.SystemTest, cliConfigFilename string, walletName ...string) (float64, error) {
	cliutils.Wait(t, 5*time.Second)
	var (
		output []string
		err    error
	)
	if len(walletName) > 0 && walletName[0] != "" {
		output, err = getBalanceForWalletJSON(t, cliConfigFilename, walletName[0])
		if err != nil {
			return 0, err
		}
	} else {
		output, err = getBalanceForWalletJSON(t, cliConfigFilename, escapedTestName(t))
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

	return strconv.ParseFloat(balance.ZCN, 64)
}

func escapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func getBalanceForWalletJSON(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent --json "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}
