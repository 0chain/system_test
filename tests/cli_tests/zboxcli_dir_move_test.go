package cli_tests

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestMoveDir(t *testing.T) {
	t.Parallel()

	t.Run("move nested directory", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		dirname2 := "/rootdir/nested_directory"
		output, err = createDir(t, configPath, allocationID, dirname2, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname2+" directory created", output[0])

		dirname3 := "/destination_direcory"
		output, err = createDir(t, configPath, allocationID, dirname3, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname3+" directory created", output[0])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var directories []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &directories)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, directories, 3, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "nested_directory", Path: "/rootdir/nested_directory", Type: "d"}, "nested_directory expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "destination_direcory", Path: "/destination_direcory", Type: "d"}, "destination_direcory expected to be created")

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname2,
			"destpath":   dirname3,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME! move directory is broken. Test must be fixed when move directory fauture is fixed
		t.Log("FIXME! move directory is broken. Test must be fixed when move directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})

	t.Run("move root directory", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		dirname2 := "/destination_direcory"
		output, err = createDir(t, configPath, allocationID, dirname2, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname2+" directory created", output[0])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var directories []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &directories)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, directories, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "destination_direcory", Path: "/destination_direcory", Type: "d"}, "destination_direcory expected to be created")

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destpath":   dirname2,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME! move directory is broken. Test must be fixed when move directory fauture is fixed
		t.Log("FIXME! move directory is broken. Test must be fixed when move directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})

	t.Run("move directory containing files and folders", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 1 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		output, err := createDir(t, configPath, allocationID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, 16*KB)
		require.Nil(t, err)

		remotepath := dirname + "/" + filepath.Base(filename)
		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotepath,
			"localpath":  filename,
		}, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		expected := fmt.Sprintf(
			"Status completed callback. Type = application/octet-stream. Name = %s",
			filepath.Base(filename),
		)
		require.Equal(t, expected, output[1])

		dest_dir := "/destination_direcory"
		output, err = createDir(t, configPath, allocationID, dest_dir, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dest_dir+" directory created", output[0])

		output, err = listAll(t, configPath, allocationID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var directories []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &directories)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, directories, 3, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		requireExist(t, "/rootdir", directories, "rootdir expected to be created")
		requireExist(t, "/destination_direcory", directories, "destination_direcory expected to be created")
		requireExist(t, remotepath, directories, "a file expected to be created in /rootdir")

		output, err = moveFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destpath":   dest_dir,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME! move directory is broken. Test must be fixed when move directory fauture is fixed
		t.Log("FIXME! move directory is broken. Test must be fixed when move directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})
}

func requireExist(t *testing.T, remotePath string, list []climodel.AllocationFile, msgAndArgs ...interface{}) {
	found := false
	for _, b := range list {
		if b.Path == remotePath {
			found = true
			break
		}
	}
	require.True(t, found, msgAndArgs)
}
