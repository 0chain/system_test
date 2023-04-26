package cli_tests

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

const (
	defaultInitFaucetTokens = 5.0
)

func TestRegisterWallet(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.Parallel()

	t.Run("Register wallet outputs expected", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath, withNoFaucetPour())

		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 3, len(output))
		require.Equal(t, "Wallet registered", output[2])
	})

	t.Run("Get wallet outputs expected", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)

		require.Nil(t, err, "An error occurred retrieving a wallet", strings.Join(output, "\n"))
		require.NotNil(t, wallet.ClientID)
		require.NotNil(t, wallet.ClientPublicKey)
		require.NotNil(t, wallet.EncryptionPublicKey)
	})

	t.Run("Balance call fails due to zero ZCN in wallet", func(t *test.SystemTest) {
		_, _ = registerWallet(t, configPath, withNoFaucetPour())

		balance, err := getBalanceZCN(t, configPath)
		require.Nil(t, err)
		require.Equal(t, float64(0), balance)
	})

	t.Run("Balance of 1 is returned after faucet execution", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.Nil(t, err)

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.NoError(t, err)
		require.Len(t, output, 1)
		matcher := regexp.MustCompile("Execute faucet smart contract success with txn : {2}([a-f0-9]{64})$")
		require.Regexp(t, matcher, output[0], "Faucet execution output did not match expected")

		txnID := matcher.FindAllStringSubmatch(output[0], 1)[0][1]
		output, err = verifyTransaction(t, configPath, txnID)
		require.Nil(t, err, "Could not verify faucet transaction", strings.Join(output, "\n"))

		require.Len(t, output, 3)
		require.Equal(t, "Transaction verification success", output[0])
		require.Equal(t, "TransactionStatus: 1", output[1])
		require.Greater(t, len(output[2]), 0, output[2])

		balanceAfter, err := getBalanceZCN(t, configPath)
		require.Nil(t, err, "An error occurred retrieving wallet balance", strings.Join(output, "\n"))
		require.Equal(t, balanceBefore+1, balanceAfter)
	})
}

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

func registerWallet(t *test.SystemTest, cliConfigFilename string, opt ...registerWalletOptionFunc) ([]string, error) {
	return registerWalletForName(t, cliConfigFilename, escapedTestName(t), opt...)
}

type registerWalletOption struct {
	noPourAndReadPool bool
	debugLogs         bool
}

type registerWalletOptionFunc func(*registerWalletOption)

func withNoFaucetPour() registerWalletOptionFunc {
	return func(o *registerWalletOption) {
		o.noPourAndReadPool = true
	}
}

func registerWalletForName(t *test.SystemTest, cliConfigFilename, name string, opts ...registerWalletOptionFunc) ([]string, error) {
	t.Logf("Registering wallet...")
	regOpt := &registerWalletOption{}
	for _, opt := range opts {
		opt(regOpt)
	}

	if regOpt.noPourAndReadPool {
		return cliutils.RunCommand(t, "./zwallet register --silent "+
			"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	}

	output, err := executeFaucetWithTokensForWallet(t, name, cliConfigFilename, defaultInitFaucetTokens)
	if err != nil {
		return nil, err
	}
	t.Logf("faucet output: %v", output)

	if regOpt.debugLogs {
		return cliutils.RunCommand(t, "./zbox register "+
			"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
	}

	return cliutils.RunCommand(t, "./zbox register --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
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
	t.Logf("Verifying transaction %s", txn)
	return cliutils.RunCommand(t, "./zwallet verify --silent --wallet "+escapedTestName(t)+""+
		"_wallet.json"+" --hash "+txn+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func escapedTestName(t *test.SystemTest) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}
