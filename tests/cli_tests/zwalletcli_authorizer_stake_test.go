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

func TestAuthorizerStake(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Staking tokens against valid authorizer with valid tokens should work")

	var authorizer climodel.Validator
	t.TestSetup("get sharders", func() {
		if _, err := os.Stat("./config/" + sharder01NodeDelegateWalletName + "_wallet.json"); err != nil {
			t.Skipf("miner node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
		}

		createWallet(t)

		createWalletForName(sharder01NodeDelegateWalletName)

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

	t.Parallel()

	t.RunWithTimeout("Staking tokens against valid authorizer with valid tokens should work", 5*time.Minute, func(t *test.SystemTest) { // todo: slow
		createWallet(t)

		output, err := minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
			"tokens":   2.0,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		poolsInfo, err := pollForPoolInfo(t, miner.ID)
		require.Nil(t, err)
		require.Equal(t, float64(2.0), intToZCN(poolsInfo.Balance))

		// Unlock should work
		output, err = minerOrSharderUnlock(t, configPath, createParams(map[string]interface{}{
			"miner_id": miner.ID,
		}), true)
		require.Nil(t, err, "error unlocking tokens against a node")
		require.Len(t, output, 1)
		require.Equal(t, "tokens unlocked", output[0])

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)

		require.NotNil(t, err, "expected error when requesting unlocked pool but got output", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, `resource_not_found: can't find pool stats`, output[0])
	})


}

func authorizerLock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return authorizerLockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func authorizerLockForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("locking tokens against authorizers...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zbox sp-lock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}

func authorizerUnlock(t *test.SystemTest, cliConfigFilename, params string, retry bool) ([]string, error) {
	return minerOrSharderUnlockForWallet(t, cliConfigFilename, params, escapedTestName(t), retry)
}

func authorizerUnlockForWallet(t *test.SystemTest, cliConfigFilename, params, wallet string, retry bool) ([]string, error) {
	t.Log("unlocking tokens from authorizer pool...")
	if retry {
		return cliutils.RunCommand(t, fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename), 3, time.Second)
	} else {
		return cliutils.RunCommandWithoutRetry(fmt.Sprintf("./zbox sp-unlock %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename))
	}
}