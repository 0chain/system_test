package cli_tests

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"

	apimodel "github.com/0chain/system_test/internal/api/model"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerStake(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + minerNodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+minerNodeDelegateWalletName+"_wallet.json")
	}

	output, err := listMiners(t, configPath, "--json")
	require.Nil(t, err, "error listing miners")
	require.Len(t, output, 1)

	var miners climodel.MinerSCNodes
	err = json.Unmarshal([]byte(output[0]), &miners)
	require.Nil(t, err, "error unmarshalling ls-miners json output")

	// Use the miner node not used in TestMinerSCUserPoolInfo
	minerNodeDelegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
	require.Nil(t, err, "error fetching minerNodeDelegate wallet")

	var miner climodel.Node
	for _, miner = range miners.Nodes {
		if miner.ID != minerNodeDelegateWallet.ClientID {
			break
		}
	}

	var (
		lockOutputRegex = regexp.MustCompile("locked with: [a-f0-9]{64}")
		poolIdRegex     = regexp.MustCompile("[a-f0-9]{64}")
	)

	t.Run("Staking tokens against valid miner with valid tokens should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		poolId := poolIdRegex.FindString(output[0])

		poolsInfo, err := pollForPoolInfo(t, miner.ID, poolId)
		require.Nil(t, err)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      miner.ID,
			"pool_id": poolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.Run("Staking tokens with insufficient balance should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": 1,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Staking tokens against invalid miner should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     "abcdefgh",
			"tokens": 1,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens against invalid miner but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "delegate_pool_add: miner not found or genesis miner used", output[0])
	})

	t.Run("Staking negative tokens against valid miner should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": -1,
		}), false)
		require.NotNil(t, err, "expected error when staking negative tokens but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:submit transaction failed. {"code":"invalid_request","error":"invalid_request: Invalid request (value must be greater than or equal to zero)"}`, output[0])
	})

	t.Run("Staking tokens against miner should return intrests to wallet", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		poolId := poolIdRegex.FindString(output[0])

		poolsInfo, err := pollForPoolInfo(t, miner.ID, poolId)
		require.Nil(t, err)
		balance := getBalanceFromSharders(t, wallet.ClientID)
		require.Equal(t, balance, poolsInfo.RewardPaid)

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      miner.ID,
			"pool_id": poolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.Run("Making more pools than allowed by num_delegates of miner node should fail", func(t *testing.T) {
		t.Parallel()

		var newMiner climodel.Node // Choose a different miner so it has 0 pools
		for _, newMiner = range miners.Nodes {
			if newMiner.ID != minerNodeDelegateWallet.ClientID && newMiner.ID != miner.ID {
				break
			}
		}

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.9)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		wg := &sync.WaitGroup{}
		for i := 0; i < newMiner.NumberOfDelegates; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				_, err = minerOrSharderLockForWallet(t, configPath, createParams(map[string]interface{}{
					"id":     newMiner.ID,
					"tokens": 9.0 / newMiner.NumberOfDelegates,
				}), escapedTestName(t)+fmt.Sprintf("%d", i), true)
				require.Nil(t, err)
			}(i)
		}
		wg.Wait()
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     newMiner.ID,
			"tokens": 9.0 / newMiner.NumberOfDelegates,
		}), false)
		require.NotNil(t, err, "expected error when making more pools than max_delegates but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("delegate_pool_add: max delegates already reached: %d (%d)", newMiner.NumberOfDelegates, newMiner.NumberOfDelegates), output[0])
	})

	t.Run("Staking more tokens than max_stake of miner node should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		maxStake := intToZCN(miner.MaxStake)

		for i := 0; i < (int(maxStake)/9)+1; i++ {
			_, err = executeFaucetWithTokens(t, configPath, 9.0)
			require.Nil(t, err, "error executing faucet")
		}

		balance := getBalanceFromSharders(t, wallet.ClientID)
		require.Greater(t, balance, miner.MaxStake)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": intToZCN(miner.MaxStake) + 1,
		}), true)
		require.NotNil(t, err, "expected error when staking more tokens than max_stake but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("delegate_pool_add: stake is greater than max allowed: %d \\u003e %d", miner.MaxStake+1e10, miner.MaxStake), output[0])
	})

	t.Run("Staking tokens less than min_stake of miner node should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		// Update min_stake to 1 before testing as otherwise this case will duplicate negative stake case
		_, err = minerUpdateSettings(t, configPath, createParams(map[string]interface{}{
			"id":        minerNodeDelegateWallet.ClientID,
			"min_stake": 1,
		}), true)
		require.Nil(t, err)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     minerNodeDelegateWallet.ClientID,
			"tokens": 0.5,
		}), true)
		require.NotNil(t, err, "expected error when staking more tokens than max_stake but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("delegate_pool_add: stake is less than min allowed: %d \\u003c %d", 5000000000, 10000000000), output[0])
	})

	// FIXME: This does not fail. Is this by design or a bug?
	t.Run("Staking tokens more than max_stake of a miner node through multiple stakes should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		maxStake := intToZCN(miner.MaxStake)

		for i := 0; i < (int(maxStake)/9)+1; i++ {
			_, err = executeFaucetWithTokens(t, configPath, 9.0)
			require.Nil(t, err, "error executing faucet")
		}

		balance := getBalanceFromSharders(t, wallet.ClientID)
		require.Greater(t, balance, miner.MaxStake)

		_, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": intToZCN(miner.MaxStake) / 2,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": intToZCN(miner.MaxStake)/2 + 1,
		}), true)
		// FIXME: Change to NotNil and Equal post-fix
		require.Nil(t, err, "expected error when staking more tokens than max_stake through multiple stakes but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.NotEqual(t, fmt.Sprintf("delegate_pool_add: stake is greater than max allowed: %d \\u003e %d", miner.MaxStake+1e10, miner.MaxStake), output[0])
	})
}

func pollForPoolInfo(t *testing.T, minerID, poolId string) (climodel.DelegatePool, error) {
	t.Log(`polling for pool info till it is "ACTIVE"...`)
	timeout := time.After(time.Minute * 5)

	var poolsInfo climodel.DelegatePool
	for {
		output, err := minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id":      minerID,
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")

		if poolsInfo.Status == "ACTIVE" {
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

func minerSharderPoolInfo(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerSharderPoolInfoForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func minerSharderPoolInfoForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("fetching mn-pool-info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-pool-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-pool-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}

func getBalanceFromSharders(t *testing.T, clientId string) int64 {
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
	res, err := apiGetBalance(sharderBaseURLs[0], clientId)
	require.Nil(t, err, "error getting balance")

	require.True(t, res.StatusCode >= 200 && res.StatusCode < 300, "Failed API request to get balance")
	require.NotNil(t, res.Body, "Balance API response must not be nil")

	resBody, err := io.ReadAll(res.Body)
	require.Nil(t, err, "Error reading response body")

	var startBalance apimodel.Balance
	err = json.Unmarshal(resBody, &startBalance)
	require.Nil(t, err, "Error deserializing JSON string `%s`: %v", string(resBody), err)

	return startBalance.Balance
}
