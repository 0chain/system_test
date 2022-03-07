package cli_tests

import (
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestDeleteDir(t *testing.T) {
	t.Parallel()

	t.Run("delete empty root directory - Broken", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		var directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")

		output, err := deleteFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
		}), false)
		// FIXME! delete directory is broken. Change the following lines when delete directory fauture is fixed
		t.Log("FIXME! delete directory is broken. Change the following lines when delete directory fauture is fixed")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1)
		require.Contains(t, output[len(output)-1], "Delete failed. Delete failed: Success_rate:0")
	})

	t.Run("delete nested empty directory - Broken", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		dirname2 := "/rootdir/nested_directory"
		createDirAndAssert(t, dirname2, allocationID)

		directories := listAllAndParse(t, allocationID)

		require.Len(t, directories, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "nested_directory", Path: "/rootdir/nested_directory", Type: "d"}, "nested_directory expected to be created")

		output, err := deleteFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname2,
		}), false)
		// FIXME! delete directory is broken. Change the following lines when delete directory fauture is fixed
		t.Log("FIXME! delete directory is broken. Change the following lines when delete directory fauture is fixed")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.GreaterOrEqual(t, len(output), 1)
		require.Contains(t, output[len(output)-1], "Delete failed. Delete failed: Success_rate:0")
	})
}
