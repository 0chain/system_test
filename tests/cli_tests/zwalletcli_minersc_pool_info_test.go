package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerSCUserPoolInfo(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + minerNodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("Miner node owner wallet located at %s is missing", "./config/"+minerNodeDelegateWalletName+"_wallet.json")
	}

	minerNodeDelegateWallet, err := getWalletForName(t, configPath, minerNodeDelegateWalletName)
	require.Nil(t, err, "error fetching wallet")

	sharderNodeDelegateWallet, err := getWalletForName(t, configPath, sharderNodeDelegateWalletName)
	require.Nil(t, err, "error fetching wallet")

	t.Run("Getting MinerSC Stake pools of a wallet before and after locking against a miner should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		// before locking tokens against a miner
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Empty(t, poolsInfo.Pools)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     minerNodeDelegateWallet.ClientID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])
		poolId := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		// after locking tokens against a miner
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools["miner"][minerNodeDelegateWallet.ClientID], 1)
		require.Equal(t, poolId, poolsInfo.Pools["miner"][minerNodeDelegateWallet.ClientID][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["miner"][minerNodeDelegateWallet.ClientID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      minerNodeDelegateWallet.ClientID,
			"pool_id": poolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.Run("Getting MinerSC Stake pools of a wallet before and after locking against a sharder should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		// before locking tokens against a sharder
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Empty(t, poolsInfo.Pools)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharderNodeDelegateWallet.ClientID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		poolId := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		// after locking tokens against sharder
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools["sharder"][sharderNodeDelegateWallet.ClientID], 1)
		require.Equal(t, poolId, poolsInfo.Pools["sharder"][sharderNodeDelegateWallet.ClientID][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["sharder"][sharderNodeDelegateWallet.ClientID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      sharderNodeDelegateWallet.ClientID,
			"pool_id": poolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.Run("Getting MinerSC pools info for a different client id than wallet owner should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet")

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		targetWalletName := escapedTestName(t) + "_target"
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     minerNodeDelegateWallet.ClientID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error locking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		minerPoolId := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharderNodeDelegateWallet.ClientID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error locking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		sharderPoolId := regexp.MustCompile("[0-9a-z]{64}").FindString(output[0])

		output, err = stakePoolsInMinerSCInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"client_id": wallet.ClientID,
		}), targetWalletName, true)
		require.Nil(t, err, "error fetching Miner SC User Pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pools")

		require.Len(t, poolsInfo.Pools["miner"][minerNodeDelegateWallet.ClientID], 1)
		require.Equal(t, minerPoolId, poolsInfo.Pools["miner"][minerNodeDelegateWallet.ClientID][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["miner"][minerNodeDelegateWallet.ClientID][0].Balance))

		require.Len(t, poolsInfo.Pools["sharder"][sharderNodeDelegateWallet.ClientID], 1)
		require.Equal(t, sharderPoolId, poolsInfo.Pools["sharder"][sharderNodeDelegateWallet.ClientID][0].ID)
		require.Equal(t, float64(1), intToZCN(poolsInfo.Pools["sharder"][sharderNodeDelegateWallet.ClientID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      minerNodeDelegateWallet.ClientID,
			"pool_id": minerPoolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}

		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"id":      sharderNodeDelegateWallet.ClientID,
			"pool_id": sharderPoolId,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})
}

func stakePoolsInMinerSCInfo(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return stakePoolsInMinerSCInfoForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func stakePoolsInMinerSCInfoForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("fetching mn-user-info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-user-info %s --json --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-user-info %s --json --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
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
