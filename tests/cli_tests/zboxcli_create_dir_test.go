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

		dirname := "/rootdir"
		output, err := createDir(t, configPath, allocID, "/rootdir", true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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

		dirname := "/parent"
		output, err := createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		dirname = "/parent/child"
		output, err = createDir(t, configPath, allocID, "/parent/child", true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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

		longDirName := "/"
		for i := 0; i < 99; i++ {
			longDirName += "a"
		}

		output, err := createDir(t, configPath, allocID, longDirName, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, longDirName+" directory created", output[0])

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		wantFile := climodel.AllocationFile{Name: longDirName[1:], Path: longDirName, Type: "d"}
		require.Len(t, files, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Equal(t, wantFile, files[0])
	})

	t.Run("create attempt with 101-char dirname", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		longDirName := "/"
		for i := 0; i < 101; i++ {
			longDirName += "a"
		}

		output, err := createDir(t, configPath, allocID, longDirName, false)
		require.NotNil(t, err, "expected create dir failure command executed with output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.True(t, strings.HasPrefix(output[0], `CreateDir failed:  {"error":"ERROR: value too long for type character varying(100) (SQLSTATE 22001)"}`), "expected create dir failure command executed with output: ", strings.Join(output, "\n"))

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1, "unexpected output"+strings.Join(output, ", "))
		require.Equal(t, "[]", output[0], "unexpected output"+strings.Join(output, ", "))

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 0)
	})

	t.Run("create dir with existing dirname should fail", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		dirname := "/existingdir"
		output, err := createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = createDir(t, configPath, allocID, dirname, false)
		require.NotNil(t, err, "Expected create dir failure but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.True(t, strings.HasPrefix(output[0], `CreateDir failed:  {"code":"duplicate_file","error":"duplicate_file: File at path already exists"}`), "Expected create dir failure but got output: ", strings.Join(output, "\n"))
	})

	t.Run("create dir with no leading slash should work", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		dirname := "noleadingslash"
		output, err := createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		require.Len(t, files, 1)
		require.Equal(t, dirname, files[0].Name, "Directory must be created", files)
	})

	t.Run("create dir with no leading slash should not create duplicate dir", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		dirname := "/noleadingslash"
		output, err := createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		dirname = "noleadingslash"
		output, err = createDir(t, configPath, allocID, dirname, false)
		require.NotNil(t, err, "Expected create dir failure but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.True(t, strings.HasPrefix(output[0], `CreateDir failed:  {"code":"duplicate_file","error":"duplicate_file: File at path already exists"}`), "Expected create dir failure but got output: "+strings.Join(output, "\n"))

	})

	t.Run("create with existing dir but different case", func(t *testing.T) {
		t.Parallel()

		allocID := setupAllocation(t, configPath)

		dirname := "/existingdir"
		output, err := createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		dirname = "/existingDir"
		output, err = createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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

		dirname := "/nonexistent/child"
		output, err := createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
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

		dirname := "/abc!@#$%^&*()<>{}[]:;'?,."
		output, err := createDir(t, configPath, allocID, dirname, true)
		require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, dirname+" directory created", output[0])

		output, err = listAll(t, configPath, allocID, true)
		require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var files []climodel.AllocationFile
		err = json.Unmarshal([]byte(output[0]), &files)
		require.Nil(t, err, "Error deserializing JSON string `%s`: %v", strings.Join(output, "\n"), err)

		// Directory size is either 6 or 8. After check remove size for assertions after
		for i, dir := range files {
			require.Contains(t, []int{6, 8}, dir.Size, "Directory is not of size 6 or 8: %v", dir)
			files[i].Size = 0
		}

		wantFile := climodel.AllocationFile{Name: dirname[1:], Path: dirname, Type: "d"}
		require.Len(t, files, 1, "Expecting directories created. Possibly `createdir` failed to create on blobbers (error suppressed) or unable to `list-all` from 3/4 blobbers")
		require.Equal(t, wantFile, files[0])
	})

	t.Run("create attempt with missing dirname param", func(t *testing.T) {
		t.Parallel()

		wallet := escapedTestName(t)

		allocID := setupAllocation(t, configPath)

		output, err := createDirForWallet(t, configPath, wallet, true, allocID, false, "", false)
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error: dirname flag is missing", output[0])
	})

	t.Run("create attempt with empty dirname param", func(t *testing.T) {
		t.Parallel()

		wallet := escapedTestName(t)

		allocID := setupAllocation(t, configPath)

		output, err := createDirForWallet(t, configPath, wallet, true, allocID, true, "", false)
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "CreateDir failed:  invalid_name: Invalid name for dir", output[0])
	})

	t.Run("create attempt with missing allocation", func(t *testing.T) {
		t.Parallel()

		wallet := escapedTestName(t)

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

		output, err = createDirForWallet(t, configPath, wallet, false, "", true, "/root", false)
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

		output, err = createDirForWallet(t, configPath, wallet, true, "", true, "/root", false)
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error fetching the allocation allocation_fetch_error: Error fetching the allocation.consensus_failed: consensus failed on sharders", output[0])
	})

	t.Run("create attempt with invalid allocation", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

		output, err = createDir(t, configPath, "invalidallocation", "/root", false)
		require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.Equal(t, "Error fetching the allocation allocation_fetch_error: "+
			"Error fetching the allocation.consensus_failed: consensus failed on sharders", output[0])
	})

	t.Run("create attempt with someone else's allocation", func(t *testing.T) {
		t.Parallel()

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		allocID := setupAllocation(t, configPath)

		output, err := registerWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = createDirForWallet(t, configPath, nonAllocOwnerWallet, true, allocID, true, "/mydir", false)
		require.NotNil(t, err, "Expected create dir failure but got output: ", strings.Join(output, "\n"))
		require.Len(t, output, 1)
		require.True(t, strings.HasPrefix(output[0], `CreateDir failed:  {"code":"invalid_signature","error":"invalid_signature: Invalid signature"}`), "Expected create dir failure but got output: "+strings.Join(output, "\n"))

	})
}

func createDir(t *testing.T, cliConfigFilename, allocationID, dirname string, retry bool) ([]string, error) {
	return createDirForWallet(t, cliConfigFilename, escapedTestName(t), true, allocationID, true, dirname, retry)
}

func createDirForWallet(t *testing.T, cliConfigFilename, wallet string, withAllocationFlag bool, allocationID string, withDirnameFlag bool, dirname string, retry bool) ([]string, error) {
	cmd := "./zbox createdir --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if withAllocationFlag {
		cmd += ` --allocation "` + allocationID + `"`
	}
	if withDirnameFlag {
		cmd += ` --dirname "` + dirname + `"`
	}

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}

func listAll(t *testing.T, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	return listAllWithWallet(t, escapedTestName(t), cliConfigFilename, allocationID, retry)
}

func listAllWithWallet(t *testing.T, wallet, cliConfigFilename, allocationID string, retry bool) ([]string, error) {
	cliutils.Wait(t, 5*time.Second)
	t.Logf("Listing all...")
	cmd := "./zbox list-all --silent --allocation " + allocationID +
		" --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
