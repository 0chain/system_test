package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
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

	t.Run("Getting MinerSC Stake pools of a wallet before and after making a pool should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		// before locking tokens against a miner
		output, err = stakePoolsInMinerSCInfo(t, configPath, "", true)
		require.Nil(t, err, "error fetching stake pools")
		require.Len(t, output, 1)

		var pools climodel.MinerSCDelegatePoolInfo
		err = json.Unmarshal([]byte(output[0]), &pools)
		require.Nil(t, err, "error unmarshalling minerSC pools info")
		require.Empty(t, pools)

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id": minerNodeDelegateWallet.ClientID,
		}), true)
		require.Nil(t, err, "error ")
	})
}

func stakePoolsInMinerSCInfo(t *testing.T, cliConfigFilename, params string, retry bool) ([]string, error) {
	return stakePoolsInMinerSCInfoForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func stakePoolsInMinerSCInfoForWallet(t *testing.T, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
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
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zwallet mn-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}
