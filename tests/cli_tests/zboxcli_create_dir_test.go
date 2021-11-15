package cli_tests

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestCreateDir(t *testing.T) {
	t.Parallel()

	t.Run("create root dir", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "/rootdir")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		wantFile := climodel.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}
		require.Len(t, files, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Equal(t, wantFile, files[0])
	})

	t.Run("create nested dir", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "/parent")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = createDir(t, configPath, allocID, "/parent/child")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		require.Len(t, files, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, files, climodel.AllocationFile{Name: "parent", Path: "/parent", Type: "d"})
		require.Contains(t, files, climodel.AllocationFile{Name: "child", Path: "/parent/child", Type: "d"})
	})

	t.Run("create with 100-char dir", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		b := make([]rune, 100)
		for i := range b {
			b[i] = 'a'
		}
		longDirName := string(b)

		output, err := createDir(t, configPath, allocID, "/"+longDirName)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		wantFile := climodel.AllocationFile{Name: longDirName, Path: "/" + longDirName, Type: "d"}
		require.Len(t, files, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Equal(t, wantFile, files[0])
	})

	t.Run("create attempt with 150-char dir", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		b := make([]rune, 150)
		for i := range b {
			b[i] = 'a'
		}
		longDirName := string(b)

		output, err := createDir(t, configPath, allocID, "/"+longDirName)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: creating dir with very long directory name must throw error explicitly to not give impression it was success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 0)
	})

	t.Run("create with existing dir", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "/existingdir")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = createDir(t, configPath, allocID, "/existingdir")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: creating dir for another allocation must return a message that it was already existing

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8 then remove that size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		wantFile := climodel.AllocationFile{Name: "existingdir", Path: "/existingdir", Type: "d"}
		require.Len(t, files, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Equal(t, wantFile, files[0])
	})

	t.Run("create with existing dir but different case", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "/existingdir")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = createDir(t, configPath, allocID, "/existingDir")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		require.Len(t, files, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, files, climodel.AllocationFile{Name: "existingdir", Path: "/existingdir", Type: "d"})
		require.Contains(t, files, climodel.AllocationFile{Name: "existingDir", Path: "/existingDir", Type: "d"})
	})

	t.Run("create with non-existent parent dir", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "/nonexistent/child")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		require.Len(t, files, 2, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Contains(t, files, climodel.AllocationFile{Name: "nonexistent", Path: "/nonexistent", Type: "d"})
		require.Contains(t, files, climodel.AllocationFile{Name: "child", Path: "/nonexistent/child", Type: "d"})
	})

	t.Run("create with dir containing special characters", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "/abc!@#$%^&*()<>{}[]:;'?,.")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: createdir command has no output on success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		wantFile := climodel.AllocationFile{Name: "abc!@#$%^&*()<>{}[]:;'?,.", Path: "/abc!@#$%^&*()<>{}[]:;'?,.", Type: "d"}
		require.Len(t, files, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Equal(t, wantFile, files[0])
	})

	t.Run("create attempt with invalid dir - no leading slash", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		output, err := createDir(t, configPath, allocID, "noleadingslash")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: creating dir with no leading slash must throw error explicitly to not give impression it was success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 0)
	})

	t.Run("create attempt with missing dirname param", func(t *testing.T) {
		t.Parallel()

		wallet := escapedTestName(t)

		allocID := setupAllocation(t, configPath)

		output, err := createDirForWallet(nil, configPath, wallet, true, allocID, false, "")
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: dirname flag is missing", output[0])

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 0)
	})

	t.Run("create attempt with empty dirname param", func(t *testing.T) {
		t.Parallel()

		wallet := escapedTestName(t)

		allocID := setupAllocation(t, configPath)

		output, err := createDirForWallet(nil, configPath, wallet, true, allocID, true, "")
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "CreateDir failed. invalid_name: Invalid name for dir", output[0])

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 0)
	})

	t.Run("create attempt with missing allocation", func(t *testing.T) {
		t.Parallel()

		wallet := escapedTestName(t)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

		output, err = createDirForWallet(nil, configPath, wallet, false, "", true, "/root")
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: allocation flag is missing", output[0])
	})

	t.Run("create attempt with empty allocation", func(t *testing.T) {
		t.Parallel()

		wallet := escapedTestName(t)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

		output, err = createDirForWallet(nil, configPath, wallet, true, "", true, "/root")
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error fetching the allocation. allocation_fetch_error: Error fetching the allocation.consensus_failed: consensus failed on sharders", output[0])
	})

	t.Run("create attempt with invalid allocation", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

		output, err = createDir(t, configPath, "invalidallocation", "/root")
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error fetching the allocation. allocation_fetch_error: "+
			"Error fetching the allocation.consensus_failed: consensus failed on sharders", output[0])
	})

	t.Run("create attempt with someone else's allocation", func(t *testing.T) {
		t.Parallel()

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		allocID := setupAllocation(t, configPath)

		output, err := registerWalletForName(nil, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = createDirForWallet(nil, configPath, nonAllocOwnerWallet, true, allocID, true, "/mydir")
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 0) // FIXME: creating dir for another allocation must throw error explicitly to not give impression it was success

		output, err = listAll(t, configPath, allocID)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 0)
	})
}

func createDir(t *testing.T, cliConfigFilename, allocationID, dirname string) ([]string, error) {
	return createDirForWallet(t, cliConfigFilename, escapedTestName(t), true, allocationID, true, dirname)
}

func createDirForWallet(t *testing.T, cliConfigFilename, wallet string, withAllocationFlag bool, allocationID string, withDirnameFlag bool, dirname string) ([]string, error) {
	cmd := "./zbox createdir --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if withAllocationFlag {
		cmd += ` --allocation "` + allocationID + `"`
	}
	if withDirnameFlag {
		cmd += ` --dirname "` + dirname + `"`
	}
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func listAll(t *testing.T, cliConfigFilename, allocationID string) ([]string, error) {
	return listAllWithWallet(t, escapedTestName(t), cliConfigFilename, allocationID)
}

func listAllWithWallet(t *testing.T, wallet, cliConfigFilename, allocationID string) ([]string, error) {
	time.Sleep(5 * time.Second)
	t.Logf("Listing all...")
	return cliutils.RunCommand(t, "./zbox list-all --silent --allocation "+allocationID+
		" --wallet "+wallet+"_wallet.json --configDir ./config --config "+cliConfigFilename, 3, time.Second*2)
}
