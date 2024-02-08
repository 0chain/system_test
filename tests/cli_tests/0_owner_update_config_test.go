package cli_tests

import (
	"os"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"

	"github.com/0chain/system_test/internal/api/util/test"

	"github.com/stretchr/testify/require"
)

func TestOwnerUpdate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("should allow update of owner: StorageSC")

	var newOwnerWallet *climodel.Wallet
	var newOwnerName string
	var err error

	t.TestSetup("Create new owner wallet", func() {
		if _, err := os.Stat("./config/" + scOwnerWallet + "_wallet.json"); err != nil {
			t.Skipf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		createWallet(t)
		newOwnerWallet, err = getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet")

		newOwnerName = escapedTestName(t)
	})

	t.RunSequentiallyWithTimeout("should allow update of owner: StorageSC", 2*time.Minute, func(t *test.SystemTest) {
		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateStorageSCConfig(t, newOwnerName, map[string]string{
				ownerKey: oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err := updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			ownerKey: newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))

		var storageSCCommitPeriod int64 = 200
		lfb := getLatestFinalizedBlock(t)
		lfbRound := lfb.Round
		updateConfigRound := lfbRound + (storageSCCommitPeriod - (lfbRound % storageSCCommitPeriod))
		var frequency time.Duration = 2
		var found bool
		for i := 0; i < int(storageSCCommitPeriod)/int(frequency); i++ {
			t.Logf("fetching lfb in: %ds...", frequency)
			time.Sleep(frequency * time.Second)
			lfb = getLatestFinalizedBlock(t)
			if lfb.Round >= updateConfigRound {
				found = true
				break
			}
		}

		require.True(t, found, "operation timed out to reach valid round")

		output, err = getStorageSCConfig(t, configPath, true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		cfgAfter, _ := keyValuePairStringToMap(output)
		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// Updating config with old owner should fail
		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			"max_read_price": "99",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0])
	})

	t.RunSequentially("should allow update of owner: MinerSC", func(t *test.SystemTest) {
		createWallet(t)

		ownerKey := "owner_id"
		oldOwner := "1746b06bb09f55ee01b33b5e2e055d6cc7a900cb57c0a3a5eaabb8a0e7745802"

		t.Cleanup(func() {
			output, err := updateMinerSCConfig(t, newOwnerName, map[string]interface{}{
				"keys":   ownerKey,
				"values": oldOwner,
			}, true)
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 2, strings.Join(output, "\n"))
		})

		output, err := updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "minersc smart contract settings updated", output[0], strings.Join(output, "\n"))

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(output)

		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// Should fail update with old owner
		output, err = updateMinerSCConfig(t, scOwnerWallet, map[string]interface{}{
			"keys":   "min_stake",
			"values": "1",
		}, false)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})
}
