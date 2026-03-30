package cli_tests

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestBlobberConfigUpdate(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("update blobber capacity should work")

	var intialBlobberInfo climodel.BlobberDetails
	t.TestSetup("Create wallet, execute faucet, get blobber details", func() {
		if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
			t.Errorf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
		}

		createWallet(t)

		// Get the blobber owner wallet's client ID so we can select a blobber
		// whose delegate_wallet matches (not all blobbers may use the same delegate wallet)
		blobberOwnerWalletModel, err := getWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "error getting blobber owner wallet")

		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Greater(t, len(blobberList), 0, "blobber list is empty")

		// Collect all blobbers whose delegate_wallet matches blobber_owner_wallet.
		// Sort by most available capacity first (allocated < capacity preferred),
		// so write_price/all-params tests can find a blobber with headroom.
		var candidateBlobbers []climodel.BlobberDetails
		for _, bl := range blobberList {
			if bl.StakePoolSettings.DelegateWallet == blobberOwnerWalletModel.ClientID &&
				!bl.IsKilled && !bl.IsShutdown {
				candidateBlobbers = append(candidateBlobbers, bl)
			}
		}
		require.Greater(t, len(candidateBlobbers), 0,
			"No active blobber found with delegate_wallet matching blobber_owner_wallet (%s)", blobberOwnerWalletModel.ClientID)

		// Pick the blobber with least allocated/capacity ratio (most headroom)
		intialBlobberInfo = candidateBlobbers[0]
		bestRatio := float64(1.0)
		for _, bl := range candidateBlobbers {
			if bl.Capacity > 0 {
				ratio := float64(bl.Allocated) / float64(bl.Capacity)
				if ratio < bestRatio {
					bestRatio = ratio
					intialBlobberInfo = bl
				}
			}
		}
	})

	t.Cleanup(func() {
		createWallet(t)

		// Best-effort cleanup: SC config may have been changed by other tests (e.g., min_write_price),
		// so we log errors instead of failing the test during cleanup.
		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "capacity": intialBlobberInfo.Capacity}))
		if err != nil {
			t.Logf("Cleanup: failed to restore capacity: %s", strings.Join(output, "\n"))
		}

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID}))
		if err != nil {
			t.Logf("Cleanup: failed to update blobber with no params: %s", strings.Join(output, "\n"))
		}

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "num_delegates": intialBlobberInfo.StakePoolSettings.MaxNumDelegates}))
		if err != nil {
			t.Logf("Cleanup: failed to restore num_delegates: %s", strings.Join(output, "\n"))
		}

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "service_charge": intialBlobberInfo.StakePoolSettings.ServiceCharge}))
		if err != nil {
			t.Logf("Cleanup: failed to restore service_charge: %s", strings.Join(output, "\n"))
		}

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "read_price": intToZCN(intialBlobberInfo.Terms.ReadPrice)}))
		if err != nil {
			t.Logf("Cleanup: failed to restore read_price: %s", strings.Join(output, "\n"))
		}

		// Use a write_price of at least 0.1 to stay above any min_write_price that may have been set by other tests
		restoreWritePrice := intToZCN(intialBlobberInfo.Terms.WritePrice)
		if restoreWritePrice < 0.1 {
			restoreWritePrice = 0.1
		}
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "write_price": restoreWritePrice}))
		if err != nil {
			t.Logf("Cleanup: failed to restore write_price: %s", strings.Join(output, "\n"))
		}

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "url": intialBlobberInfo.BaseURL}))
		if err != nil {
			t.Logf("Cleanup: failed to restore url: %s", strings.Join(output, "\n"))
		}
	})

	// update blobber: managing wallet should be able to udpate delegate wallet
	t.RunSequentially("update blobber managing wallet should be able to update delegate wallet", func(t *test.SystemTest) {
		createWallet(t)

		fmt.Println("delegate wallet: ", intialBlobberInfo.StakePoolSettings.DelegateWallet)
		// create a delegate wallet and fund it so it can be used for cleanup (reverting delegate_wallet)
		createWalletForName(escapedTestName(t) + "_delegate")
		_, _ = executeFaucetWithTokensForWallet(t, escapedTestName(t)+"_delegate", configPath, 9)
		delegateWallet, err := getWalletForName(t, configPath, escapedTestName(t)+"_delegate")
		require.Nil(t, err, "error occurred when getting delegate wallet")

		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":      intialBlobberInfo.ID,
			"delegate_wallet": delegateWallet.ClientID,
		}))

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		updatedOutput, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(updatedOutput, "\n"))
		require.Len(t, updatedOutput, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(updatedOutput[0]), &finalBlobberInfo)

		require.Nil(t, err, strings.Join(updatedOutput, "\n"))

		fmt.Println("upadted output : \n\n", finalBlobberInfo)

		require.Equal(t, delegateWallet.ClientID, finalBlobberInfo.StakePoolSettings.DelegateWallet)

		// revert back the delegate wallet id to original one
		// Must use the NEW delegate wallet (not blobberOwnerWallet) since the delegate was just changed
		newDelegateWalletName := escapedTestName(t) + "_delegate"
		t.Cleanup(func() {
			output, err := updateBlobberInfoForWallet(t, configPath, createParams(map[string]interface{}{
				"blobber_id":      intialBlobberInfo.ID,
				"delegate_wallet": intialBlobberInfo.StakePoolSettings.DelegateWallet,
			}), newDelegateWalletName)
			if err != nil {
				t.Logf("Cleanup: error reverting delegate wallet using new delegate: %s", strings.Join(output, "\n"))
				return
			}

			output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
			if err != nil {
				t.Logf("Cleanup: error verifying delegate wallet revert: %s", strings.Join(output, "\n"))
				return
			}
			if len(output) > 0 {
				var finalBlobberInfo climodel.BlobberDetails
				err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
				if err == nil {
					t.Logf("Cleanup: delegate wallet reverted to %s (expected %s)", finalBlobberInfo.StakePoolSettings.DelegateWallet, intialBlobberInfo.StakePoolSettings.DelegateWallet)
				}
			}
		})
	})

	t.RunSequentially("update blobber capacity should work", func(t *test.SystemTest) {
		// create wallet for normal user
		createWallet(t)

		newCapacity := 301 * GB

		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "capacity": newCapacity}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, int64(newCapacity), finalBlobberInfo.Capacity)
	})

	t.RunSequentially("update blobber number of delegates should work", func(t *test.SystemTest) {
		createWallet(t)

		newNumberOfDelegates := 5 // Must be within the chain's max_delegates (typically 10)

		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "num_delegates": newNumberOfDelegates}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// Wait for SC to commit the settings change before reading back
		cliutil.Wait(t, 5*time.Second)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newNumberOfDelegates, finalBlobberInfo.StakePoolSettings.MaxNumDelegates)
	})

	t.RunSequentially("update blobber service charge should work", func(t *test.SystemTest) {
		createWallet(t)

		newServiceCharge := 0.1

		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "service_charge": newServiceCharge}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newServiceCharge, finalBlobberInfo.StakePoolSettings.ServiceCharge)
	})

	t.RunSequentially("update no params should work", func(t *test.SystemTest) {
		createWallet(t)

		// Get current blobber info to check if delegate wallet matches blobberOwnerWallet
		output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var currentBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &currentBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		// If delegate wallet was changed by a previous test, revert it to the original
		if currentBlobberInfo.StakePoolSettings.DelegateWallet != intialBlobberInfo.StakePoolSettings.DelegateWallet {
			// Revert to original delegate wallet using the managing wallet (blobberOwnerWallet should be the managing wallet)
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
				"blobber_id":      intialBlobberInfo.ID,
				"delegate_wallet": intialBlobberInfo.StakePoolSettings.DelegateWallet,
			}))
			require.Nil(t, err, "error reverting delegate wallet: %v", strings.Join(output, "\n"))
		}

		// Now update with no params - this should work with blobberOwnerWallet
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME: since we are not updating any params, the output should not say `updated successfully`
		require.Equal(t, "blobber settings updated successfully", output[0])
	})

	t.RunSequentially("update without blobber ID should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := updateBlobberInfo(t, configPath, "")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 32, "Expected length", len(output))
		require.Equalf(t, "Error: required flag(s) \"blobber_id\" not set", output[0], "output was: %s", output[0])
	})

	t.RunSequentially("update with invalid blobber ID should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": "invalid-blobber-id"}))
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, "internal_error: error retrieving blobber invalid-blobber-id, error record not found", output[1])
	})

	t.RunSequentially("update with invalid blobber wallet/owner should fail", func(t *test.SystemTest) {
		createWallet(t)

		output, err := cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update --silent --wallet %s_wallet.json --configDir ./config --config %s %s", escapedTestName(t), configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID})), 1, time.Second*2)
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "update_blobber_settings_failed: access denied, allowed for delegate_wallet owner only",
			output[0], strings.Join(output, "\n"))
	})

	t.RunSequentially("update blobber read price should work", func(t *test.SystemTest) {
		createWallet(t)

		oldReadPrice := intialBlobberInfo.Terms.ReadPrice
		// Use small increment (0.001 ZCN) to stay within max_read_price SC config
		newReadPrice := intToZCN(oldReadPrice) + 0.001

		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "read_price": newReadPrice}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		// Wait for on-chain state to propagate
		cliutils.Wait(t, 5*time.Second)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newReadPrice, intToZCN(finalBlobberInfo.Terms.ReadPrice))
	})

	t.RunSequentially("update blobber write price should work", func(t *test.SystemTest) {
		createWallet(t)

		// Re-read blobber info to get current state
		output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var currentBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &currentBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		// Use a whole-ZCN increment to avoid float rounding between CLI → SC → readback.
		// Small increments like +0.01 can lose precision in the ZCN↔SAS conversion chain.
		newWritePrice := math.Round(intToZCN(currentBlobberInfo.Terms.WritePrice)) + 1.0

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": intialBlobberInfo.ID, "write_price": newWritePrice}))
		if err != nil && strings.Contains(strings.Join(output, "\n"), "staked capacity") {
			t.Errorf("Blobber staked capacity is less than allocated capacity, cannot change write_price")
			return
		}
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newWritePrice, intToZCN(finalBlobberInfo.Terms.WritePrice))
	})

	t.RunSequentially("update all params at once should work", func(t *test.SystemTest) {
		createWallet(t)

		// Re-read the current blobber info (previous tests or other test suites may have changed SC config)
		output, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var currentBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &currentBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		// Use current write_price as base (may have been changed by other tests)
		// Use a whole-ZCN increment to avoid float rounding between CLI → SC → readback.
		// Small increments like +0.01 can lose precision in the ZCN↔SAS conversion chain.
		newWritePrice := math.Round(intToZCN(currentBlobberInfo.Terms.WritePrice)) + 1.0
		if newWritePrice < 0.1 {
			newWritePrice = 0.1
		}
		newServiceCharge := currentBlobberInfo.StakePoolSettings.ServiceCharge + 0.1
		newReadPrice := intToZCN(currentBlobberInfo.Terms.ReadPrice) + 1
		newNumberOfDelegates := currentBlobberInfo.StakePoolSettings.MaxNumDelegates + 1
		newCapacity := currentBlobberInfo.Capacity + 1
		newNotAvailable := !currentBlobberInfo.NotAvailable
		url := fmt.Sprintf("https://dev-5.devnet-0chain.net/testblobber_%d", time.Now().UnixNano())

		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id":     intialBlobberInfo.ID,
			"write_price":    newWritePrice,
			"service_charge": newServiceCharge,
			"read_price":     newReadPrice,
			"num_delegates":  newNumberOfDelegates,
			"capacity":       newCapacity,
			"not_available":  newNotAvailable,
			"url":            url,
		}))
		if err != nil && strings.Contains(strings.Join(output, "\n"), "staked capacity") {
			t.Errorf("Blobber staked capacity is less than allocated capacity, cannot update all params at once")
			return
		}
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		if newNotAvailable {
			t.Cleanup(func() { setNotAvailability(t, intialBlobberInfo.ID, false) })
		}

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, newWritePrice, intToZCN(finalBlobberInfo.Terms.WritePrice))
		require.Equal(t, newServiceCharge, finalBlobberInfo.StakePoolSettings.ServiceCharge)
		require.Equal(t, newReadPrice, intToZCN(finalBlobberInfo.Terms.ReadPrice))
		require.Equal(t, newNumberOfDelegates, finalBlobberInfo.StakePoolSettings.MaxNumDelegates)
		require.Equal(t, newCapacity, finalBlobberInfo.Capacity)
		require.Equal(t, newNotAvailable, finalBlobberInfo.NotAvailable)
		require.Equal(t, url, finalBlobberInfo.BaseURL)
	})

	t.RunSequentially("update base_url should work", func(t *test.SystemTest) {
		createWallet(t)

		url := fmt.Sprintf("https://dev-5.devnet-0chain.net/testblobber_%d", time.Now().UnixNano())

		output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"blobber_id": intialBlobberInfo.ID,
			"url":        url,
		}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "blobber settings updated successfully", output[0])

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": intialBlobberInfo.ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		require.Equal(t, url, finalBlobberInfo.BaseURL)
	})
}

func getBlobberInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-info --silent --wallet %s_wallet.json --configDir ./config --config %s %s", escapedTestName(t), cliConfigFilename, params), 3, time.Second*2)
}

func updateBlobberInfo(t *test.SystemTest, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update --silent --wallet %s_wallet.json --configDir ./config --config %s %s", blobberOwnerWallet, cliConfigFilename, params), 5, time.Second*5)
}
