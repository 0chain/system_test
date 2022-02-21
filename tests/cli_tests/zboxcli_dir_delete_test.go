package cli_tests

import (
	"encoding/json"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestDeleteDir(t *testing.T) {
	t.Parallel()

	t.Run("delete root dir", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")

		output, err = deleteFile(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
		}), false)
		// FIXME! delete directory is broken. Change the following lines when delete directory fauture is fixed
		t.Log("FIXME! delete directory is broken. Change the following lines when delete directory fauture is fixed")
		require.NotNil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 4)
		require.Contains(t, output[3], "Delete failed. Delete failed: Success_rate:0")
	})
}
