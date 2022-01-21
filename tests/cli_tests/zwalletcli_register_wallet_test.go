package cli_tests

import (
	"encoding/json"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
)

func TestRegisterWallet(t *testing.T) {
	t.Parallel()

	t.Run("Register wallet outputs expected", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)

		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))
		require.Len(t, output, 4)
		require.Equal(t, "ZCN wallet created", output[0])
		require.Equal(t, "Creating related read pool for storage smart-contract...", output[1])
		require.Equal(t, "Read pool created successfully", output[2])
		require.Equal(t, "Wallet registered", output[3])
	})

	t.Run("Get wallet outputs expected", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)

		require.Nil(t, err, "An error occurred retrieving a wallet", strings.Join(output, "\n"))
		require.NotNil(t, wallet.ClientID)
		require.NotNil(t, wallet.ClientPublicKey)
		require.NotNil(t, wallet.EncryptionPublicKey)
	})

	t.Run("Balance call fails due to zero ZCN in wallet", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		output, err = getBalance(t, configPath)

		require.NotNil(t, err, "Expected initial balance operation to fail", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Failed to get balance:", output[0])
	})

	t.Run("Balance of 1 is returned after faucet execution", func(t *testing.T) {
		t.Parallel()
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "An error occurred registering a wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

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

		output, err = getBalance(t, configPath)
		require.Nil(t, err, "An error occurred retrieving wallet balance", strings.Join(output, "\n"))

		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \([0-9.]+ USD\)$`), output[0])
	})
}

func registerWallet(t *testing.T, cliConfigFilename string) ([]string, error) {
	return registerWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func registerWalletForName(t *testing.T, cliConfigFilename, name string) ([]string, error) {
	t.Logf("Registering wallet...")
	return cliutils.RunCommand(t, "./zbox register --silent "+
		"--wallet "+name+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getBalance(t *testing.T, cliConfigFilename string) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	return getBalanceForWallet(t, cliConfigFilename, escapedTestName(t))
}

func getBalanceForWallet(t *testing.T, cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand(t, "./zwallet getbalance --silent "+
		"--wallet "+wallet+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func getWallet(t *testing.T, cliConfigFilename string) (*climodel.Wallet, error) {
	return getWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func getWalletForName(t *testing.T, cliConfigFilename, name string) (*climodel.Wallet, error) {
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

func verifyTransaction(t *testing.T, cliConfigFilename, txn string) ([]string, error) {
	t.Logf("Verifying transaction...")
	return cliutils.RunCommand(t, "./zwallet verify --silent --wallet "+escapedTestName(t)+""+
		"_wallet.json"+" --hash "+txn+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}

func escapedTestName(t *testing.T) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-", "(", "-",
		")", "-", "<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-",
		"?", "-")
	return replacer.Replace(t.Name())
}

func assertChargeableError(t *testing.T, output []string, msg string) {
	require.Len(t, output, 2, strings.Join(output, "\n"))

	split := strings.Split(output[1], ":")
	require.Len(t, split, 2, strings.Join(split, "\n"))
	output, _ = verifyTransaction(t, configPath, split[1])
	require.Len(t, output, 3, strings.Join(output, "\n"))
	confStatus := strings.Trim(strings.Split(output[1], ":")[1], " ") //TransactionStatus: 2
	require.Equal(t, "2", confStatus, strings.Join(output, "\n"))
	errString := strings.Trim(strings.Trim(strings.SplitN(output[2], ":", 2)[1], " "), "\"") //TransactionOutput: "update_settings:key max_pour_amount, unable to convert x to state.balance
	require.Equal(t, msg, errString, strings.Join(output, "\n"))
}

func assertChargeableErrorRegexp(t *testing.T, output []string, reg *regexp.Regexp) {
	require.Len(t, output, 2, strings.Join(output, "\n"))

	split := strings.Split(output[1], ":")
	require.Len(t, split, 2, strings.Join(split, "\n"))
	output, _ = verifyTransaction(t, configPath, split[1])
	require.Len(t, output, 3, strings.Join(output, "\n"))
	confStatus := strings.Trim(strings.Split(output[1], ":")[1], " ") //TransactionStatus: 2
	require.Equal(t, "2", confStatus, strings.Join(output, "\n"))
	errString := strings.Trim(strings.Trim(strings.SplitN(output[2], ":", 2)[1], " "), "\"") //TransactionOutput: "update_settings:key max_pour_amount, unable to convert x to state.balance
	require.Regexp(t, reg, errString, strings.Join(output, "\n"))
}
