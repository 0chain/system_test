package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
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
			t.Errorf("miner node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
		}

		createWallet(t)

		createWalletForName(sharder01NodeDelegateWalletName)

		sharders := getShardersListForWallet(t, sharder01NodeDelegateWalletName)
		if len(sharders) == 0 {
			testSetup.Skip("No sharders in MagicBlock (DKG/VC deadlock) — skipping TestSharderStake")
			return
		}

		sharderNodeDelegateWallet, err := getWalletForName(t, configPath, sharder01NodeDelegateWalletName)
		require.Nil(t, err, "error fetching sharderNodeDelegate wallet")

		// Raise num_delegates for all sharders to prevent "max_delegates reached" failures
		// from accumulated pools across test runs. Use --sharder flag for sharder settings.
		for _, s := range sharders {
			out, err := minerSharderUpdateSettings(t, configPath, sharder01NodeDelegateWalletName, createParams(map[string]interface{}{
				"id":            s.ID,
				"num_delegates": 200,
				"sharder":       "",
			}), true)
			if err != nil {
				t.Logf("WARN: failed to raise num_delegates for sharder %s: %v — %s", s.ID[:12], err, strings.Join(out, " "))
			}
		}

		// Pick a sharder that has available delegate slots (not full).
		// Query each sharder's node info to check pool capacity.
		found := false
		for _, s := range sharders {
			if s.ID == sharderNodeDelegateWallet.ClientID {
				continue
			}
			// Check if this sharder has room for new delegates
			output, err := getNode(t, configPath, s.ID)
			if err == nil && len(output) == 1 {
				var nodeInfo climodel.Node
				if json.Unmarshal([]byte(output[0]), &nodeInfo) == nil {
					poolCount := len(nodeInfo.StakePool.Pools)
					maxDelegates := nodeInfo.StakePool.Settings.MaxNumDelegates
					if maxDelegates == 0 || poolCount < maxDelegates {
						sharder = s
						found = true
						t.Logf("Selected sharder %s with %d/%d delegates", s.ID, poolCount, maxDelegates)
						break
					}
					t.Logf("Sharder %s is full: %d/%d delegates", s.ID, poolCount, maxDelegates)
					continue // skip full sharder, don't use as fallback
				}
			}
			// Fallback: pick this sharder if we couldn't query node info
			if !found {
				sharder = s
				found = true
			}
		}
		if !found {
			t.Skip("No sharder with available delegate slots found (all sharders at max_delegates) — infrastructure: sharders registered with low num_delegates, pools exhausted from prior runs")
		}
	})

	var (
		lockOutputRegex = regexp.MustCompile("locked with: [a-f0-9]{64}")
	)

	t.RunSequentiallyWithTimeout("Staking tokens against valid sharder with valid tokens should work, unlocking should work", 80*time.Second, func(t *test.SystemTest) {
		createWallet(t)

		// Pre-unlock any existing stake from previous runs to avoid balance accumulation.
		// Use retry=false to avoid consuming nonces on "no such delegate pool" failures —
		// each failed retry increments the client nonce, desynchronizing subsequent transactions.
		_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
		}), false)
		cliutils.Wait(t, 5*time.Second)

		// Record balance before locking (pre-unlock may not clear a stale pool immediately)
		var balanceBeforeLock int64
		if preOutput, preErr := minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true); preErr == nil && len(preOutput) == 1 {
			var prePool climodel.DelegatePool
			if json.Unmarshal([]byte(preOutput[0]), &prePool) == nil {
				balanceBeforeLock = prePool.Balance
			}
		}

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     1,
		}), true)
		if err != nil && strings.Contains(strings.Join(output, "\n"), "max_delegates reached") {
			t.Fatalf("sharder delegate pools full (max_delegates reached) — run deploy_local.sh chain to reset num_delegates")
		}
		require.Nil(t, err, "error locking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		poolsInfo, err := pollForPoolInfo(t, sharder.ID)
		require.Nil(t, err)
		// Balance should increase by exactly 1 ZCN from what it was before locking
		expectedBalance := balanceBeforeLock + int64(1e10)
		require.Equal(t, expectedBalance, poolsInfo.Balance, "pool balance should increase by 1 ZCN")

		// unlock should work
		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
		}), true)
		require.Nil(t, err, "error unlocking tokens against a node")
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked", output[0])

		// Wait for the unlock transaction to be committed (pool removal takes 1-2 rounds)
		cliutils.Wait(t, 10*time.Second)

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true)
		require.NotNil(t, err, "expected error when requesting unlocked pool but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `resource_not_found: can't find pool stats`, output[0])
	})

	t.RunSequentiallyWithTimeout("Multiple stakes against a sharder should not create multiple pools", 3*time.Minute, func(t *test.SystemTest) {
		createWallet(t)
		faucetFundWalletOrSkip(t, escapedTestName(t), 9.0)

		// Pre-unlock any existing stake from previous runs to avoid balance accumulation.
		// Use retry=false to avoid consuming nonces on "no such delegate pool" failures —
		// each failed retry increments the client nonce, desynchronizing subsequent transactions.
		_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
		}), false)
		cliutils.Wait(t, 5*time.Second)

		// Ensure cleanup: unstake after test regardless of outcome
		t.Cleanup(func() {
			_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
				"sharder_id": sharder.ID,
			}), true)
		})

		// Record actual balance before staking (pre-unlock may not clear stale pools immediately)
		var balanceBefore int64
		if preOutput, preErr := minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true); preErr == nil && len(preOutput) == 1 {
			var prePool climodel.DelegatePool
			if json.Unmarshal([]byte(preOutput[0]), &prePool) == nil {
				balanceBefore = prePool.Balance
			}
		}

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     2,
		}), true)
		if err != nil && strings.Contains(strings.Join(output, "\n"), "max_delegates reached") {
			t.Fatalf("sharder delegate pools full (max_delegates reached) — run deploy_local.sh chain to reset num_delegates")
		}
		require.NoError(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		// Wait for first stake transaction to be confirmed before second stake
		cliutils.Wait(t, 15*time.Second)

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
		require.Len(t, poolsInfo.Pools[sharder.ID], 1, "multiple stakes should merge into single pool")

		// Balance should have increased by 4 ZCN (2+2) from before
		expectedBalance := balanceBefore + int64(4e10)
		require.Equal(t, expectedBalance, poolsInfo.Pools[sharder.ID][0].Balance,
			"balance should increase by 4 ZCN (was %d, expected %d)", balanceBefore, expectedBalance)
	})

	t.RunSequentially("Staking tokens with insufficient balance should fail", func(t *test.SystemTest) {
		createWallet(t)

		// Try to get 1 ZCN from faucet, but handle gracefully if faucet is empty
		faucetOutput, err := executeFaucetWithTokens(t, configPath, 1)
		if err != nil {
			// Check if the error is due to empty faucet
			outputStr := strings.Join(faucetOutput, "\n")
			if strings.Contains(outputStr, "faucet has no tokens") {
				// Since wallets are pre-funded with 1000 ZCN, we can't test insufficient balance
				// without first draining the wallet, which is complex. Skip this test.
				t.Errorf("Faucet is empty and wallet is pre-funded with 1000 ZCN, cannot test insufficient balance scenario")
				return
			}
			// If it's a different error, fail the test
			require.NoError(t, err, "Unexpected error from faucet: %v, Output: %s", err, outputStr)
		}

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     100000,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output", strings.Join(output, "\n"))
		combined := strings.Join(output, "\n")
		require.True(t, strings.Contains(combined, "lock amount is greater than balance") ||
			strings.Contains(combined, "too large stake to lock") ||
			strings.Contains(combined, "insufficient balance to pay fee"),
			"expected error about insufficient balance or stake limit, got: %s", combined)
	})

	t.RunSequentially("Staking negative tokens against valid sharder should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     -1,
		}), false)
		require.NotNil(t, err, "expected error when staking negative tokens but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `invalid token amount: negative`, output[0])
	})

	t.RunSequentiallyWithTimeout("Staking tokens against sharder should return interest to wallet", 2*time.Minute, func(t *test.SystemTest) {
		createWallet(t)

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err := getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ ZCN \(\d*\.?\d+ USD\)$`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?(ZCN|SAS)`).FindString(output[0])
		initialBalance, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing balance")

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
			"tokens":     1,
		}), true)
		if err != nil && strings.Contains(strings.Join(output, "\n"), "max_delegates reached") {
			t.Fatalf("sharder delegate pools full (max_delegates reached) — run deploy_local.sh chain to reset num_delegates")
		}
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
		createWallet(t)
		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err := minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder.ID,
		}), false)
		require.NotNil(t, err, "expected error when using invalid node id")
		require.Len(t, output, 1)
		require.True(t,
			output[0] == "stake_pool_unlock_failed: no such delegate pool: "+wallet.ClientID ||
				strings.Contains(output[0], "invalid transaction nonce"),
			"expected 'no such delegate pool' or nonce error, got: %s", output[0])
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
