package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/core/common"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

type RecentlyAddedRefResult struct {
	TotalPages int    `json:"total_pages"`
	Offset     int    `json:"offset"`
	Refs       []ORef `json:"refs"`
}

type ORef struct {
	SimilarField
	ID int64 `json:"id"`
}

type SimilarField struct {
	Type                string           `json:"type"`
	AllocationID        string           `json:"allocation_id"`
	LookupHash          string           `json:"lookup_hash"`
	Name                string           `json:"name"`
	Path                string           `json:"path"`
	PathHash            string           `json:"path_hash"`
	ParentPath          string           `json:"parent_path"`
	PathLevel           int              `json:"level"`
	Size                int64            `json:"size"`
	ActualFileSize      int64            `json:"actual_file_size"`
	ActualFileHash      string           `json:"actual_file_hash"`
	MimeType            string           `json:"mimetype"`
	ActualThumbnailSize int64            `json:"actual_thumbnail_size"`
	ActualThumbnailHash string           `json:"actual_thumbnail_hash"`
	CreatedAt           common.Timestamp `json:"created_at"`
	UpdatedAt           common.Timestamp `json:"updated_at"`
}

func TestRecentlyAddedRefs(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	t.Run("Recently Added Refs Should be listed", func(t *testing.T) {
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
		t1 := time.Now()
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"localpath":  fPath,
			"remotepath": remotePath,
			"encrypt":    "",
		}, true)
		require.Nil(t, err, "upload failed", strings.Join(output, "\n"))
		require.Len(t, output, 2)

		output, err = listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"from_date":  fmt.Sprintf("%v", time.Since(t1)),
			"page":       1,
		}), true)

		require.Nil(t, err, "List recent files failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		result := RecentlyAddedRefResult{}
		err = json.Unmarshal([]byte(output[0]), &result)
		require.Nil(t, err)

		paths, err := cliutils.GetSubPaths(remotePath)

		require.Nil(t, err)
		require.Equal(t, 1, result.TotalPages)
		require.Equal(t, len(paths), result.Offset)
		require.Len(t, result.Refs, len(paths))

		for i := 0; i < len(paths); i++ {
			expectedPath := paths[i]
			actualPath := result.Refs[i]
			require.Equal(t, expectedPath, actualPath)
		}
	})

	t.Run("Refs created 30 seconds ago should not be listed with from-date less than 30 seconds", func(t *testing.T) {
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
		require.Len(t, output, 2)

		t1 := time.Now()
		time.Sleep(time.Second * 30)

		output, err = listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"from_date":  fmt.Sprintf("%v", time.Since(t1)),
			"page":       1,
		}), true)

		require.Nil(t, err, "List recent files failed", strings.Join(output, "\n"))
		require.Len(t, output, 1)

		result := RecentlyAddedRefResult{}
		err = json.Unmarshal([]byte(output[0]), &result)
		require.Nil(t, err)

		require.Nil(t, err)
		require.Equal(t, 0, result.TotalPages)
		require.Equal(t, 0, result.Offset)
		require.Len(t, result.Refs, 0)

	})

	t.Run("Invalid parameters should return error", func(t *testing.T) {
		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size": 10000,
		})

		_, err = listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"from_date":  "6m",
			"page":       -1,
		}), true)

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid argument")

		_, err = listRecentlyAddedRefs(t, configPath, createParams(map[string]interface{}{
			"allocation": allocationID,
			"json":       "",
			"from_date":  "invalid string",
			"page":       1,
		}), true)

		require.Error(t, err)
		require.Contains(t, err.Error(), "invalid argument")
	})

}

func listRecentlyAddedRefs(t *testing.T, cliConfigFilename, param string, retry bool) ([]string, error) {
	return listRecentlyAddedRefsForWallet(t, escapedTestName(t), cliConfigFilename, param, retry)
}

func listRecentlyAddedRefsForWallet(t *testing.T, wallet, cliConfigFilename, param string, retry bool) ([]string, error) {
	t.Log("Listing recently added refs")
	cmd := fmt.Sprintf(
		"./zbox recent-refs %s --silent --wallet %s --configDir ./config --config %s",
		param,
		escapedTestName(t)+"_wallet.json",
		cliConfigFilename,
	)

	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	}
	return cliutils.RunCommandWithoutRetry(cmd)
}
