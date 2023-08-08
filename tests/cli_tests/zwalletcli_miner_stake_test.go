package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

var lockOutputRegex = regexp.MustCompile("locked with: [a-f0-9]{64}")

func TestMinerStake(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Staking tokens against valid miner with valid tokens should work")

	var miner climodel.Node
	var miners climodel.MinerSCNodes
	t.TestSetup("Get miner details", func() {
		if _, err := os.Stat("./config/" + miner01NodeDelegateWalletName + "_wallet.json"); err != nil {
			t.Skipf("miner node owner wallet located at %s is missing", "./config/"+miner01NodeDelegateWalletName+"_wallet.json")
		}

		output, err := listMiners(t, configPath, "--json")
		require.NoError(t, err, "error listing miners")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &miners)
		require.Nil(t, err, "error unmarshalling ls-miners json output")

		for _, miner = range miners.Nodes {
			if miner.ID == miner01ID {
				break
			}
		}
	})

	t.Parallel()

	t.RunWithTimeout("Staking tokens against valid miner with valid tokens should work", 5*time.Minute, func(t *test.SystemTest) { // todo: slow
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   2.0,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		poolsInfo, err := pollForPoolInfo(t, miner.ID)
		require.Nil(t, err)
		require.Equal(t, float64(2.0), intToZCN(poolsInfo.Balance))

		// Unlock should work
		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
		}), true)
		require.Nil(t, err, "error unlocking tokens against a node")
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked", output[0])

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)

		require.NotNil(t, err, "expected error when requesting unlocked pool but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `resource_not_found: can't find pool stats`, output[0])
	})

	t.RunWithTimeout("Multiple stakes against a miner should add balance to client's stake pool", 5*time.Minute, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 6.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		var poolsInfoBefore climodel.MinerSCUserPoolsInfo
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &poolsInfoBefore)
		require.Nil(t, err, "error unmarshalling Miner SC user pool info")

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   2,
		}), true)
		require.Nil(t, err,
			"error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])

		// wait for pool to be active from pending status, usually need to wait for 50 rounds
		waitForStakePoolActive(t)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   2,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])

		var poolsInfo climodel.MinerSCUserPoolsInfo
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.NoError(t, err)
		require.Len(t, poolsInfo.Pools[miner.ID], 1)
	})

	t.Run("Staking tokens with insufficient balance should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   10,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: stake pool digging error: lock amount is greater than balance", output[0])
	})

	// this case covers both invalid miner and sharder id, so is not repeated in zwalletcli_sharder_stake_test.go
	t.Run("Staking tokens against invalid node id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": "abcdefgh",
			"tokens":   1,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens against invalid miner but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: can't get stake pool: get_stake_pool: miner not found or genesis miner used", output[0])
	})

	t.Run("Staking negative tokens against valid miner should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   -1,
		}), false)
		require.NotNil(t, err, "expected error when staking negative tokens but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `invalid token amount: negative`, output[0])
	})

	// todo rewards not transferred to wallet until a collect reward transaction
	t.RunSequentially("Staking tokens against miner should return interest to wallet", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   1,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		poolsInfo, err := pollForPoolInfo(t, miner.ID)
		require.Nil(t, err)
		balance := getBalanceFromSharders(t, wallet.ClientID)
		require.GreaterOrEqual(t, balance, poolsInfo.Reward)

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.Run("Making more pools than allowed by max_delegates in minersc should fail", func(t *test.SystemTest) {
		var newMiner climodel.Node // Choose a different miner so it has 0 pools
		for _, newMiner = range miners.Nodes {
			if newMiner.ID == miner02ID {
				break
			}
		}

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.9)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfg, _ := keyValuePairStringToMap(output)
		maxDelegates, err := strconv.ParseInt(cfg["max_delegates"], 10, 0)
		require.Nil(t, err)

		wg := &sync.WaitGroup{}
		for i := 0; i < int(maxDelegates); i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				walletName := escapedTestName(t) + fmt.Sprintf("%d", i)
				_, err := executeFaucetWithTokensForWallet(t, walletName, configPath, 1.0)
				require.Nil(t, err)

				output, err = minerOrSharderLockForWallet(t, configPath, createParams(map[string]interface{}{
					"miner_id": newMiner.ID,
					"tokens":   0.1,
				}), walletName, true)
				require.NoError(t, err)
				require.Len(t, output, 1)
				require.Regexp(t, lockOutputRegex, output[0])
			}(i)
		}
		wg.Wait()
		require.NotEqual(t, 0, newMiner.Settings.MaxNumDelegates)
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": newMiner.ID,
			"tokens":   9.0,
		}), false)
		require.NotNil(t, err, "expected error when making more pools than max_delegates but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("stake_pool_lock_failed: max_delegates reached: %d, no more stake pools allowed", maxDelegates), output[0])
	})

	t.Run("Staking 0 tokens against miner should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner01ID,
			"tokens":   0,
		}), false)
		require.NotNil(t, err, "expected error when staking more tokens than max_stake but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: no stake to lock: 0", output[0])
	})

	// this case covers both invalid miner and sharder id, so is not repeated in zwalletcli_sharder_stake_test.go
	t.Run("Unlock tokens with invalid node id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   2,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": "abcdefgh",
		}), false)
		require.NotNil(t, err, "expected error when using invalid node id")
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_unlock_failed: can't get related stake pool: get_stake_pool: miner not found or genesis miner used", output[0])

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})
}

func pollForPoolInfo(t *test.SystemTest, minerID string) (climodel.DelegatePool, error) {
	t.Log(`polling for pool info till it is "ACTIVE"...`)
	timeout := time.After(time.Minute * 5)

	var poolsInfo climodel.DelegatePool
	for {
		output, err := minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": minerID,
		}), true)
		require.Nil(t, err, "error fetching Miner Sharder pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner Sharder pools")
		require.NotEmpty(t, poolsInfo)

		if poolsInfo.Status == int(climodel.Active) {
			return poolsInfo, nil
		}
		select {
		case <-timeout:
			return climodel.DelegatePool{}, errors.New("Pool status did not change to active")
		default:
			cliutils.Wait(t, time.Second*15)
		}
	}
}

func minerSharderPoolInfo(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerSharderPoolInfoForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func minerSharderPoolInfoForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("fetching mn-pool-info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-pool-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-pool-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}

func getBalanceFromSharders(t *test.SystemTest, clientId string) int64 {
	output, err := getSharders(t, configPath)
	require.Nil(t, err, "get sharders failed", strings.Join(output, "\n"))
	require.Greater(t, len(output), 1)
	require.Equal(t, "MagicBlock Sharders", output[0])

	var sharders map[string]*climodel.Sharder
	err = json.Unmarshal([]byte(strings.Join(output[1:], "")), &sharders)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output[1:], "\n"), err)
	require.NotEmpty(t, sharders, "No sharders found: %v", strings.Join(output[1:], "\n"))

	// Get base URL for API calls.
	sharderBaseURLs := getAllSharderBaseURLs(sharders)
	res, err := apiGetBalance(t, sharderBaseURLs[0], clientId)
	require.Nil(t, err, "error getting balance")

	if res.StatusCode == 400 {
		return 0
	}
	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get balance")
	require.NotNil(t, res.Body, "Balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body")

	var startBalance climodel.Balance
	err = json.Unmarshal(resBody, &startBalance)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

	return startBalance.Balance
}

func minerOrSharderLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerOrSharderLockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func minerOrSharderLockForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("locking tokens against miner/sharder...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}

func minerOrSharderUnlock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerOrSharderUnlockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func minerOrSharderUnlockForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("unlocking tokens from miner/sharder pool...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
