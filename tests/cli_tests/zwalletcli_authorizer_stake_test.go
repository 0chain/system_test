package cli_tests

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestAuthorizerStake(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Staking tokens against valid authorizer with valid tokens should work")

	var authorizer climodel.Node
	var authorizers climodel.AutorizerNodes
	t.TestSetup("Get authorizer details", func() {
		output, err := listAuthorizer(t, configPath, "--json")
		require.NoError(t, err, "error listing authorizers")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &authorizers)
		require.Nil(t, err, "error unmarshalling bridge-list-auth json output")

		for _, authorizer = range authorizers.Nodes {
			if authorizer.ID == authorizer01ID {
				break
			}
		}
	})

	t.Parallel()

	t.RunWithTimeout("Staking tokens against valid authorizer with valid tokens should work", 5*time.Minute, func(t *test.SystemTest) { // todo: slow
		createWallet(t)

		// Lock should work
		output, err := authorizerLock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": authorizer.ID,
			"tokens":        2.0,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		poolsInfo, err := pollForPoolInfo(t, authorizer.ID)
		require.Nil(t, err)
		require.Equal(t, float64(2.0), intToZCN(poolsInfo.Balance))

		// Unlock should work
		output, err = authorizerUnlock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": authorizer.ID,
		}), true)
		require.Nil(t, err, "error unlocking tokens against a node")
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked", output[0])

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": authorizer.ID,
		}), true)

		require.NotNil(t, err, "expected error when requesting unlocked pool but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `resource_not_found: can't find pool stats`, output[0])
	})

	t.Run("Staking tokens with insufficient balance should fail", func(t *test.SystemTest) {
		_, err := executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet")

		output, err := authorizerLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": authorizer.ID,
			"tokens":   10,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: stake pool digging error: lock amount is greater than balance", output[0])
	})

	t.Run("Staking tokens against invalid node id should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := authorizerLock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": "abcdefgh",
			"tokens":        1,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens against invalid miner but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: can't get stake pool: get_stake_pool: miner not found or genesis miner used", output[0])
	})

	t.Run("Staking negative tokens against valid authorizer should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := authorizerLock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": authorizer.ID,
			"tokens":        -1,
		}), false)
		require.NotNil(t, err, "expected error when staking negative tokens but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `invalid token amount: negative`, output[0])
	})

	t.Run("Staking 0 tokens against authorizer should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": authorizer01ID,
			"tokens":        0,
		}), false)
		require.NotNil(t, err, "expected error when staking more tokens than max_stake but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: no stake to lock: 0", output[0])
	})

	t.Run("Unlock tokens with invalid node id should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := authorizerLock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": authorizer.ID,
			"tokens":        2,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		output, err = authorizerUnlock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": "abcdefgh",
		}), false)
		require.NotNil(t, err, "expected error when using invalid node id")
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_unlock_failed: can't get related stake pool: get_stake_pool: miner not found or genesis miner used", output[0])

		// teardown
		_, err = authorizerUnlock(t, configPath, createParams(map[string]interface{}{
			"authorizer_id": authorizer.ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})
}

func listAuthorizer(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	cmd := fmt.Sprintf(
		"./zwallet bridge-list-auth --silent "+
			"--configDir ./config --config %s",
		configPath,
	)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func authorizerLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return authorizerLockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func authorizerLockForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("locking tokens against authorizers...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}

func authorizerUnlock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return authorizerUnlockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func authorizerUnlockForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("unlocking tokens from authorizer pool...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
