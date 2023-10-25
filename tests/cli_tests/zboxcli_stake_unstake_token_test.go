package cli_tests

import (
	"encoding/json"
	"fmt"
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

func TestStakeUnstakeTokens(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Staked tokens should move from wallet to Provider's stake pool, unstaking should move tokens back to wallet")

	t.Parallel()

	t.Run("Staked tokens should move from wallet to Provider's stake pool, unstaking should move tokens back to wallet", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "Error getting wallet", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		balanceBefore, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     1.0,
		}), true)
		require.Nil(t, err, "Error staking tokens", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("tokens locked, txn hash: ([a-f0-9]{64})"), output[0])
		require.Nil(t, err, "Error extracting txn hash from sp-lock output", strings.Join(output, "\n"))

		// Wallet balance should decrease by locked amount
		balanceAfter, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		// less than balanceBefore - 1 due to txn fee
		require.Less(t, balanceAfter, balanceBefore-1)

		// Use sp-info to check the staked tokens in blobber's stake pool
		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		stakePool := climodel.StakePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates := stakePool.Delegate

		// Pool Id returned by sp-lock should be found in sp-info with correct balance
		found := false
		for _, delegate := range delegates {
			if delegate.ID == wallet.ClientID {
				t.Log("Pool ID returned by sp-lock found in stake pool info...")
				found = true
				require.Equal(t, int64(10000000000), delegate.Balance, "User Locked 5000000000 SAS but the pool balance is ", delegate.Balance)
				require.Equal(t, wallet.ClientID, delegate.DelegateID, "Delegate ID of pool created by sp-lock is not equal to the wallet ID of User.",
					"Delegate ID: ", delegate.DelegateID, "Wallet ID: ", wallet.ClientID)
			}
		}
		require.True(t, found, fmt.Sprintf("Pool id returned by sp-lock not found in blobber's sp-info: %s, clientID: %s",
			strings.Join(output, "\n"), wallet.ClientID))

		// Unstake the tokens
		output, err = unstakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
		}), true)
		require.Nil(t, err, "Error unstaking tokens from stake pool", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked: 10000000000, pool deleted", output[0])

		// Wallet balance should increase by unlocked amount
		output, err = getBalance(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile(`Balance: \d*\.?\d+ ZCN \(\d*\.?\d+ USD\)$`), output[0])
		newBalance := regexp.MustCompile(`\d+\.?\d* [um]?(ZCN|SAS)`).FindString(output[0])
		newBalanceValue, err := strconv.ParseFloat(strings.Fields(newBalance)[0], 64)
		require.Nil(t, err, "error parsing float from balance")
		require.GreaterOrEqual(t, newBalanceValue, float64(1.0))

		// Pool Id must be deleted from stake pool now
		output, err = stakePoolInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"json":       "",
		}))
		require.Nil(t, err, "Error fetching stake pool info", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &stakePool)
		require.Nil(t, err, "Error unmarshalling stake pool info", strings.Join(output, "\n"))
		require.NotEmpty(t, stakePool)

		delegates = stakePool.Delegate

		// Pool Id returned by sp-lock should be deleted from sp-info
		found = false
		for _, delegate := range delegates {
			if delegate.ID == wallet.ClientID {
				t.Log("Delegate ID found in stake pool information but it should have been deleted after unlocking staked tokens...")
				found = true
			}
		}
		require.False(t, found, "Pool id found in blobber's sp-info even after unlocking tokens", strings.Join(output, "\n"))
	})

	t.Run("Staking tokens without specifying amount of tokens to lock should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": "-",
		}), false)
		require.NotNil(t, err, "Expected error when amount of tokens to stake is not specified", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "missing required 'tokens' flag", output[0])
	})

	t.Run("Staking tokens without specifying provider should generate corresponding error", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"tokens": 1.0,
		}), false)
		require.NotNil(t, err, "Expected error when blobber to stake tokens to is not specified", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1)
		require.Equal(t, "missing flag: one of 'miner_id', 'sharder_id', 'blobber_id', 'validator_id', 'authorizer_id' is required", output[0])
	})

	t.Run("Staking more tokens than in wallet should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Wallet balance before staking tokens
		balance, err := getBalanceZCN(t, configPath)
		require.Nil(t, err, "Error fetching balance", strings.Join(output, "\n"))

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     10,
		}), false)
		require.NotNil(t, err, "Expected error when staking more tokens than in wallet", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1)
		require.Equal(t, "Failed to lock tokens in stake pool: stake_pool_lock_failed: stake pool digging error: lock amount is greater than balance", output[0])

		// Wallet balance after staking tokens
		balance2, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Less(t, balance2, balance) // pay txn fee
	})

	t.Run("Staking 0 tokens against blobber should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))
		t.Logf("output: %v", output)
		txnHash := getTransactionHash(output, false)

		// Wallet balance before staking tokens
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		t.Logf("wallet balance: %v, txn hash: %v", balance, txnHash)
		output, err = getTransaction(t, configPath, createParams(map[string]interface{}{
			"hash": txnHash,
		}))
		require.NoError(t, err)
		t.Logf("transaction output: %v", output)

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     0.0,
		}), false)
		require.NotNil(t, err, "Expected error when staking 0 tokens than in stake pool", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1)
		require.Equal(t, "Failed to lock tokens in stake pool: stake_pool_lock_failed: no stake to lock: 0", output[0])

		// Wallet balance after staking tokens
		balance2, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Less(t, balance2, balance) // pay txn fee
	})

	t.Run("Staking negative tokens should fail", func(t *test.SystemTest) {
		output, err := createWallet(t, configPath)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		// Wallet balance before staking tokens
		balance, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)

		blobbers := []climodel.BlobberInfo{}
		output, err = listBlobbers(t, configPath, "--json")
		require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &blobbers)
		require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
		require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

		// Pick a random blobber
		blobber := blobbers[time.Now().Unix()%int64(len(blobbers))]

		// Stake tokens against this blobber
		output, err = stakeTokens(t, configPath, createParams(map[string]interface{}{
			"blobber_id": blobber.Id,
			"tokens":     -1.0,
		}), false)
		require.NotNil(t, err, "Expected error when staking negative tokens than in stake pool", strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1)
		require.Equal(t, "invalid token amount: negative", output[0])

		// Wallet balance after staking tokens
		balance2, err := getBalanceZCN(t, configPath)
		require.NoError(t, err)
		require.Equal(t, balance2, balance)
	})
}

func listBlobbers(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox ls-blobbers %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func getTransaction(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Get transaction list...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zwallet verify %s --silent --configDir ./config --config %s", params, cliConfigFilename), 3, time.Second*2)
}

func stakeTokens(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return stakeTokensForWallet(t, cliConfigFilename, escapedTestName(t), params, retry)
}

func stakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string, retry bool) ([]string, error) {
	t.Log("Staking tokens...")
	cmd := fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func stakePoolInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Fetching stake pool info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func unstakeTokens(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return unstakeTokensForWallet(t, cliConfigFilename, escapedTestName(t), params, retry)
}

func unstakeTokensForWallet(t *test.SystemTest, cliConfigFilename, wallet, params string, retry bool) ([]string, error) {
	t.Log("Unlocking tokens from stake pool...")
	cmd := fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func getBlobbersList(t *test.SystemTest) []climodel.BlobberInfo {
	blobbers := []climodel.BlobberInfo{}
	output, err := listBlobbers(t, configPath, "--json")
	require.Nil(t, err, "Error listing blobbers", strings.Join(output, "\n"))
	require.Len(t, output, 1)

	err = json.Unmarshal([]byte(output[0]), &blobbers)
	require.Nil(t, err, "Error unmarshalling blobber list", strings.Join(output, "\n"))
	require.True(t, len(blobbers) > 0, "No blobbers found in blobber list")

	return blobbers
}
