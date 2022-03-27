package cli_tests

import (
	"regexp"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestVestingPoolList(t *testing.T) {
	t.Parallel()

	// get current valid vesting configs
	output, err := registerWallet(t, configPath)
	require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

	output, err = getVestingPoolSCConfig(t, configPath, true)
	require.Nil(t, err, "error fetching vesting pool config", strings.Join(output, "\n"))

	vpConfigMap := configFromKeyValuePair(output)
	validDuration := getValidDuration(t, vpConfigMap)

	// VP-LIST cases
	t.Run("Vesting pool list before and after adding pool must work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching client wallet")

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": wallet.ClientID,
		}), true)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "no vesting pools", output[0])

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		// add a vesting pool for sending 0.1 to target wallet
		output, err = vestingPoolAdd(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": wallet.ClientID,
		}), true)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "-  "+poolId, output[0])
	})

	// FIXME: Is this expected behavior that any wallet can list any wallet's vesting pools?
	t.Run("Listing pools for someone else's client-id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		foreignWalletName := "foreignWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error registering new wallet", strings.Join(output, "\n"))

		foreignWallet, err := getWalletForName(t, configPath, foreignWalletName)
		require.Nil(t, err, "error fetching wallet")

		output, err = executeFaucetWithTokensForWallet(t, foreignWalletName, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		targetWalletName := "targetWallet" + escapedTestName(t)
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering target wallet", strings.Join(output, "\n"))

		targetWallet, err := getWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error fetching destination wallet")

		output, err = vestingPoolAddForWallet(t, configPath, createParams(map[string]interface{}{
			"d":        targetWallet.ClientID + ":0.1",
			"lock":     0.1,
			"duration": validDuration,
		}), true, foreignWalletName)
		require.Nil(t, err, "error adding a new vesting pool")
		require.Len(t, output, 2)
		require.Regexp(t, regexp.MustCompile("Vesting pool added successfully:[a-z0-9]{64}:vestingpool:[a-z0-9]{64}"), output[0], "output did not match expected vesting pool pattern")
		poolId := regexp.MustCompile("[a-z0-9]{64}:vestingpool:[a-z0-9]{64}").FindString(output[0])
		require.NotEmpty(t, poolId, "expected pool ID as output to vp-add command")

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": foreignWallet.ClientID,
		}), true)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "-  "+poolId, output[0])
	})

	t.Run("Vesting pool list with invalid client id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error requesting tokens from faucet", strings.Join(output, "\n"))

		output, err = vestingPoolList(t, configPath, createParams(map[string]interface{}{
			"client_id": "abcdef123456",
		}), false)
		require.Nil(t, err, "error listing vesting pools")
		require.Len(t, output, 1)
		require.Equal(t, "no vesting pools", output[0])
	})
}

func vestingPoolList(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Log("Listing vesting pools...")
	if retry {
		return cliutils.RunCommand(t, "./zwallet vp-list "+params+
			" --silent --wallet "+escapedTestName(t)+"_wallet.json"+" --configDir ./config --config "+cliConfigFilename, 3, time.Second*5)
	} else {
		return cliutils.RunCommandWithoutRetry("./zwallet vp-list " + params +
			" --silent --wallet " + escapedTestName(t) + "_wallet.json" + " --configDir ./config --config " + cliConfigFilename)
	}
}
