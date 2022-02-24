package cli_tests

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	climodel "github.com/0chain/system_test/internal/cli/model"
	"github.com/stretchr/testify/require"
)

func TestRenameDir(t *testing.T) {
	t.Parallel()

	// FIXME!
	t.Run("rename empty nested directory - Broken", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		nested_dirname := "/rootdir/nested_directory"
		createDirAndAssert(t, nested_dirname, allocationID)

		var directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")
		require.Contains(t, directories, climodel.AllocationFile{Name: "nested_directory", Path: "/rootdir/nested_directory", Type: "d"}, "nested_directory expected to be created")

		output, err := renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": nested_dirname,
			"destname":   "nested_directory_renamed",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "/rootdir/nested_directory renamed", output[0])

		directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")

		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")

		// FIXME! command result was successful, but folder is not renamed
		// Fix the following lines after problem is fixed
		require.NotContains(t, directories, climodel.AllocationFile{Name: "nested_directory_renamed", Path: "/rootdir/nested_directory_renamed", Type: "d"}, "rootdir expected to be renamed tp rootdir_renamed")
		require.Contains(t, directories, climodel.AllocationFile{Name: "nested_directory", Path: "/rootdir/nested_directory", Type: "d"}, "rootdir must have been renamed")
	})

	// FIXME!
	t.Run("rename empty root directory - Broken", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath)
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		var directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir expected to be created")

		output, err := renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destname":   "rootdir_renamed",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "/rootdir renamed", output[0])

		directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")

		// FIXME! command result was successful, but folder is not renamed
		// Fix the following lines after problem is fixed
		require.NotContains(t, directories, climodel.AllocationFile{Name: "rootdir_renamed", Path: "/rootdir_renamed", Type: "d"}, "rootdir expected to be renamed tp rootdir_renamed")
		require.Contains(t, directories, climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}, "rootdir must have been renamed")
	})

	t.Run("rename nested directory containing files - Working", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 1 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

		nested_dirname := "/rootdir/nested_directory"
		createDirAndAssert(t, nested_dirname, allocationID)

		filename := generateRandomTestFileName(t)
		err := createFileWithSize(filename, 16*KB)
		require.Nil(t, err)

		file_remote_path := nested_dirname + "/" + filepath.Base(filename)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file_remote_path,
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

		require.Len(t, directories, 3, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		requireExist(t, "/rootdir", directories, "rootdir expected to be created")
		requireExist(t, "/rootdir/nested_directory", directories, "nested_directory expected to be created")
		requireExist(t, file_remote_path, directories, "a file expected to be created in /rootdir")

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": nested_dirname,
			"destname":   "nested_directory_renamed",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "/rootdir/nested_directory renamed", output[0])

		directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 3, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")

		requireExist(t, "/rootdir", directories, "rootdir expected to be created")
		requireExist(t, "/rootdir/nested_directory_renamed", directories, "nested_directory expected to be renamed to nested_directory_renamed")
		requireExist(t, "/rootdir/nested_directory_renamed/"+filepath.Base(filename), directories, "nested_directory expected to be renamed to nested_directory_renamed")
	})

	t.Run("rename root directory containing files - Working", func(t *testing.T) {
		t.Parallel()

		allocationID := setupAllocation(t, configPath, map[string]interface{}{"size": 1 * MB})
		defer createAllocationTestTeardown(t, allocationID)

		dirname := "/rootdir"
		createDirAndAssert(t, dirname, allocationID)

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

		require.Len(t, directories, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		requireExist(t, "/rootdir", directories, "rootdir expected to be created")
		requireExist(t, remotepath, directories, "a file expected to be created in /rootdir")

		output, err = renameFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": dirname,
			"destname":   "rootdir_renamed",
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "/rootdir renamed", output[0])

		directories = listAllAndParse(t, allocationID)

		require.Len(t, directories, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")

		requireExist(t, "/rootdir_renamed", directories, "rootdir expected to be renamed tp rootdir_renamed")
		requireExist(t, "/rootdir_renamed/"+filepath.Base(filename), directories, "rootdir expected to be renamed tp rootdir_renamed")
	})
}
