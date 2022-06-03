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

func TestMinerSharderPoolInfo(t *testing.T) {
	t.Parallel()

	if _, err := os.Stat("./config/" + sharder01NodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+sharder01NodeDelegateWalletName+"_wallet.json")
	}
	if _, err := os.Stat("./config/" + sharderNodeWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+sharderNodeWalletName+"_wallet.json")
	}

	if _, err := os.Stat("./config/" + miner01NodeDelegateWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+miner01NodeDelegateWalletName+"_wallet.json")
	}
	if _, err := os.Stat("./config/" + minerNodeWalletName + "_wallet.json"); err != nil {
		t.Skipf("miner node owner wallet located at %s is missing", "./config/"+minerNodeWalletName+"_wallet.json")
	}

	minerNodeWallet, err := getWalletForName(t, configPath, minerNodeWalletName)
	require.Nil(t, err, "error fetching minerNodeDelegate wallet")

	sharderNodeWallet, err := getWalletForName(t, configPath, sharderNodeWalletName)
	require.Nil(t, err, "error fetching sharderNodeDelegate wallet")

	var (
		lockOutputRegex = regexp.MustCompile("locked with: [a-f0-9]{64}")
		poolIdRegex     = regexp.MustCompile("[a-f0-9]{64}")
	)

	t.Run("Miner pool info after locking against miner should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     minerNodeWallet.ClientID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		poolId := poolIdRegex.FindString(output[0])

		var poolsInfo climodel.DelegatePool
		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id":      minerNodeWallet.ClientID,
			"pool_id": poolId,
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

		output, err = executeFaucetWithTokens(t, configPath, 2.0)
		require.Nil(t, err, "error executing faucet", strings.Join(output, "\n"))

		output, err = minerOrSharderLock(t, configPath, createParams(map[string]interface{}{
			"id":     sharderNodeWallet.ClientID,
			"tokens": 1,
		}), true)
		require.Nil(t, err, "error staking tokens against a node")
		require.Len(t, output, 1)
		require.Regexp(t, lockOutputRegex, output[0])
		poolId := poolIdRegex.FindString(output[0])

		var poolsInfo climodel.DelegatePool
		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id":      sharderNodeWallet.ClientID,
			"pool_id": poolId,
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
			"id":      "abcdefgh",
			"pool_id": "dummy pool id",
		}), false)
		require.NotNil(t, err, "expected error when trying to fetch pool info from invalid id")
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"code":"resource_not_found","error":"resource_not_found: can't get miner node: value not present"}`, output[0])
	})

	t.Run("Miner pool info for invalid pool id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id":      minerNodeWallet.ClientID,
			"pool_id": "dummy pool id",
		}), false)
		require.NotNil(t, err, "expected error when trying to fetch pool info from invalid id")
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"code":"resource_not_found","error":"resource_not_found: can't find pool stats"}`, output[0])
	})

	t.Run("Sharder pool info for invalid pool id should fail", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "error registering wallet", strings.Join(output, "\n"))

		output, err = minerSharderPoolInfo(t, configPath, createParams(map[string]interface{}{
			"id":      sharderNodeWallet.ClientID,
			"pool_id": "dummy pool id",
		}), false)
		require.NotNil(t, err, "expected error when trying to fetch pool info from invalid id")
		require.Len(t, output, 1)
		require.Equal(t, `fatal:{"code":"resource_not_found","error":"resource_not_found: can't find pool stats"}`, output[0])
	})
}
