package cli_tests

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestFileUpdate(t *testing.T) {
	t.Parallel()

	t.Run("update size of existing file", func(t *testing.T) {
		t.Parallel()

		// this sets allocation of 10MB and locks 0.5 ZCN
		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 10 * MB})

		filesize := int64(0.5 * MB)
		remotepath := "/"
		filename := generateFileAndUpload(t, allocationID, remotepath, filesize)

		cost, unit := uploadCostWithUnit(t, configPath, allocationID, filename)
		expectedUploadCostInZCN := unitToZCN(cost, unit)

		// Expected cost takes into account data+parity, so we divide by that
		expectedUploadCostPerEntity := (expectedUploadCostInZCN / (2 + 2))

		// Wait for write pool blobber balances to be deduced for initial 0.5 MB
		wait(t, time.Minute)

		// Get write pool info before file update
		output := writePoolInfo(t, configPath)
		initialWritePool := []climodel.WritePoolInfo{}
		err := json.Unmarshal([]byte(output[0]), &initialWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, initialWritePool[0].Id)
		require.InEpsilonf(t, 0.5, intToZCN(initialWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), initialWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		newFileSize := 2 * MB
		updateFileWithRandomlyGeneratedData(t, allocationID, filepath.Join("/", filepath.Base(filename)), int64(newFileSize))

		// Wait before fetching final write pool
		wait(t, time.Minute)

		// Get the new Write Pool info after update
		output = writePoolInfo(t, configPath)
		finalWritePool := []climodel.WritePoolInfo{}
		err = json.Unmarshal([]byte(output[0]), &finalWritePool)
		require.Nil(t, err, "Error unmarshalling write pool info", strings.Join(output, "\n"))

		require.Equal(t, allocationID, finalWritePool[0].Id)
		require.InEpsilon(t, (0.5 - expectedUploadCostPerEntity), intToZCN(finalWritePool[0].Balance), epsilon, "Write pool Balance after upload expected to be [%v] but was [%v]", 0.5-expectedUploadCostPerEntity, intToZCN(initialWritePool[0].Balance))
		require.IsType(t, int64(1), finalWritePool[0].ExpireAt)
		require.Equal(t, allocationID, initialWritePool[0].AllocationId, "Check allocation of write pool matches created allocation id")
		require.Less(t, 0, len(initialWritePool[0].Blobber), "Minimum 1 blobber should exist")
		require.Equal(t, true, initialWritePool[0].Locked, "tokens should not have expired by now")

		go createAllocationTestTeardown(t, allocationID)
	})
}
