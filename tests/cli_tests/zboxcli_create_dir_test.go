package cli_tests

import (
	"encoding/json"
	cli_model "github.com/0chain/system_test/internal/cli/model"
	cli_utils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
	"strings"
	"testing"
)

func TestCreateDir(t *testing.T) {

	t.Run("parallel", func(t *testing.T) {
		t.Run("create root dir", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/rootdir")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on success

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			wantFile := cli_model.AllocationFile{Name: "rootdir", Path: "/rootdir", Type: "d"}
			require.Len(t, files, 1)
			require.Equal(t, wantFile, files[0])
		})

		t.Run("create nested dir", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/parent")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on success

			output, err = createDir(t, configPath, allocID, "/parent/child")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on success

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			require.Len(t, files, 2)
			require.Contains(t, files, cli_model.AllocationFile{Name: "parent", Path: "/parent", Type: "d"})
			require.Contains(t, files, cli_model.AllocationFile{Name: "child", Path: "/parent/child", Type: "d"})
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
			require.Len(t, output, 0) // no output on success

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			wantFile := cli_model.AllocationFile{Name: longDirName, Path: "/" + longDirName, Type: "d"}
			require.Len(t, files, 1)
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
			require.Len(t, output, 0) // no output on error

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			require.Len(t, files, 0)
		})

		t.Run("create with existing dir", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/existingdir")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on error

			output, err = createDir(t, configPath, allocID, "/existingdir")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on error

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			wantFile := cli_model.AllocationFile{Name: "existingdir", Path: "/existingdir", Type: "d"}
			require.Len(t, files, 1)
			require.Equal(t, wantFile, files[0])
		})

		t.Run("create with existing dir but different case", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/existingdir")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on success

			output, err = createDir(t, configPath, allocID, "/existingDir")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on success

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			require.Len(t, files, 2)
			require.Contains(t, files, cli_model.AllocationFile{Name: "existingdir", Path: "/existingdir", Type: "d"})
			require.Contains(t, files, cli_model.AllocationFile{Name: "existingDir", Path: "/existingDir", Type: "d"})
		})

		t.Run("create with non-existent parent dir", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/nonexistent/child")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on success

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			require.Len(t, files, 2)
			require.Contains(t, files, cli_model.AllocationFile{Name: "nonexistent", Path: "/nonexistent", Type: "d"})
			require.Contains(t, files, cli_model.AllocationFile{Name: "child", Path: "/nonexistent/child", Type: "d"})
		})

		t.Run("create with dir containing special characters", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "/abc!@#$%^&*()<>{}[]:;'?,.")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on error

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			wantFile := cli_model.AllocationFile{Name: "abc!@#$%^&*()<>{}[]:;'?,.", Path: "/abc!@#$%^&*()<>{}[]:;'?,.", Type: "d"}
			require.Len(t, files, 1)
			require.Equal(t, wantFile, files[0])
		})

		t.Run("create attempt with invalid dir - no leading slash", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "noleadingslash")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on error

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			require.Len(t, files, 0)
		})

		t.Run("create attempt with missing dirname param", func(t *testing.T) {
			t.Parallel()

			allocID := setupAllocation(t, configPath)

			output, err := createDir(t, configPath, allocID, "")
			require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)
			require.Equal(t, "Error: dirname flag is missing", output[0])

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			require.Len(t, files, 0)
		})

		t.Run("create attempt with missing allocation", func(t *testing.T) {
			t.Parallel()

			output, err := registerWallet(t, configPath)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = executeFaucetWithTokens(t, configPath, 1)
			require.Nil(t, err, "faucet execution failed", err, strings.Join(output, "\n"))

			output, err = createDir(t, configPath, "", "/root")
			require.NotNil(t, err, "Expecting create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)                                        // no output on error
			require.Equal(t, "Error: allocation flag is missing", output[0]) // no output on error
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

			output, err := registerWalletForName(configPath, nonAllocOwnerWallet)
			require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

			output, err = createDirForWallet(configPath, nonAllocOwnerWallet, allocID, "/mydir")
			require.Nil(t, err, "Unexpected create dir failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 0) // no output on error

			output, err = listAll(t, configPath, allocID)
			require.Nil(t, err, "Unexpected list all failure %s", strings.Join(output, "\n"))
			require.Len(t, output, 1)

			var files []cli_model.AllocationFile
			err = json.NewDecoder(strings.NewReader(output[0])).Decode(&files)
			require.Nil(t, err, "error deserializing JSON %v", err)

			require.Len(t, files, 0)
		})
	})
}

func createDir(t *testing.T, cliConfigFilename string, allocationID string, dirname string) ([]string, error) {
	return createDirForWallet(cliConfigFilename, escapedTestName(t), allocationID, dirname)
}

func createDirForWallet(cliConfigFilename string, wallet string, allocationID string, dirname string) ([]string, error) {
	cmd := "./zbox createdir --silent --wallet " + wallet + "_wallet.json --configDir ./config --config " + cliConfigFilename
	if allocationID != "" {
		cmd += ` --allocation "` + allocationID + `"`
	}
	if dirname != "" {
		cmd += ` --dirname "` + dirname + `"`
	}
	return cli_utils.RunCommand(cmd)
}

func listAll(t *testing.T, cliConfigFilename string, allocationID string) ([]string, error) {
	return cli_utils.RunCommand("./zbox list-all --silent --allocation " + allocationID +
		" --wallet " + escapedTestName(t) + "_wallet.json --configDir ./config --config " + cliConfigFilename)
}
