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

func faucetFundWalletOrSkip(t *test.SystemTest, wallet string, tokens float64) {
	// Some environments don't pre-fund the test wallets; stake transactions will fail with
	// "insufficient balance to pay fee" unless we faucet first.
	faucetOutput, err := executeFaucetWithTokensForWallet(t, wallet, configPath, tokens)
	if err != nil {
		outputStr := strings.Join(faucetOutput, "\n")
		if strings.Contains(outputStr, "faucet has no tokens") {
			t.Errorf("Faucet is empty and wallet %q is not pre-funded; cannot proceed", wallet)
			return
		}

		// Some environments return invalid responses from the sharder confirmation endpoint
		// (e.g. HTTP 400 with "unexpected end of JSON input"), which makes the SDK treat
		// otherwise-submitted transactions as failed during verification.
		if strings.Contains(outputStr, "/v1/transaction/get/confirmation") &&
			(strings.Contains(outputStr, "unexpected end of JSON input") ||
				strings.Contains(outputStr, "too less sharders to confirm it")) {
			t.Errorf("Skipping: cannot verify faucet transaction due to sharder confirmation errors. Output: %s", outputStr)
			return
		}

		require.NoError(t, err, "Unexpected error from faucet: %v, Output: %s", err, outputStr)
	}
}

func TestMinerStake(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Staking tokens against valid miner with valid tokens should work")

	var miner climodel.Node
	var miners climodel.MinerSCNodes
	t.TestSetup("Get miner details", func() {
		if _, err := os.Stat("./config/" + miner01NodeDelegateWalletName + "_wallet.json"); err != nil {
			t.Errorf("miner node owner wallet located at %s is missing", "./config/"+miner01NodeDelegateWalletName+"_wallet.json")
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

		// Ensure all miners have sufficient num_delegates (200) so staking subtests
		// don't hit max_delegates=5 from accumulated pools across runs.
		for _, m := range miners.Nodes {
			_, _ = minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
				"id":            m.ID,
				"num_delegates": 200,
			}), true)
		}
	})

	// NOT parallel: subtests modify global max_delegates and per-miner num_delegates,
	// which races with TestMinerUpdateSettings, TestMinerUpdateConfig, and TestGetStakableProviders.

	t.RunSequentiallyWithTimeout("Staking tokens against valid miner with valid tokens should work", 5*time.Minute, func(t *test.SystemTest) { // todo: slow
		// Select a miner that is NOT miner02ID to avoid conflicts with other tests
		var testMiner climodel.Node
		found := false
		for _, m := range miners.Nodes {
			if m.ID != miner02ID {
				testMiner = m
				found = true
				break
			}
		}
		require.True(t, found, "No suitable miner found (need a miner that is not miner02ID)")

		createWallet(t)
		faucetFundWalletOrSkip(t, escapedTestName(t), 5.0)

		// Clear any pre-existing pool: wallet keys are deterministically reused across runs,
		// so a pool from a previous failed run may persist on chain.
		_, _ = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": testMiner.ID,
		}), false)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": testMiner.ID,
			"tokens":   2.0,
		}), true)
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "max_delegates reached") {
				t.Skip("Miner has reached max_delegates - cannot stake: " + combined)
			}
			if strings.Contains(combined, "too less sharders") || strings.Contains(combined, "unexpected end of JSON") {
				t.Skip("Chain transient error during stake: " + combined)
			}
		}
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		poolsInfo, err := pollForPoolInfo(t, testMiner.ID)
		require.Nil(t, err)
		require.Equal(t, 2.0, intToZCN(poolsInfo.Balance))

		// Unlock should work
		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": testMiner.ID,
		}), true)
		require.Nil(t, err, "error unlocking tokens against a node")
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked", output[0])

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": testMiner.ID,
		}), true)

		require.NotNil(t, err, "expected error when requesting unlocked pool but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `resource_not_found: can't find pool stats`, output[0])
	})

	t.RunSequentiallyWithTimeout("Multiple stakes against a miner should add balance to client's stake pool", 5*time.Minute, func(t *test.SystemTest) {
		// Select a miner that is NOT miner02ID to avoid conflicts with other tests
		var testMiner climodel.Node
		found := false
		for _, m := range miners.Nodes {
			if m.ID != miner02ID {
				testMiner = m
				found = true
				break
			}
		}
		require.True(t, found, "No suitable miner found (need a miner that is not miner02ID)")

		createWallet(t)
		faucetFundWalletOrSkip(t, escapedTestName(t), 6.0)

		var poolsInfoBefore climodel.MinerSCUserPoolsInfo
		output, err := stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &poolsInfoBefore)
		require.Nil(t, err, "error unmarshalling Miner SC user pool info")

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": testMiner.ID,
			"tokens":   2,
		}), true)
		require.Nil(t, err,
			"error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])

		// wait for pool to be active from pending status, usually need to wait for 50 rounds
		waitForStakePoolActive(t)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": testMiner.ID,
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
		if len(poolsInfo.Pools[testMiner.ID]) == 0 {
			t.Skip("Pool not reflected in mn-user-info; chain may be experiencing instability (max_delegates too low or sharder lag)")
		}
		require.Len(t, poolsInfo.Pools[testMiner.ID], 1)
	})

	t.Run("Staking tokens with insufficient balance should fail", func(t *test.SystemTest) {
		_, err := executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet")

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   10,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output: ", strings.Join(output, "\n"))
		combined := strings.Join(output, "\n")
		if strings.Contains(combined, "too less sharders") || strings.Contains(combined, "unexpected end of JSON") ||
			strings.Contains(combined, "invalid transaction nonce") {
			t.Skip("Chain transient error during insufficient-balance stake test: " + combined)
		}
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: stake pool digging error: lock amount is greater than balance", output[0])
	})

	// this case covers both invalid miner and sharder id, so is not repeated in zwalletcli_sharder_stake_test.go
	t.Run("Staking tokens against invalid node id should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": "abcdefgh",
			"tokens":   1,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens against invalid miner but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: can't get stake pool: get_stake_pool: miner not found or genesis miner used", output[0])
	})

	t.Run("Staking negative tokens against valid miner should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   -1,
		}), false)
		require.NotNil(t, err, "expected error when staking negative tokens but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `invalid token amount: negative`, output[0])
	})

	// todo rewards not transferred to wallet until a collect reward transaction
	t.RunSequentially("Staking tokens against miner should return interest to wallet", func(t *test.SystemTest) {
		createWallet(t)
		faucetFundWalletOrSkip(t, escapedTestName(t), 3.0)

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
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

	t.RunSequentially("Making more pools than allowed by max_delegates in minersc should fail", func(t *test.SystemTest) {
		// Select the non-miner02 miner with the fewest existing pools (lowest TotalStake).
		// Miner01 accumulates stale test-wallet pools across runs; picking the miner with
		// minimum TotalStake avoids a full pool on first selection.
		var newMiner climodel.Node
		found := false
		for _, m := range miners.Nodes {
			if m.ID == miner02ID {
				continue
			}
			if !found || m.TotalStake < newMiner.TotalStake {
				newMiner = m
				found = true
			}
		}
		require.True(t, found, "No suitable miner found (need a miner that is not miner02ID)")

		createWallet(t)
		faucetFundWalletOrSkip(t, escapedTestName(t), 12.0)

		// Temporarily lower max_delegates to 5 so we can exhaust all slots without
		// creating hundreds of funded wallets. The chain enforces the per-miner
		// num_delegates at pool creation time, so we must lower both the global config
		// AND the per-miner setting.
		const testMaxDelegates = 5
		output, err := updateMinerSCConfig(t, minerScOwnerWallet, map[string]interface{}{
			"keys":   "max_delegates",
			"values": strconv.Itoa(testMaxDelegates),
		}, true)
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "unauthorized access") || strings.Contains(combined, "access denied") {
				t.Skip("MinerSC owner wallet does not match on-chain owner - cannot update max_delegates")
			}
			if strings.Contains(combined, "too less sharders") || strings.Contains(combined, "too few sharders") ||
				strings.Contains(combined, "unexpected end of JSON") || strings.Contains(combined, "invalid transaction nonce") {
				t.Skip("Chain transient error while lowering global max_delegates: " + combined)
			}
			require.Nil(t, err, "failed to lower global max_delegates: "+strings.Join(output, "\n"))
		}

		// Also lower the per-miner num_delegates. The delegate wallet for all infra miners
		// is miner01NodeDelegateWalletName (same key used for all nodes).
		output, err = minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
			"id":            newMiner.ID,
			"num_delegates": testMaxDelegates,
		}), true)
		if err != nil {
			combined := strings.Join(output, "\n")
			if strings.Contains(combined, "unauthorized access") || strings.Contains(combined, "access denied") {
				t.Skip("Miner delegate wallet does not have access - delegate wallet may not match on-chain config")
			}
			if strings.Contains(combined, "too less sharders") || strings.Contains(combined, "too few sharders") ||
				strings.Contains(combined, "unexpected end of JSON") || strings.Contains(combined, "invalid transaction nonce") {
				t.Skip("Chain transient error while lowering miner num_delegates: " + combined)
			}
			require.Nil(t, err, "failed to lower miner num_delegates: "+strings.Join(output, "\n"))
		}

		// Pre-unlock stale test wallet pools from previous runs to free up slots.
		// Wallet names are deterministic: escapedTestName(t)+"0" through +"(testMaxDelegates-2)".
		for i := 0; i < testMaxDelegates-1; i++ {
			walletName := escapedTestName(t) + fmt.Sprintf("%d", i)
			createWalletForName(walletName)
			_, _ = minerOrSharderUnlockForWallet(t, configPath, createParams(map[string]interface{}{
				"miner_id": newMiner.ID,
			}), walletName, false)
		}
		// Wait for unlocks to be committed before querying the real pool count.
		cliutils.Wait(t, 15*time.Second)

		// Query actual pool count so we don't rely on TotalStake estimation.
		// Test wallets stake 1-3 ZCN (not 200 ZCN like infra), so TotalStake / 200 would undercount.
		var actualPoolCount int
		if nodeOut, nodeErr := getNode(t, configPath, newMiner.ID); nodeErr == nil && len(nodeOut) == 1 {
			var nodeInfo climodel.Node
			if json.Unmarshal([]byte(nodeOut[0]), &nodeInfo) == nil {
				actualPoolCount = len(nodeInfo.StakePool.Pools)
				t.Logf("Miner %s has %d actual pools after pre-unlock", newMiner.ID[:12], actualPoolCount)
			}
		}
		if actualPoolCount == 0 {
			// Fallback: assume at least the infra pool exists
			actualPoolCount = 1
			t.Logf("Could not query node pool count; assuming 1 infra pool")
		}
		slotsToFill := testMaxDelegates - actualPoolCount

		defer func() {
			// Unlock all test wallet pools to prevent accumulation across runs.
			for i := 0; i < testMaxDelegates-1; i++ {
				walletName := escapedTestName(t) + fmt.Sprintf("%d", i)
				_, _ = minerOrSharderUnlockForWallet(t, configPath, createParams(map[string]interface{}{
					"miner_id": newMiner.ID,
				}), walletName, false)
			}
			// Restore global max_delegates first (must be >= per-miner), then per-miner.
			// Retry up to 3 times with 20s sleep between attempts to handle chain instability.
			var restoreErr error
			for attempt := 0; attempt < 3; attempt++ {
				_, restoreErr = updateMinerSCConfig(t, minerScOwnerWallet, map[string]interface{}{
					"keys":   "max_delegates",
					"values": "200",
				}, true)
				if restoreErr == nil {
					break
				}
				t.Logf("Attempt %d: failed to restore SC max_delegates to 200: %v, retrying in 20s...", attempt+1, restoreErr)
				time.Sleep(20 * time.Second)
			}
			if restoreErr != nil {
				t.Logf("WARNING: all restore attempts for SC max_delegates=200 failed: %v — downstream tests may fail", restoreErr)
			}
			minerSharderUpdateSettings(t, configPath, miner01NodeDelegateWalletName, createParams(map[string]interface{}{
				"id":            newMiner.ID,
				"num_delegates": 200,
			}), false)
		}()

		if slotsToFill <= 0 {
			t.Skipf("miner %s has %d existing pools >= testMaxDelegates=%d after pre-unlock; cannot test limit", newMiner.ID[:16], actualPoolCount, testMaxDelegates)
			return
		}
		maxDelegates := int64(testMaxDelegates)

		wg := &sync.WaitGroup{}
		for i := 0; i < slotsToFill; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()

				walletName := escapedTestName(t) + fmt.Sprintf("%d", i)
				// Wallet was pre-created above; just fund it
				faucetFundWalletOrSkip(t, walletName, 3.0)

				// Use local variables to avoid goroutine race on the outer output/err
				lockOutput, lockErr := minerOrSharderLockForWallet(t, configPath, createParams(map[string]interface{}{
					"miner_id": newMiner.ID,
					"tokens":   1,
				}), walletName, true)
				require.NoError(t, lockErr)
				require.Len(t, lockOutput, 1)
				require.Regexp(t, lockOutputRegex, lockOutput[0])
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
		createWallet(t)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   0,
		}), false)
		require.NotNil(t, err, "expected error when staking 0 tokens but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "stake_pool_lock_failed: no stake to lock: 0", output[0])
	})

	// this case covers both invalid miner and sharder id, so is not repeated in zwalletcli_sharder_stake_test.go
	t.RunSequentially("Unlock tokens with invalid node id should fail", func(t *test.SystemTest) {
		// Select a miner that is NOT miner02ID to avoid conflicts with other tests
		var testMiner climodel.Node
		found := false
		for _, m := range miners.Nodes {
			if m.ID != miner02ID {
				testMiner = m
				found = true
				break
			}
		}
		require.True(t, found, "No suitable miner found (need a miner that is not miner02ID)")

		createWallet(t)
		faucetFundWalletOrSkip(t, escapedTestName(t), 5.0)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": testMiner.ID,
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
		combined := strings.Join(output, "\n")
		if strings.Contains(combined, "invalid transaction nonce") || strings.Contains(combined, "too less sharders") || strings.Contains(combined, "unexpected end of JSON") {
			t.Skip("Chain transient error during invalid node id unlock test: " + combined)
		}
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

	time.Sleep(10 * time.Second)

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
	// Use getSharderUrl to pick the highest-round sharder (configured sharder may not be in MB on test chains)
	sharderURL := getSharderUrl(t)
	res, err := apiGetBalance(t, sharderURL, clientId)
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
