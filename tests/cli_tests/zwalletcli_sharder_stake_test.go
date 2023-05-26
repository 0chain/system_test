package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestSharderStake(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Staking tokens against valid sharder with valid tokens should work, unlocking should work")

	var sharder climodel.Sharder
	t.TestSetup("get sharders", func() {
		if _, err := os.Stat("./config/" + sharder01NodeDelegateWalletName + "_wallet.json"); err != nil {
			t.Skipf("miner node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
		}

		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = createWalletForName(t, configPath, sharder01NodeDelegateWalletName)
		require.Nil(t, err, "Failed to create wallet", strings.Join(output, "\n"))

		sharders := getShardersListForWallet(t, sharder01NodeDelegateWalletName)

		sharderNodeDelegateWallet, err := getWalletForName(t, configPath, sharder01NodeDelegateWalletName)
		require.Nil(t, err, "error fetching sharderNodeDelegate wallet")

		for i, s := range sharders {
			if s.ID != sharderNodeDelegateWallet.ClientID {
				sharder = sharders[i]
				break
			}
		}
	})

	var (
		lockOutputRegex = regexp.MustCompile("locked with: [a-f0-9]{64}")
	)

	t.RunSequentiallyWithTimeout("Staking tokens against valid sharder with valid tokens should work, unlocking should work", 80*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     1,
		}), true)
		require.Nil(t, err, "error locking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		poolsInfo, err := pollForPoolInfo(t, sharder.ID)
		require.Nil(t, err)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Balance))

		// unlock should work
		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
		}), true)
		require.Nil(t, err, "error unlocking tokens against a node")
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked", output[0])

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true)
		require.NotNil(t, err, "expected error when requesting unlocked pool but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `resource_not_found: can't find pool stats`, output[0])
	})

	t.RunSequentiallyWithTimeout("Multiple stakes against a sharder should not create multiple pools", 80*time.Second, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     2,
		}), true)
		require.NoError(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     2,
		}), true)

		require.NoError(t, err, "error staking tokens against node: %s", output)
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.NoError(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools[sharder.ID], 1)

		require.Equal(t, poolsInfo.Pools[sharder.ID][0].Balance, int64(4e10))
	})

	t.RunSequentially("Staking tokens with insufficient balance should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     6,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `stake_pool_lock_failed: stake pool digging error: lock amount is greater than balance`, output[0])
	})

	t.RunSequentially("Staking negative tokens against valid sharder should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     -1,
		}), false)
		require.NotNil(t, err, "expected error when staking negative tokens but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `invalid token amount: negative`, output[0])
	})

	t.RunSequentiallyWithTimeout("Staking tokens against sharder should return interest to wallet", 2*time.Minute, func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		initialBalance := 1.0
		output, err = executeFaucetWithTokens(t, configPath, initialBalance)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     1,
		}), true)

		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		cliutils.Wait(t, time.Second*15)
		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}

		balance := getBalanceFromSharders(t, wallet.ClientID)
		require.Greater(t, balance, int64(initialBalance))
	})

	t.RunSequentially("Unlock tokens with invalid pool id should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "error creating wallet", strings.Join(output, "\n"))
		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
		}), false)
		require.NotNil(t, err, "expected error when using invalid node id")
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_unlock_failed: no such delegate pool: "+wallet.ClientID, output[0])
	})
}

// waitForRoundsGT waits for at least r rounds passed
func waitForRoundsGT(t *test.SystemTest, r int) error {
	var (
		lfb      = getLatestFinalizedBlock(t)
		endRound = lfb.Round + int64(r)
		checkTk  = time.NewTicker(5 * time.Second)
	)

	for {
		select {
		case <-time.After(3 * time.Minute):
			return fmt.Errorf("wait timeout")
		case <-checkTk.C:
			lfb = getLatestFinalizedBlock(t)
			if lfb.Round > endRound {
				return nil
			}
		}
	}
}

func waitForStakePoolActive(t *test.SystemTest) {
	vs := getMinerSCConfiguration(t)
	round, ok := vs["reward_round_frequency"]
	require.True(t, ok, "could not get reward_round_frequency from minersc config")

	err := waitForRoundsGT(t, int(round))
	require.NoError(t, err)
}
