package cli_tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"
	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

// maxHealthCheckAge is the maximum allowed age (in seconds) of a blobber's last_health_check
// timestamp. Set to 90 minutes to accommodate a 60-minute health check interval plus startup lag.
const maxHealthCheckAge = int64(90 * 60)

func TestBlobberHealthCheck(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("active blobbers should have a recent last health check timestamp")

	t.Parallel()

	t.Run("active blobbers should have a recent last health check timestamp", func(t *test.SystemTest) {
		createWallet(t)

		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, "error unmarshalling blobber list")
		require.Greater(t, len(blobberList), 0, "blobber list should not be empty")

		now := time.Now().Unix()
		activeCount := 0

		for _, bl := range blobberList {
			if bl.IsKilled || bl.IsShutdown {
				continue
			}
			activeCount++

			require.NotEmpty(t, bl.ID, "active blobber should have an ID")
			require.NotEmpty(t, bl.BaseURL, "active blobber %s should have a base URL", bl.ID)
			require.Greater(t, bl.LastHealthCheck, int64(0),
				"active blobber %s should have a non-zero last_health_check timestamp", bl.ID)

			age := now - bl.LastHealthCheck
			require.Less(t, age, maxHealthCheckAge,
				"blobber %s last_health_check is %ds old (max %ds) - health check transaction may not be sending",
				bl.ID, age, maxHealthCheckAge)
		}

		require.GreaterOrEqual(t, activeCount, minBlobbersForOtherTests,
			"expected at least %d active blobbers, found %d", minBlobbersForOtherTests, activeCount)
		t.Logf("Verified %d active blobbers all have recent last_health_check timestamps", activeCount)
	})

	t.Run("blobbers registered on chain should be usable for new allocations", func(t *test.SystemTest) {
		createWallet(t)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10 * MB,
		})
		createAllocationTestTeardown(t, allocationID)

		require.NotEmpty(t, allocationID, "allocation should be created successfully with registered blobbers")
		t.Logf("Allocation %s created - blobbers are registered and accepting allocations", allocationID)
	})

	t.Run("blobbers should have positive capacity and valid registration data", func(t *test.SystemTest) {
		createWallet(t)

		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, "error unmarshalling blobber list")

		for _, bl := range blobberList {
			if bl.IsKilled || bl.IsShutdown {
				continue
			}
			require.NotEmpty(t, bl.ID, "active blobber should have an ID")
			require.NotEmpty(t, bl.BaseURL, "active blobber %s should have a base URL", bl.ID)
			require.Greater(t, bl.Capacity, int64(0),
				"active blobber %s should have positive total capacity", bl.ID)
		}
	})

	t.Run("individual blobber info should match list blobbers output", func(t *test.SystemTest) {
		createWallet(t)

		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobberList)
		require.Nil(t, err, "error unmarshalling blobber list")

		// Find an active blobber to inspect individually
		var targetBlobber *climodel.BlobberDetails
		for i := range blobberList {
			if !blobberList[i].IsKilled && !blobberList[i].IsShutdown && !blobberList[i].NotAvailable {
				targetBlobber = &blobberList[i]
				break
			}
		}
		require.NotNil(t, targetBlobber, "need at least one active blobber for individual inspection")

		// Fetch the same blobber via bl-info
		infoOutput, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"json":       "",
			"blobber_id": targetBlobber.ID,
		}))
		require.Nil(t, err, strings.Join(infoOutput, "\n"))
		require.Len(t, infoOutput, 1)

		var blobberInfo climodel.BlobberDetails
		err = json.Unmarshal([]byte(infoOutput[0]), &blobberInfo)
		require.Nil(t, err, "error unmarshalling individual blobber info")

		require.Equal(t, targetBlobber.ID, blobberInfo.ID, "blobber ID should match between list and info")
		require.Equal(t, targetBlobber.BaseURL, blobberInfo.BaseURL, "blobber URL should match between list and info")
		require.Equal(t, targetBlobber.IsKilled, blobberInfo.IsKilled, "blobber is_killed should match")
		require.Equal(t, targetBlobber.IsShutdown, blobberInfo.IsShutdown, "blobber is_shutdown should match")
		t.Logf("Blobber %s registration data is consistent across ls-blobbers and bl-info", targetBlobber.ID)
	})

	t.Run("last health check timestamp should not decrease over time", func(t *test.SystemTest) {
		createWallet(t)

		output, err := listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobbersBefore []climodel.BlobberDetails
		err = json.Unmarshal([]byte(output[0]), &blobbersBefore)
		require.Nil(t, err, "error unmarshalling blobber list")

		var targetBlobber *climodel.BlobberDetails
		for i := range blobbersBefore {
			if !blobbersBefore[i].IsKilled && !blobbersBefore[i].IsShutdown && !blobbersBefore[i].NotAvailable {
				targetBlobber = &blobbersBefore[i]
				break
			}
		}
		require.NotNil(t, targetBlobber, "need at least one active blobber")
		t.Logf("Monitoring blobber %s (last_health_check=%d)", targetBlobber.ID, targetBlobber.LastHealthCheck)

		// Wait briefly then re-fetch
		cliutils.Wait(t, 5*time.Second)

		infoOutput, err := getBlobberInfo(t, configPath, createParams(map[string]interface{}{
			"json":       "",
			"blobber_id": targetBlobber.ID,
		}))
		require.Nil(t, err, strings.Join(infoOutput, "\n"))
		require.Len(t, infoOutput, 1)

		var blobberAfter climodel.BlobberDetails
		err = json.Unmarshal([]byte(infoOutput[0]), &blobberAfter)
		require.Nil(t, err, "error unmarshalling blobber info")

		require.GreaterOrEqual(t, blobberAfter.LastHealthCheck, targetBlobber.LastHealthCheck,
			"blobber %s last_health_check should never decrease", targetBlobber.ID)
		t.Logf("Blobber %s last_health_check: before=%d after=%d",
			targetBlobber.ID, targetBlobber.LastHealthCheck, blobberAfter.LastHealthCheck)
	})
}
