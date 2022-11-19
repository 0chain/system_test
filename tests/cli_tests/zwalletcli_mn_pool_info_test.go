package cli_tests

import (
	"encoding/json"
	"github.com/0chain/system_test/internal/api/util/test"
	"os"
	"regexp"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestMinerSharderPoolInfo(testSetup *testing.T) {
	t := test.SystemTest{T: testSetup}

	t.Parallel()

	if _, err := os.Stat("./config/" + sharder01NodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
	}

	sharders := getShardersListForWallet(testSetup, sharder01NodeDelegateWalletName)

	sharderNodeDelegateWallet, err := getWalletForName(testSetup, configPath, sharder01NodeDelegateWalletName)
	require.Nil(t, err, "error fetching sharderNodeDelegate wallet")

	var sharder climodel.Sharder
	for _, sharder = range sharders {
		if sharder.ID != sharderNodeDelegateWallet.ClientID {
			break
		}
	}

	if _, err := os.Stat("./config/" + miner02NodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+miner02NodeDelegateWalletName+"_wallet.json")
	}

	miners := getMinersListForWallet(testSetup, miner02NodeDelegateWalletName)

	var miner climodel.Node
	for _, miner = range miners.Nodes {
		if miner.ID == miner03ID {
			break
		}
	}

	var (
		lockOutputRegex = regexp.MustCompile("locked with: [a-f0-9]{64}")
	)

	t.Run("Miner pool info after locking against miner should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     miner.ID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		var poolsInfo climodel.DelegatePool
		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": miner.ID,
		}), true)
		require.Nil(t, err, "error fetching Miner Sharder pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner Sharder pools")
	})

	t.Run("Miner pool info after locking against sharder should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 9.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharder.ID,
			"tokens": 5,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])

		var poolsInfo climodel.DelegatePool

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": sharder.ID,
		}), true)
		require.Nil(t, err, "error fetching Miner Sharder pools")
		require.Len(t, output, 1)

		err = json.Unmarshal([]byte(output[0]), &poolsInfo)
		require.Nil(t, err, "error unmarshalling Miner Sharder pools")
	})

	t.Run("Miner/Sharder pool info for invalid node id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id": "abcdefgh",
		}), false)
		require.NotNil(t, err, "expected error when trying to fetch pool info from invalid id")
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"code":"resource_not_found","error":"resource_not_found: can't get miner node: value not present"}`, output[0])
	})
}
