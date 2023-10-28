package cli_tests

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestZCNBridgeAuthorizerRegisterAndDelete(testSetup *testing.T) { // nolint:gocyclo // team preference is to have codes all within test.
	t := test.NewSystemTest(testSetup)
	output, err := createWallet(t, configPath)
	require.NoError(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))

	t.RunSequentially("Register authorizer to DEX smartcontract", func(t *test.SystemTest) {
		output, err = scRegisterAuthorizer(t, "0xEa36456C79caD6Dd941Fe552285594C7217Fe258", true)
		require.NoError(t, err, "error trying to register authorizer to DEX sc: %s", strings.Join(output, "\n"))
		t.Log("register authorizer DEX SC successfully")
	})

	t.RunSequentially("Remove authorizer from DEX smartcontract", func(t *test.SystemTest) {
		output, err = scRemoveAuthorizer(t, "0xEa36456C79caD6Dd941Fe552285594C7217Fe258", true)
		require.NoError(t, err, strings.Join(output, "\n"))
		t.Log("remove authorizer DEX SC successfully")
	})
}

func TestZCNAuthorizerRegisterAndDelete(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	output, err := createWallet(t, configPath)
	require.NoError(t, err, "Unexpected create wallet failure", strings.Join(output, "\n"))

	w, err := getWallet(t, configPath)
	require.NoError(t, err)

	var (
		publicKey = w.ClientPublicKey
		clientID  = w.ClientID
		authURL   = "http://systemtest.network/authorizer01"
	)
	// faucet pour to sc owner wallet
	output, err = executeFaucetWithTokensForWallet(t, "wallets/sc_owner", configPath, defaultInitFaucetTokens)
	require.NoError(t, err, "Unexpected faucet execution failure", strings.Join(output, "\n"))

	t.RunSequentially("Register authorizer to zcnsc smartcontract", func(t *test.SystemTest) {
		output, err := registerAuthorizer(t, clientID, publicKey, authURL, true)
		require.NoError(t, err, "error trying to register authorizer to zcnsc: %s", strings.Join(output, "\n"))
		t.Log("register authorizer zcnsc successfully")
	})

	t.RunSequentially("Remove authorizer from zcnsc smartcontract", func(t *test.SystemTest) {
		output, err := removeAuthorizer(t, clientID, true)
		require.NoError(t, err, strings.Join(output, "\n"))
		t.Log("remove authorizer zcnsc successfully")
	})
}

func scRegisterAuthorizer(t *test.SystemTest, authAddress string, retry bool) ([]string, error) {
	t.Logf("Register authorizer to SC...")
	cmd := fmt.Sprintf(
		"./zwallet auth-sc-register "+
			"--ethereum_address=%s "+
			"--silent "+
			"--path %s "+
			"--configDir ./config "+
			"--wallet %s",
		authAddress,
		configDir,
		escapedTestName(t)+"_wallet.json",
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func scRemoveAuthorizer(t *test.SystemTest, authAddress string, retry bool) ([]string, error) {
	t.Logf("Remove authorizer to SC ...")
	cmd := fmt.Sprintf(
		"./zwallet auth-sc-delete --ethereum_address=%s "+
			"--silent "+
			"--path %s "+
			"--configDir ./config "+
			"--wallet %s",
		authAddress,
		configDir,
		escapedTestName(t)+"_wallet.json",
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func registerAuthorizer(t *test.SystemTest, clientID, publicKey, authURL string, retry bool) ([]string, error) {
	t.Log("Register authorizer to zcnsc ...")

	cmd := fmt.Sprintf(`
		./zwallet auth-register --silent
		--configDir ./config
		--path %s
		--wallet wallets/sc_owner_wallet.json
		--client_id %s
		--client_key %s
		--url %s
		--min_stake 1
		--max_stake 10
		--service_charge 0.1
		--num_delegates 5`,
		configDir, clientID, publicKey, authURL)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func removeAuthorizer(t *test.SystemTest, clientID string, retry bool) ([]string, error) {
	t.Log("Remove authorizer from zcnsc ...")

	cmd := fmt.Sprintf(`
		./zwallet bridge-auth-delete --silent
		--configDir ./config
		--wallet wallets/sc_owner_wallet.json
		--id %s`, clientID)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
