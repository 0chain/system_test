package cli_tests

import (
	"encoding/json"
	"os"
	"regexp"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestSharderStake(t *testing.T) {
	if _, err := os.Stat("./config/" + sharder01NodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
	}

	sharders := getShardersListForWallet(t, sharder01NodeDelegateWalletName)

	sharderNodeDelegateWallet, err := getWalletForName(t, configPath, sharder01NodeDelegateWalletName)
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

	t.Run("Staking tokens against valid sharder with valid tokens should work, unlocking should work", func(t *testing.T) {
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

		// unlock should work
		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      sharder.ID,
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error unlocking tokens against a node")
		require.Len(t, output, 1)
		require.Equal(t, "tokens will be unlocked next VC", output[0])

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id":      sharder.ID,
			"pool_id": poolId,
		}), true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Equal(t, int(climodel.Deleting), poolsInfo.Status)
	})

	t.Run("Multiple stakes against a sharder should create multiple pools", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		var poolsInfoBefore climodel.MinerSCUserPoolsInfo
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)
		err = json.Unmarshal([]byte(output[0]), &poolsInfoBefore)
		require.Nil(t, err, "error unmarshalling pools info")
		beforeNumPools := len(poolsInfoBefore.Pools)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharder.ID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		poolId1 := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharder.ID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		poolId2 := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools[sharder.ID], 2)

		found1 := false
		found2 := false
		for _, pool := range poolsInfo.Pools[sharder.ID] {
			if pool.ID == poolId1 {
				found1 = true
				require.Equal(t, float64(1), intToZCN(poolsInfo.Pools[sharder.ID][beforeNumPools].Balance))
			}
			if pool.ID == poolId2 {
				found2 = true
				require.Equal(t, float64(1), intToZCN(poolsInfo.Pools[sharder.ID][1+beforeNumPools].Balance))
			}
		}
		require.Equal(t, true, found1)
		require.Equal(t, true, found2)
	})

	t.Run("Staking tokens with insufficient balance should fail", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharder.ID,
			"tokens": 1,
		}), false)
		require.NotNil(t, err, "expected error when staking tokens with insufficient balance but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `delegate_pool_add: digging delegate pool: lock amount is greater than balance`, output[0])
	})

	t.Run("Staking negative tokens against valid sharder should fail", func(t *testing.T) {
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
		require.Equal(t, `invalid token amount: negative`, output[0])
	})

	// todo rewards not transferred to wallet until a collect reward transaction
	t.Run("Staking tokens against sharder should return intrests to wallet", func(t *testing.T) {
		t.Skip("rewards not transferred to wallet until a collect reward transaction")

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error getting wallet")

		output, err = executeFaucetWithTokens(t, configPath, 1.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

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
		require.GreaterOrEqual(t, balance, poolsInfo.Reward)

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      sharder.ID,
			"pool_id": poolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.Run("Unlock tokens with invalid pool id should fail", func(t *testing.T) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      sharder.ID,
			"pool_id": "abcdefgh",
		}), false)
		require.NotNil(t, err, "expected error when using invalid node id")
		require.Len(t, output, 1)
		require.Equal(t, "delegate_pool_del: pool does not exist for deletion", output[0])
	})
}
