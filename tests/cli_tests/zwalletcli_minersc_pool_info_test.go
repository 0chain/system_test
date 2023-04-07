package cli_tests

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestMinerSCUserPoolInfo(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)

	t.RunSequentially("Getting MinerSC Stake pools of a wallet before and after locking against a miner should work", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))
		w, err := getWallet(t, configPath)
		require.NoError(t, err)

		// before locking tokens against a miner
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Empty(t, poolsInfo.Pools)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner01ID,
			"tokens":   5,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [a-z0-9]{64}"), output[0])

		// after locking tokens against a miner
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools[miner01ID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[miner01ID][0].ID)
		require.Equal(t, float64(5), intToZCN(poolsInfo.Pools[miner01ID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner01ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.RunSequentially("Getting MinerSC Stake pools of a wallet before and after locking against a sharder should work", func(t *test.SystemTest) {
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
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
			"sharder_id": sharder01ID,
			"tokens":     5,
		}), true)
		require.Nil(t, err, "error staking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])
		w, err := getWallet(t, configPath)
		require.NoError(t, err)

		// after locking tokens against sharder
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching Miner SC User pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pool")
		require.Len(t, poolsInfo.Pools[sharder01ID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[sharder01ID][0].ID)
		require.Equal(t, float64(5), intToZCN(poolsInfo.Pools[sharder01ID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder01ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})

	t.RunSequentiallyWithTimeout("Getting MinerSC pools info for a different client id than wallet owner should work", 100*time.Second, func(t *test.SystemTest) { //TODO: slow
		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet")

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		targetWalletName := escapedTestName(t) + "_target"
		output, err = registerWalletForName(t, configPath, targetWalletName)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner01ID,
			"tokens":   5,
		}), true)
		require.Nil(t, err, "error locking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		waitForStakePoolActive(t)
		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder01ID,
			"tokens":     5,
		}), true)
		require.Nil(t, err, "error locking tokens against node")
		require.Len(t, output, 1)
		require.Regexp(t, regexp.MustCompile("locked with: [0-9a-z]{64}"), output[0])

		output, err = stakePoolsInMinerSCInfoForWallet(t, configPath, createParams(map[string]interface{}{
			"client_id": wallet.ClientID,
		}), targetWalletName, true)
		require.Nil(t, err, "error fetching Miner SC User Pools")
		require.Len(t, output, 1)

		var poolsInfo climodel.MinerSCUserPoolsInfo
		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner SC User Pools")

		w, err := getWallet(t, configPath)
		require.NoError(t, err)

		require.Len(t, poolsInfo.Pools[miner01ID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[miner01ID][0].ID)
		require.Equal(t, float64(5), intToZCN(poolsInfo.Pools[miner01ID][0].Balance))

		require.Len(t, poolsInfo.Pools[sharder01ID], 1)
		require.Equal(t, w.ClientID, poolsInfo.Pools[sharder01ID][0].ID)
		require.Equal(t, float64(5), intToZCN(poolsInfo.Pools[sharder01ID][0].Balance))

		// teardown
		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner01ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}

		_, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"sharder_id": sharder01ID,
		}), true)
		if err != nil {
			t.Log("error unlocking tokens after test: ", t.Name())
		}
	})
}

func stakePoolsInMinerSCInfo(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return stakePoolsInMinerSCInfoForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func stakePoolsInMinerSCInfoForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("fetching mn-user-info...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-user-info %s --json --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-user-info %s --json --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
