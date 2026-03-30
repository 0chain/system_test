package cli_tests

import (
	"encoding/json"
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
			t.Errorf("SC owner wallet located at %s is missing", "./config/"+scOwnerWallet+"_wallet.json")
		}

		createWallet(t)
		newOwnerWallet, err = getWallet(t, configPath)
		require.Nil(t, err, "error fetching wallet")

		newOwnerName = escapedTestName(t)
	})

	t.RunSequentiallyWithTimeout("should allow update of owner: StorageSC", 3*time.Minute, func(t *test.SystemTest) {
		ownerKey := "owner_id"

		// Read expected owner from sc_owner_wallet.json
		scWalletPath := "./config/" + scOwnerWallet + "_wallet.json"
		scWalletData, readErr := os.ReadFile(scWalletPath)
		require.NoError(t, readErr, "Cannot read SC owner wallet")
		var scWalletJSON map[string]interface{}
		require.NoError(t, json.Unmarshal(scWalletData, &scWalletJSON), "Cannot parse SC owner wallet")
		oldOwner := scWalletJSON["client_id"].(string)

		// Verify StorageSC owner on-chain matches the sc_owner_wallet
		output, err := getStorageSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		cfgBefore, _ := keyValuePairStringToMap(output)
		if currentOwner := cfgBefore[ownerKey]; currentOwner != oldOwner {
			t.Fatalf("SC owner wallet mismatch: on-chain owner is %s but sc_owner_wallet has %s — run deploy_local.sh redeploy to fix", currentOwner, oldOwner)
		}

		// STEP 1: Transfer ownership to test wallet
		output, err = updateStorageSCConfig(t, scOwnerWallet, map[string]string{
			ownerKey: newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "storagesc smart contract settings updated", output[0], strings.Join(output, "\n"))

		// STEP 2: IMMEDIATELY restore ownership back — before any assertions that
		// could fail and leave the SC owner permanently changed.
		// The new owner is now newOwnerWallet, so we use newOwnerName to restore.
		time.Sleep(5 * time.Second) // brief wait for transfer to commit
		restoreOK := false
		for attempt := 1; attempt <= 5; attempt++ {
			output, err = updateStorageSCConfig(t, newOwnerName, map[string]string{
				ownerKey: oldOwner,
			}, false)
			if err == nil && len(output) >= 2 {
				t.Logf("StorageSC owner restored to %s on attempt %d", oldOwner, attempt)
				restoreOK = true
				break
			}
			t.Logf("Restore attempt %d/5: %v", attempt, err)
			time.Sleep(time.Duration(attempt*3) * time.Second)
		}
		require.True(t, restoreOK, "CRITICAL: Could not restore StorageSC owner — chain may need redeploy")

		// STEP 3: Verify the transfer worked by checking the config after commit period
		// (the owner was transferred to test wallet and back — verify the round-trip)
		time.Sleep(5 * time.Second)
		output, err = getStorageSCConfig(t, configPath, true)
		require.NoError(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		cfgAfter, _ := keyValuePairStringToMap(output)
		require.Equal(t, oldOwner, cfgAfter[ownerKey], "SC owner should be restored to original: %s", oldOwner)
	})

	t.RunSequentially("should allow update of owner: MinerSC", func(t *test.SystemTest) {
		createWallet(t)

		ownerKey := "owner_id"

		// Read expected owner from miner_sc_owner_wallet.json
		scWalletPath := "./config/" + minerScOwnerWallet + "_wallet.json"
		scWalletData, readErr := os.ReadFile(scWalletPath)
		require.NoError(t, readErr, "Cannot read MinerSC owner wallet")
		var scWalletJSON map[string]interface{}
		require.NoError(t, json.Unmarshal(scWalletData, &scWalletJSON), "Cannot parse MinerSC owner wallet")
		oldOwner := scWalletJSON["client_id"].(string)

		// First verify the minerScOwnerWallet can access MinerSC by reading config
		output, err := getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))
		cfgBefore, _ := keyValuePairStringToMap(output)
		currentOwner := cfgBefore[ownerKey]
		if currentOwner != oldOwner {
			t.Fatalf("MinerSC owner on chain (%s) does not match expected owner (%s) in miner_sc_owner_wallet.json — run deploy_local.sh chain to fix", currentOwner, oldOwner)
		}

		t.Cleanup(func() {
			// Use best-effort restore — do NOT use require (panics are swallowed,
			// causing PERMANENT MinerSC owner loss).
			for attempt := 1; attempt <= 3; attempt++ {
				output, err := updateMinerSCConfig(t, newOwnerName, map[string]interface{}{
					"keys":   ownerKey,
					"values": oldOwner,
				}, false)
				if err == nil && len(output) >= 2 {
					t.Logf("MinerSC owner restored to %s on attempt %d", oldOwner, attempt)
					return
				}
				t.Logf("WARNING: MinerSC owner restore attempt %d/3 failed: %v %s", attempt, err, strings.Join(output, "\n"))
				time.Sleep(time.Duration(attempt*5) * time.Second)
			}
			t.Errorf("CRITICAL: Failed to restore MinerSC owner to %s after 3 attempts — MinerSC owner may be permanently changed!", oldOwner)
		})

		output, err = updateMinerSCConfig(t, minerScOwnerWallet, map[string]interface{}{
			"keys":   ownerKey,
			"values": newOwnerWallet.ClientID,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2, strings.Join(output, "\n"))
		require.Equal(t, "minersc smart contract settings updated", output[0], strings.Join(output, "\n"))

		// Wait for the owner update to be committed on chain (same commit period as StorageSC)
		var minerSCCommitPeriod int64 = 200
		lfb := getLatestFinalizedBlock(t)
		updateConfigRound := lfb.Round + (minerSCCommitPeriod - (lfb.Round % minerSCCommitPeriod))
		var frequency time.Duration = 2
		var found bool
		for i := 0; i < int(minerSCCommitPeriod)/int(frequency); i++ {
			t.Logf("waiting for chain to reach round %d (current: %d)...", updateConfigRound, lfb.Round)
			time.Sleep(frequency * time.Second)
			lfb = getLatestFinalizedBlock(t)
			if lfb.Round >= updateConfigRound {
				found = true
				break
			}
		}
		require.True(t, found, "operation timed out to reach valid round — chain is too slow to commit MinerSC config within 200 rounds")

		output, err = getMinerSCConfig(t, configPath, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(output), 0, strings.Join(output, "\n"))

		cfgAfter, _ := keyValuePairStringToMap(output)

		require.Equal(t, newOwnerWallet.ClientID, cfgAfter[ownerKey], "new value [%s] for owner was not set", newOwnerWallet.ClientID)

		// Should fail update with old owner
		output, err = updateMinerSCConfig(t, minerScOwnerWallet, map[string]interface{}{
			"keys":   "min_stake",
			"values": "1",
		}, true)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		require.Equal(t, "update_settings: unauthorized access - only the owner can access", output[0], strings.Join(output, "\n"))
	})
}
