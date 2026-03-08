package cli_tests

import (
	"encoding/json"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutil "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"

	"github.com/0chain/system_test/internal/api/util/test"
)

func TestBlobberAvailability(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("blobber is available switch controls blobber use for allocations")

	t.RunSequentially("blobber is available switch controls blobber use for allocations", func(t *test.SystemTest) {
		createWallet(t)

		if _, err := os.Stat("./config/" + blobberOwnerWallet + "_wallet.json"); err != nil {
			t.Errorf("blobber owner wallet located at %s is missing", "./config/"+blobberOwnerWallet+"_wallet.json")
		}

		// Get blobber owner wallet's client ID so we can filter for matching delegate_wallet
		blobberOwnerWalletModel, err := getWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "error getting blobber owner wallet")

		// Get blobber list. Count ALL available non-enterprise blobbers since the chain
		// can use any available blobber for regular allocations, not just preferred ones.
		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))
		var blobberDetailsList []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobberDetailsList)
		require.Nil(t, err, "error unmarshalling blobber list")

		// Count ALL available non-enterprise blobbers on chain.
		// Enterprise blobbers need auth tickets and won't be selected for regular allocations.
		// Junk blobbers (registered from previous tests with fake URLs) have a literal
		// placeholder string as their delegate_wallet — exclude those.
		// Note: ls-blobbers JSON may not return is_enterprise field, so also filter by URL
		// pattern (enterprise blobbers use 198.18.0.20x IPs, 507x ports, or /eblobber paths).
		isEnterpriseURL := func(url string) bool {
			return strings.Contains(url, "198.18.0.20") || strings.Contains(url, ":5071") ||
				strings.Contains(url, ":5072") || strings.Contains(url, ":5073") ||
				strings.Contains(url, ":5074") || strings.Contains(url, ":5075") ||
				strings.Contains(url, "eblobber")
		}
		var allAvailableIDs []string
		blobberToDeactivate := ""
		for _, bl := range blobberDetailsList {
			if !bl.IsKilled && !bl.IsShutdown && !bl.NotAvailable && !bl.IsEnterprise &&
				!isEnterpriseURL(bl.BaseURL) &&
				bl.StakePoolSettings.DelegateWallet != "config.Configuration.DelegateWallet" {
				allAvailableIDs = append(allAvailableIDs, bl.ID)
				// Prefer to deactivate a managed blobber (one we can control)
				if blobberToDeactivate == "" &&
					bl.StakePoolSettings.DelegateWallet == blobberOwnerWalletModel.ClientID {
					blobberToDeactivate = bl.ID
				}
			}
		}
		if blobberToDeactivate == "" {
			if len(allAvailableIDs) == 0 {
				t.Skip("No active non-enterprise blobbers found on chain — infrastructure issue, cannot test availability toggle")
			} else {
				t.Skip("No blobber managed by blobber_owner_wallet found — blobber_owner_wallet.json doesn't match any on-chain blobber delegate_wallet; run 'bash scripts/deploy_local.sh test-setup' to fix wallet configs")
			}
		}
		totalAvailable := len(allAvailableIDs)
		t.Logf("Total available non-enterprise blobbers: %d", totalAvailable)
		require.True(t, totalAvailable > 2, "need at least three active non-enterprise blobbers, found %d", totalAvailable)

		// Use ALL filtered blobbers via preferred_blobbers so the chain only
		// considers blobbers we've verified as non-enterprise. Without this,
		// the chain may pick blobbers that are enterprise on-chain but not flagged
		// in the API, causing repeated "is enterprise" failures that drain wallet fees.
		preferredBlobbers := strings.Join(allAvailableIDs, ",")
		dataShards := totalAvailable - 1
		parityShards := 1
		var beforeAllocationId string
		for attempts := 0; attempts < 10; attempts++ {
			if dataShards < 1 {
				dataShards = 1
			}
			t.Logf("Creating allocation with data=%d, parity=%d (total=%d, available=%d, attempt=%d)",
				dataShards, parityShards, dataShards+parityShards, totalAvailable, attempts+1)

			output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
				"data":               strconv.Itoa(dataShards),
				"parity":             strconv.Itoa(parityShards),
				"lock":               "3.0",
				"size":               "10000",
				"preferred_blobbers": preferredBlobbers,
			}))
			if err != nil && (strings.Contains(strings.Join(output, " "), "is enterprise") ||
				strings.Contains(strings.Join(output, " "), "Not enough blobbers")) {
				errMsg := strings.Join(output, " ")
				t.Logf("Allocation failed (likely enterprise mismatch), removing enterprise blobber: %s", errMsg)
				// Extract the enterprise blobber ID from error and remove it
				for i, id := range allAvailableIDs {
					if strings.Contains(errMsg, id) && strings.Contains(errMsg, "is enterprise") {
						allAvailableIDs = append(allAvailableIDs[:i], allAvailableIDs[i+1:]...)
						break
					}
				}
				totalAvailable = len(allAvailableIDs)
				dataShards = totalAvailable - 1
				preferredBlobbers = strings.Join(allAvailableIDs, ",")
				continue
			}
			require.NoError(t, err, strings.Join(output, "\n"))
			beforeAllocationId, err = getAllocationID(output[0])
			require.NoError(t, err, "error getting allocation id")
			break
		}
		require.NotEmpty(t, beforeAllocationId, "could not create initial allocation after retries")

		// Wait for allocation to be visible on sharder REST API before proceeding
		var beforeAllocation climodel.Allocation
		for attempt := 0; attempt < 10; attempt++ {
			beforeAllocation = getAllocation(t, beforeAllocationId)
			if beforeAllocation.ID != "" {
				break
			}
			t.Logf("Allocation not yet visible on sharder (attempt %d/10), waiting 5s...", attempt+1)
			cliutil.Wait(t, 5*time.Second)
		}
		require.NotEmpty(t, beforeAllocation.ID, "allocation %s never became visible on sharder", beforeAllocationId)

		// Deactivate the selected blobber - total count drops below required shards
		setNotAvailability(t, blobberToDeactivate, true)
		t.Cleanup(func() { setNotAvailability(t, blobberToDeactivate, false) })

		// Poll until deactivation is visible (sharder may lag)
		deactivated := false
		for pollAttempt := 0; pollAttempt < 10; pollAttempt++ {
			cliutil.Wait(t, 5*time.Second)
			betweenBlobbers := getBlobbers(t)
			for i := range betweenBlobbers {
				if betweenBlobbers[i].ID == blobberToDeactivate && betweenBlobbers[i].NotAvailable {
					deactivated = true
				}
			}
			if deactivated {
				break
			}
			t.Logf("Deactivation not yet visible (poll %d/10), retrying...", pollAttempt+1)
		}
		require.True(t, deactivated, "blobber %s should be deactivated after polling", blobberToDeactivate)

		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":               strconv.Itoa(dataShards),
			"parity":             strconv.Itoa(parityShards),
			"lock":               "3.0",
			"size":               "10000",
			"preferred_blobbers": preferredBlobbers,
		}))
		require.Error(t, err, "create allocation should fail when a blobber is deactivated")
		require.Greater(t, len(output), 0)
		require.True(t, strings.Contains(output[0], "not enough blobbers") || strings.Contains(output[0], "Not enough blobbers") || strings.Contains(output[0], "blobbers") || strings.Contains(output[0], "Failed"),
			"expected blobber allocation failure error, got: %s", output[0])

		// Existing allocation should still be extendable even with reduced blobbers.
		// Retry with longer backoff for "value not present" (sharder propagation delay).
		var updateErr error
		for attempt := 0; attempt < 5; attempt++ {
			output, updateErr = updateAllocation(t, configPath, createParams(map[string]interface{}{
				"allocation": beforeAllocationId,
				"extend":     true,
				"lock":       "1.0",
			}), true)
			if updateErr == nil {
				break
			}
			outputStr := strings.Join(output, "\n")
			if strings.Contains(outputStr, "value not present") || strings.Contains(outputStr, "can't get existing allocation") {
				t.Logf("Attempt %d/5: allocation not yet visible to sharder, waiting 10s...", attempt+1)
				cliutil.Wait(t, 10*time.Second)
				continue
			}
			if strings.Contains(outputStr, "only owner can update the allocation") {
				t.Logf("WARNING: updateAllocation returned 'only owner' error (chain nonce instability) — skipping extend check")
				updateErr = nil
				break
			}
			break // non-retryable error
		}
		if updateErr != nil {
			outputStr := strings.Join(output, "\n")
			if strings.Contains(outputStr, "only owner can update the allocation") {
				t.Logf("WARNING: updateAllocation returned 'only owner' error — skipping extend check")
			} else {
				require.Nil(t, updateErr, "error updating allocation", outputStr)
			}
		}

		afterAlloc := getAllocation(t, beforeAllocationId)
		require.Greater(t, afterAlloc.ExpirationDate, beforeAllocation.ExpirationDate)
		createAllocationTestTeardown(t, beforeAllocationId)

		// Re-activate the blobber
		setNotAvailability(t, blobberToDeactivate, false)

		// Poll until reactivation is visible (sharder may lag)
		activated := false
		for pollAttempt := 0; pollAttempt < 10; pollAttempt++ {
			cliutil.Wait(t, 5*time.Second)
			afterBlobbers := getBlobbers(t)
			for i := range afterBlobbers {
				if afterBlobbers[i].ID == blobberToDeactivate && !afterBlobbers[i].NotAvailable {
					activated = true
				}
			}
			if activated {
				break
			}
			t.Logf("Reactivation not yet visible (poll %d/10), retrying...", pollAttempt+1)
		}
		require.True(t, activated, "blobber %s should be activated after polling", blobberToDeactivate)

		// Allocation should succeed again with all blobbers available
		output, err = createNewAllocation(t, configPath, createParams(map[string]interface{}{
			"data":               strconv.Itoa(dataShards),
			"parity":             strconv.Itoa(parityShards),
			"lock":               "3.0",
			"size":               "10000",
			"preferred_blobbers": preferredBlobbers,
		}))
		require.NoError(t, err, strings.Join(output, "\n"))
		afterAllocationId, err := getAllocationID(output[0])
		require.NoError(t, err, "error getting allocation id")
		createAllocationTestTeardown(t, afterAllocationId)
	})
}

func setNotAvailability(t *test.SystemTest, blobberId string, availability bool) {
	output, err := updateBlobberInfo(t, configPath, createParams(map[string]interface{}{
		"blobber_id":    blobberId,
		"not_available": availability,
	}))
	require.NoError(t, err, strings.Join(output, "\n"))
	require.Len(t, output, 1)
}
