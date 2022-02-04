package cli_tests

import (
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerSharderStake(t *testing.T) {
	t.Parallel()

	miners := getMinersList(t)
	minerId := miners.Nodes[1].ID

	sharders := getShardersList(t)
	sharderId := reflect.ValueOf(sharders).MapKeys()[1].String()

	t.Run("Stake tokens against a miner should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     minerId,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])
		poolId := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		var poolsInfo climodel.MinerSCUserPoolsInfo
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools["miner"][minerId], 1)
		require.Equal(t, poolId, poolsInfo.Pools["miner"][minerId][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["miner"][minerId][0].Balance))
	})

	t.Run("Multiple stakes against a miner should create multiple pools", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     minerId,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])
		poolId1 := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     minerId,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])
		poolId2 := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		var poolsInfo climodel.MinerSCUserPoolsInfo
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools["miner"][minerId], 2)
		require.Equal(t, poolId1, poolsInfo.Pools["miner"][minerId][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["miner"][minerId][0].Balance))
		require.Equal(t, poolId2, poolsInfo.Pools["miner"][minerId][1].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["miner"][minerId][1].Balance))
	})

	t.Run("Stake tokens against a sharder should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharderId,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		poolId := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools["sharder"][sharderId], 1)
		require.Equal(t, poolId, poolsInfo.Pools["sharder"][sharderId][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["sharder"][sharderId][0].Balance))
	})

	t.Run("Multiple stakes against a sharder should create multiple pools", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharderId,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		poolId1 := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharderId,
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
		require.Len(t, poolsInfo.Pools["sharder"][sharderId], 2)
		require.Equal(t, poolId1, poolsInfo.Pools["sharder"][sharderId][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["sharder"][sharderId][0].Balance))
		require.Equal(t, poolId2, poolsInfo.Pools["sharder"][sharderId][1].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["sharder"][sharderId][1].Balance))
	})
}

func minerOrSharderLock(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerOrSharderLockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func minerOrSharderLockForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("locking tokens against miner/sharder...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}

func minerOrSharderUnlock(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerOrSharderUnlockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func minerOrSharderUnlockForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("unlocking tokens from miner/sharder pool...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
