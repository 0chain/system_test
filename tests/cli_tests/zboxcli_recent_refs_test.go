package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/util/test"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"

	"github.com/stretchr/testify/require"
)

func TestRecentlyAddedRefs(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.SetSmokeTests("Recently Added Refs Should be listed")

	t.Parallel()

	t.TestSetup("Create tmp dir", func() {
		// Create a folder to keep all the generated files to be uploaded
		err := os.MkdirAll("tmp", os.ModePerm)
		require.Nil(t, err)
	})

	t.Run("Recently Added Refs Should be listed", func(t *test.SystemTest) { //todo: slow
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		t.Logf("Allocation: %s", allocationID)
		t2 := time.Now()
		fileSize := int64(10)
		p := "/d1/d2/d3/d4/d5/d6/"
		fPath := generateRandomTestFileName(t)
		fileName := filepath.Base(fPath)
		remotePath := filepath.Join(p, fileName)
		err := createFileWithSize(fPath, fileSize)
		require.Nil(t, err)

		time.Sleep(time.Second * 30)

		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fPath,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, "upload failed", strings.Join(output, "\n"))
		require.Len(t, output, 3)

		in := time.Since(t2).String()
		t.Log("Recent refs in: ", in)

		output, err = listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"in":         in,
			"page":       1,
			"page_limit": 100,
		}), true)

		require.Nil(t, err, "List recent files failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		result := climodel.RecentlyAddedRefResult{}

		err = json.Unmarshal([]byte(output[len(output)-1]), &result)
		require.Nil(t, err)

		paths, err := cliutils.GetSubPaths(remotePath)

		require.Nil(t, err)
		require.Equal(t, len(paths), result.Offset, "output: ", strings.Join(output, "\n"))
		require.Len(t, result.Refs, len(paths), "output: ", strings.Join(output, "\n"))

		for i := 0; i < len(paths); i++ {
			expectedPath := paths[i]
			actualPath := result.Refs[i].Path
			require.Equal(t, expectedPath, actualPath, "output: ", strings.Join(output, "\n"))
		}
	})

	t.Run("Refs created 30 seconds ago should not be listed with in-date less than 30 seconds", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		fileSize := int64(10)
		p := "/d1/d2/d3/d4/d5/d6/"
		fPath := generateRandomTestFileName(t)
		fileName := filepath.Base(fPath)
		remotePath := filepath.Join(p, fileName)
		err := createFileWithSize(fPath, fileSize)
		require.Nil(t, err)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fPath,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)

		require.Nil(t, err, "upload failed", strings.Join(output, "\n"))
		require.Len(t, output, 3)

		t1 := time.Now()
		time.Sleep(time.Second * 30)

		output, err = listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"in":         time.Since(t1).String(),
			"page":       1,
		}), true)

		require.Nil(t, err, "List recent files failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		result := climodel.RecentlyAddedRefResult{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &result)
		require.Nil(t, err)

		require.Equal(t, 0, result.Offset)
		require.Len(t, result.Refs, 0)
	})

	t.Run("Refs of someone else's allocation should return zero refs", func(t *test.SystemTest) { //todo: slow
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		fileSize := int64(10)
		p := "/d1/d2/d3/d4/d5/d6/"
		fPath := generateRandomTestFileName(t)
		fileName := filepath.Base(fPath)
		remotePath := filepath.Join(p, fileName)
		err := createFileWithSize(fPath, fileSize)
		require.Nil(t, err)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fPath,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)

		require.Nil(t, err, "upload failed", strings.Join(output, "\n"))
		require.Len(t, output, 3)

		nonAllocOwnerWallet := escapedTestName(t) + "_NON_OWNER"

		output, err = createWalletForName(t, configPath, nonAllocOwnerWallet)
		require.Nil(t, err, "creating wallet failed", strings.Join(output, "\n"))

		t1 := time.Now()
		time.Sleep(time.Second * 30)

		output, err = listRecentlyAddedRefsForWallet(
			t, nonAllocOwnerWallet, configPath,
			createParams(map[string]interface{}{
				"allocation": allocationID,
				"json":       "",
				"in":         time.Since(t1).String(),
				"page":       1,
			}), true)

		require.Nil(t, err)
		result := climodel.RecentlyAddedRefResult{}
		err = json.Unmarshal([]byte(output[len(output)-1]), &result)
		require.Nil(t, err)

		require.Len(t, result.Refs, 0)
	})

	t.Run("Invalid parameters should return error", func(t *test.SystemTest) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		output, err := listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"in":         "6m",
			"page":       -1,
		}), true)

		require.Error(t, err)
		aggregatedOutput := strings.ToLower(strings.Join(output, " "))
		require.Contains(t, aggregatedOutput, "invalid argument")

		output, err = listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"in":         "invalid string",
			"page":       1,
		}), true)

		require.Error(t, err)
		aggregatedOutput = strings.ToLower(strings.Join(output, " "))
		require.Contains(t, aggregatedOutput, "invalid argument")
	})
}

func listRecentlyAddedRefs(t *test.SystemTest, cliConfigFilename, param string, retry bool) ([]string, error) {
	return listRecentlyAddedRefsForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func listRecentlyAddedRefsForWallet(t *test.SystemTest, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	t.Log("Listing recently added refs")
	cmd := fmt.Sprintf(
		"./zbox recent-refs %s --silent --wallet %s --configDir ./config --config %s ",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	}
	return cliutils.RunCommandWithoutRetry(cmd)
}
