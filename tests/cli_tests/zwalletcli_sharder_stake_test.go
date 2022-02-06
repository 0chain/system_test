package cli_tests

import (
	"os"
	"regexp"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestSharderStake(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + sharderNodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+sharderNodeDelegateWalletName+"_wallet.json")
	}

	sharders := getShardersListForWallet(t, sharderNodeDelegateWalletName)

	sharderNodeDelegateWallet, err := getWalletForName(t, configPath, sharderNodeDelegateWalletName)
	require.Nil(t, err, "error fetching sharderNodeDelegate wallet")

	var sharder climodel.Sharder
	for _, sharder = range sharders {
		if sharder.ID != sharderNodeDelegateWallet.ClientID {
			break
		}
	}

	var (
		lockOutputRegex = regexp.MustCompile("locked with: [a-f0-9]{64}")
		poolIdRegex     = regexp.MustCompile("[a-f0-9]{64}")
	)

	t.Run("Staking tokens against valid sharder with valid tokens should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharder.ID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error locking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		poolId := poolIdRegex.FindString(output[0])

		poolsInfo, err := pollForPoolInfo(t, sharder.ID, poolId)
		require.Nil(t, err)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      sharder.ID,
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
			"id":     sharder.ID,
			"tokens": 1,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"error": "verify transaction failed"}`, output[0])
	})

	t.Run("Staking negative tokens against valid sharder should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharder.ID,
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
		require.Equal(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharder.ID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		poolId := poolIdRegex.FindString(output[0])

		poolsInfo, err := pollForPoolInfo(t, sharder.ID, poolId)
		require.Nil(t, err)
		balance := getBalanceFromSharders(t, wallet.ClientID)
		require.Equal(t, balance, poolsInfo.RewardPaid)

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      sharder.ID,
			"pool_id": poolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})
}
