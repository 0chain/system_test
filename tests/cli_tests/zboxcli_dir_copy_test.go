package cli_tests

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestCopyDir(t *testing.T) {
	t.Parallel()

	t.Run("copy root directory", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		dest_dirname := "/destination"
		createDirAndAssert(t, dest_dirname, allocationID)

		var directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 2, "Expecting rootdir created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "destination", Path: "/destination", Type: "d"}, "rootdir expected to be created")

		output, err := copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destpath":   dest_dirname,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		// FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed
		t.Log("FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})

	t.Run("copy nested directory", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		nested_dirname := "/rootdir/nested_directory"
		createDirAndAssert(t, nested_dirname, allocationID)

		dest_dirname := "/destination_direcory"
		createDirAndAssert(t, dest_dirname, allocationID)

		var directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 3, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "nested_directory", Path: "/rootdir/nested_directory", Type: "d"}, "nested_directory expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "destination_direcory", Path: "/destination_direcory", Type: "d"}, "destination_direcory expected to be created")

		output, err := copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": nested_dirname,
			"destpath":   dest_dirname,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		// FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed
		t.Log("FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})

	t.Run("copy directory containing files and folders", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 1 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		nested_dirname := "/rootdir/nested_directory"
		createDirAndAssert(t, nested_dirname, allocationID)

		dest_dirname := "/destination_direcory"
		createDirAndAssert(t, dest_dirname, allocationID)

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 16*KB)
		require.Nil(t, err)

		remotepath := dirname + "/" + filepath.Base(filename)
		output, err := uploadFile(t, configPath, map[string]interface{}{
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

		var directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 4, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		requireExist(t, "/rootdir", directories, "rootdir expected to be created")
		requireExist(t, "/destination_direcory", directories, "destination_direcory expected to be created")
		requireExist(t, "/rootdir/nested_directory", directories, "nested_directory expected to be created")
		requireExist(t, remotepath, directories, "a file expected to be created in /rootdir")

		output, err = copyFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destpath":   dest_dirname,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		// FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed
		t.Log("FIXME! copy directory is broken. Test must be fixed when copy directory fauture is fixed")
		require.Equal(t, "Copy failed: Commit consensus failed", output[0])
	})
}
