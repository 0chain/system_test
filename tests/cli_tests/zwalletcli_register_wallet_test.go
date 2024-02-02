package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func createWalletAndLockReadTokens(t *test.SystemTest, cliConfigFilename string) error {
	createWalletForName(escapedTestName(t))

	// Lock half the tokens for read pool
	readPoolParams := createParams(map[string]interface{}{
		"tokens": 2,
	})
	_, err := readPoolLock(t, cliConfigFilename, readPoolParams, true)
	if err != nil {
		return err
	}

	return nil
}

func createWallet(t *test.SystemTest) {
	createWalletForName(escapedTestName(t))
}

func createWalletForName(name string) {
	walletPath := fmt.Sprintf("./config/%s_wallet.json", name)

	// check if wallet already exists
	if _, err := os.Stat(walletPath); err == nil {
		return
	}

	walletMutex.Lock()
	wallet := wallets[walletIdx]
	walletIdx++
	walletMutex.Unlock()

	err := os.WriteFile(walletPath, wallet, 0644)
	if err != nil {
		fmt.Printf("Error writing file %s: %v\n", walletPath, err)
	} else {
		fmt.Printf("File %s written successfully.\n", walletPath)
	}
}

func createWalletForNameAndLockReadTokens(t *test.SystemTest, cliConfigFilename, name string) {
	var tokens = 2.0
	createWalletForName(name)
	readPoolParams := createParams(map[string]interface{}{
		"tokens": tokens / 2,
	})
	_, err := readPoolLockWithWallet(t, name, cliConfigFilename, readPoolParams, true)
	require.NoErrorf(t, err, "error occurred when locking read pool for %s", name)
}

func getBalance(t *test.SystemTest, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	return getBalanceForWallet(t, cliConfigFilename, escapedTestName(t))
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

	balanceFloat, err := strconv.ParseFloat(balance.ZCN, 64)
	if err != nil {
		return 0, err
	}

	// round up to 2 decimal places
	return float64(int(balanceFloat*100)) / 100, nil
}

func getBalanceForWallet(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getBalanceForWalletJSON(t *test.SystemTest, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent --json "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getWallet(t *test.SystemTest, cliConfigFilename string) (*climodel.Wallet, error) {
	return getWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func getWalletForName(t *test.SystemTest, cliConfigFilename, name string) (*climodel.Wallet, error) {
	t.Logf("Getting wallet...")
	output, err := cliutils.RunCommand(t, "./zbox getwallet --json --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)

	if err != nil {
		return nil, err
	}

	require.True(t, len(output) == 1 || len(output) == 2)

	var wallet *climodel.Wallet
	if len(output) == 1 {
		err = json.Unmarshal([]byte(output[0]), &wallet)
		if err != nil {
			t.Errorf("failed to unmarshal the result into wallet")
			return nil, err
		}
	} else {
		require.EqualValues(t, "ZCN wallet created", output[0])
		err = json.Unmarshal([]byte(output[1]), &wallet)
		if err != nil {
			t.Errorf("failed to unmarshal the result into wallet")
			return nil, err
		}
	}

	return wallet, err
}

func escapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

// executeFaucetWithTokens executes faucet command with given tokens.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeFaucetWithTokens(t *test.SystemTest, cliConfigFilename string, tokens float64) ([]string, error) {
	return executeFaucetWithTokensForWallet(t, escapedTestName(t), cliConfigFilename, tokens)
}

// executeFaucetWithTokensForWallet executes faucet command with given tokens and wallet.
// Tokens greater than or equal to 10 are considered to be 1 token by the system.
func executeFaucetWithTokensForWallet(t *test.SystemTest, wallet, cliConfigFilename string, tokens float64) ([]string, error) {
	t.Logf("Executing faucet...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet faucet --methodName "+
		"pour --tokens %f --input {} --silent --wallet %s_wallet.json --configDir ./config --config %s",
		tokens,
		wallet,
		cliConfigFilename,
	), 3, time.Second*5)
}
