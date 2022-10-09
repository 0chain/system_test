package cli_tests

import (
	// "encoding/json"
	// "encoding/json"
	"fmt"
	"net/http"
	// climodel "github.com/0chain/system_test/internal/cli/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/system_test/internal/api/model"
	"github.com/0chain/system_test/internal/api/util/crypto"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/lithammer/shortuuid/v3"
	"github.com/stretchr/testify/require"
	// zeroChainutils "../../internal/api/util";
)

func TestRepairFile(t *testing.T) {
	t.Parallel()

	t.Run("Attempt file repair on the single file that needs repair", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		wallet := loadWallet(t, escapedTestName(t))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		_, err = getWallet(t, configPath)
		// walletOwner := escapedTestName(t)
		require.Nil(t, err, "Error occurred when retrieving wallet")

		// first uploading the file
		allocSize := int64(1 * MB)
		fileSize := int64(512 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 2,
			"data":   2,
		})

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(filename)), output[1])

		allocation := getAllocation(t, allocationID)
		require.Len(t, allocation.Blobbers, 4)

		// Make API call to delete file from a single blobber
		connectionID := shortuuid.New()

		sign, err := crypto.SignHash(allocation.ID, []model.RawKeyPair{})
		require.Nil(t, err)

		resp, err := deleteFile(t, connectionID, wallet, sign, allocation.ID)
		require.Nil(t, err)
		require.NotNil(t, resp)

		// now we will try to repair the file and will create another folder to keep the same
		err = os.MkdirAll(os.TempDir(), os.ModePerm)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
			"rootpath":   os.TempDir(),
		})

		output, _ = repairAllocation(t, walletOwner, configPath, params, false)
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("Repair file completed, Total files repaired: %s", "2"), output[len(output)-1])
	})

	return

	t.Run("Attempt file repair on the file that does not need repair", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Unexpected register wallet failure", strings.Join(output, "\n"))

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		_, err = getWallet(t, configPath)
		walletOwner := escapedTestName(t)
		require.Nil(t, err, "Error occurred when retrieving wallet")

		// first uploading the file
		allocSize := int64(1 * MB)
		fileSize := int64(512 * KB)

		allocationID := setupAllocation(t, configPath, map[string]interface{}{
			"size":   allocSize,
			"parity": 1,
			"data":   1,
		})

		filename := generateRandomTestFileName(t)
		err = createFileWithSize(filename, fileSize)
		require.Nil(t, err)

		output, err = uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": "/",
			"localpath":  filename,
		}, true)
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)

		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 2)
		require.Equal(t, fmt.Sprintf("Status completed callback. Type = application/octet-stream. Name = %s", filepath.Base(filename)), output[1])

		// now we will try to repair the file and will create another folder to keep the same
		err = os.MkdirAll("tmp_repair", os.ModePerm)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
			"rootpath":   "tmp_repair/",
		})

		output, _ = repairAllocation(t, walletOwner, configPath, params, false)
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("Repair file completed, Total files repaired:  0"), output[0])
	})
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

func deleteFile(t *testing.T, connectionId string, wallet *climodel.Wallet, sign []byte, allocationID string) {
	formData := map[string]string{
		"connection_id": connectionID,
		"path":          "/" + filepath.Base(filename),
	}

	headers := map[string]string{
		"X-App-Client-Id":        wallet.ClientID,
		"X-App-Client-Key":       wallet.ClientPublicKey,
		"X-App-Client-Signature": sign,
		"X-Content-Type":         "multipart/form-data",
	}

	url := blobberURL + "/v1/file/upload/" + allocationID
	req, err := http.NewRequest(http.MethodDelete, url, nil)
	req.headers = headers
	req.body = formData

	client = &http.client{}
	resp, err := client.Do(req)
	return resp, err
}
