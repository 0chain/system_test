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
	t.Run("parallel", func(t *testing.T) {
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

			require.Len(t, output, 1)
			require.Equal(t, "Transaction verification success", output[0])

			output, err = getBalance(t, configPath)
			require.Nil(t, err, "An error occurred retrieving wallet balance", strings.Join(output, "\n"))

			require.Len(t, output, 1)
			require.Regexp(t, regexp.MustCompile(`Balance: 1.000 ZCN \([0-9.]+ USD\)$`), output[0])
		})
	})
}

func registerWallet(t *testing.T, cliConfigFilename string) ([]string, error) {
	t.Logf("Registering wallet...")
	return registerWalletForName(cliConfigFilename, escapedTestName(t))
}

func registerWalletForName(cliConfigFilename, name string) ([]string, error) {
	return cliutils.RunCommand("./zbox register --silent " +
		"--wallet " + name + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func getBalance(t *testing.T, cliConfigFilename string) ([]string, error) {
	time.Sleep(5 * time.Second)
	return getBalanceForWallet(cliConfigFilename, escapedTestName(t))
}

func getBalanceForWallet(cliConfigFilename, wallet string) ([]string, error) {
	return cliutils.RunCommand("./zwallet getbalance --silent " +
		"--wallet " + wallet + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
}

func getWallet(t *testing.T, cliConfigFilename string) (*climodel.Wallet, error) {
	return getWalletForName(t, cliConfigFilename, escapedTestName(t))
}

func getWalletForName(t *testing.T, cliConfigFilename, name string) (*climodel.Wallet, error) {
	t.Logf("Getting wallet...")
	output, err := cliutils.RunCommand("./zbox getwallet --json --silent " +
		"--wallet " + name + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)

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
	return cliutils.RunCommand("./zwallet verify --silent --wallet " + escapedTestName(t) + "" +
		"_wallet.json" + " --hash " + txn + " --configDir ./config --config " + cliConfigFilename)
}

func escapedTestName(t *testing.T) string {
	replacer := strings.NewReplacer("/", "-", "\"", "-", ":", "-",
		"<", "LESS_THAN", ">", "GREATER_THAN", "|", "-", "*", "-", "?", "-")
	return replacer.Replace(t.Name())
}
