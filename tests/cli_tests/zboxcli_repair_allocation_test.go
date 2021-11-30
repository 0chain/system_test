package cli_tests

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/require"
)

func TestRepairAllocation(t *testing.T) {
	t.Parallel()

	t.Run("Repair not required", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// now try to repair allocation to different folder
		// Create a folder to keep all the generated files to be uploaded
		err = os.MkdirAll("tmp_repair", os.ModePerm)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
			"rootpath":   "tmp_repair/",
		})

		output, err = repairAllocation(t, walletOwner, configPath, params, false)
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("Repair file completed, Total files repaired:  0"), output[0])
	})

	t.Run("Attempt file repair on single file that needs repaired", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocationWithParity(t, configPath, walletOwner, 3, 3)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// getting any single blobber
		allocation := getAllocation(t, allocationID)
		require.Len(t, allocation.Blobbers, 6)

		// deleting the file in single blobber
		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"remotepath": file,
			"blobber_url":   allocation.Blobbers[1].BaseUrl,
		})
		output, err = deleteFile(t, walletOwner, params, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("%s deleted", file), output[1])

		// now try to repair allocation to different folder
		// Create a folder to keep all the generated files to be uploaded
		err = os.MkdirAll("tmp_repair", os.ModePerm)
		require.Nil(t, err)

		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
			"rootpath":   "tmp_repair/",
		})

		output, err = repairAllocation(t, walletOwner, configPath, params, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("Repair file completed, Total files repaired:  %s", "1"), output[len(output)-1])
	})

	t.Run("Attempt file repair on multiple files that needs repaired", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocationWithParity(t, configPath, walletOwner, 3, 3)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		file2 := generateRandomTestFileName(t)
		file2Size := int64(256)
		err = createFileWithSize(file2, file2Size)
		require.Nil(t, err)

		files:= []string{file, file2}
		uploadAndDeleteInSingleBlobber(t, walletOwner, allocationID, files)

		// now try to repair allocation to different folder
		// Create a folder to keep all the generated files to be uploaded
		err = os.MkdirAll("tmp_repair", os.ModePerm)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
			"rootpath":   "tmp_repair/",
		})

		output, err := repairAllocation(t, walletOwner, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("Repair file completed, Total files repaired:  %s", "2"), output[len(output)-1])
	})

	t.Run("Attempt file repair on file that does need repaired with a file that does not need repaired", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocationWithParity(t, configPath, walletOwner, 3, 3)

		// upload file
		file := generateRandomTestFileName(t)
		fileSize := int64(256)
		err := createFileWithSize(file, fileSize)
		require.Nil(t, err)

		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		file2 := generateRandomTestFileName(t)
		file2Size := int64(256)
		err = createFileWithSize(file2, file2Size)
		require.Nil(t, err)

		files:= []string{file2}
		uploadAndDeleteInSingleBlobber(t, walletOwner, allocationID, files)

		// now try to repair allocation to different folder
		// Create a folder to keep all the generated files to be uploaded
		err = os.MkdirAll("tmp_repair", os.ModePerm)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
			"rootpath":   "tmp_repair/",
		})

		output, err = repairAllocation(t, walletOwner, configPath, params, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Equal(t, fmt.Sprintf("Repair file completed, Total files repaired:  %s", "1"), output[len(output)-1])
	})

	t.Run("Don't supply repair path", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"rootpath":   "tmp_repair/",
		})

		output, _ := repairAllocation(t, walletOwner, configPath, params, false)
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("Error: repairpath flag is missing"), output[0])
	})

	t.Run("Don't supply root path", func(t *testing.T) {
		t.Parallel()

		walletOwner := escapedTestName(t)
		allocationID, _ := registerAndCreateAllocation(t, configPath, walletOwner)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
		})

		output, _ := repairAllocation(t, walletOwner, configPath, params, false)
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("Error: rootpath flag is missing"), output[0])
	})
}

func uploadAndDeleteInSingleBlobber(t *testing.T, walletOwner, allocationID string, files []string) error {
	for _, file := range files {
		uploadParams := map[string]interface{}{
			"allocation": allocationID,
			"localpath":  file,
			"remotepath": file,
		}
		output, err := uploadFile(t, configPath, uploadParams, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(file)), output[1])

		// getting any single blobber
		allocation := getAllocation(t, allocationID)
		require.Len(t, allocation.Blobbers, 6)

		// deleting the file in single blobber
		params := createParams(map[string]interface{}{
			"allocation":  allocationID,
			"remotepath":  file,
			"blobber_url": allocation.Blobbers[1].BaseUrl,
		})
		output, err = deleteFile(t, walletOwner, params, false)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("%s deleted", file), output[1])
	}
	return nil
}

func repairAllocation(t *testing.T, wallet, cliConfigFilename, params string, retry bool) ([]string, error) {
	t.Logf("Repairing allocation...")
	cmd := fmt.Sprintf("./zbox start-repair --silent %s --wallet %s_wallet.json --configDir ./config --config %s", params, wallet, cliConfigFilename)
	if retry {
		return cliutils.RunCommand(t, cmd, 3, time.Second*2)
	} else {
		return cliutils.RunCommandWithoutRetry(cmd)
	}
}
