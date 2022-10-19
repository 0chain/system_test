package cli_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/core/encryption"
	"github.com/0chain/system_test/internal/api/util/crypto"
	climodel "github.com/0chain/system_test/internal/cli/model"

	// "github.com/0chain/system_test/internal/api/model"
	// "github.com/0chain/system_test/internal/api/util/crypto"
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

		output, err = executeFaucetWithTokens(t, configPath, 1)
		require.Nil(t, err, "Unexpected faucet failure", strings.Join(output, "\n"))

		wallet, err := getWallet(t, configPath)
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

		blobberUrl := allocation.Blobbers[0].Baseurl

		keyPair := crypto.GenerateKeys(t, wallet.Mnemonics)
		hash := encryption.Hash(allocation.ID)
		sign := crypto.SignHexString(t, hash, &keyPair.PrivateKey)
		require.Nil(t, err)

		blobberDeleteConnectionRequest := &climodel.BlobberDeleteConnectionRequest{
			ClientKey:       wallet.ClientPublicKey,
			ClientSignature: string(sign),
			ClientID:        wallet.ClientID,
			ConnectionID:    connectionID,
			AllocationID:    allocationID,
			Path:            "/" + filepath.Base(filename),
			URL:             blobberUrl,
			BlobberID:       allocation.Blobbers[0].ID,
		}
		require.NotNil(t, blobberDeleteConnectionRequest)

		deleteBlobberFile(t, blobberDeleteConnectionRequest)

		// now we will try to repair the file and will create another folder to keep the same
		err = os.MkdirAll(os.TempDir(), os.ModePerm)
		require.Nil(t, err)

		params := createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": "/",
			"rootpath":   os.TempDir(),
		})

		walletOwner := escapedTestName(t)
		output, _ = repairAllocation(t, walletOwner, configPath, params, false)
		require.Len(t, output, 1)
		require.Equal(t, fmt.Sprintf("Repair file completed, Total files repaired: %s", "1"), output[len(output)-1])
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

func deleteBlobberFile(t *testing.T, blobberDeleteConnectionRequest *climodel.BlobberDeleteConnectionRequest) {

	query := &url.Values{}

	query.Add("connection_id", blobberDeleteConnectionRequest.ConnectionID)
	query.Add("path", blobberDeleteConnectionRequest.Path)
	query.Add("repair_request", "true")

	url, err := url.Parse(blobberDeleteConnectionRequest.URL)
	require.Nil(t, err)
	url.Path = path.Join(url.Path, "/v1/file/upload/", blobberDeleteConnectionRequest.AllocationID)
	url.RawQuery = query.Encode()
	req, _ := http.NewRequest(http.MethodDelete, url.String(), nil)

	// Setting the request headers
	req.Header.Set("X-App-Client-Id", blobberDeleteConnectionRequest.ClientID)
	req.Header.Set("X-App-Client-Key", blobberDeleteConnectionRequest.ClientKey)
	req.Header.Set("X-App-Client-Signature", blobberDeleteConnectionRequest.ClientSignature)
	// req.Header.Set("X-Content-Type", "multipart/form-data")

	// Sending the request
	client := &http.Client{}
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.NotNil(t, resp)

	// TODO Also add commit request, otherwise file is not yet deleted

	var allocationRoot = blobberDeleteConnectionRequest.AllocationID
	var allocationID = blobberDeleteConnectionRequest.AllocationID
	var size = int64(512 * KB)
	var blobberID = blobberDeleteConnectionRequest.BlobberID
	type Timestamp int64
	var timestamp = Timestamp(time.Now().Unix())
	var clientID = blobberDeleteConnectionRequest.ClientID
	var signature = blobberDeleteConnectionRequest.ClientSignature
	var writeMarker = &climodel.WriteMarker{
		AllocationRoot:       allocationRoot,
		PrevAllocationRoot:   "",
		AllocationID:         allocationID,
		Size:                 size,
		BlobberID:            blobberID,
		WriteMarkerTimeStamp: timestamp,
		ClientID:             clientID,
		Signature:            signature,
	}

	WriteMarkerMarshal, err := json.Marshal(writeMarker)

	if err != nil {
		fmt.Println(err)
		return
	}
	WriteMarkerString := string(WriteMarkerMarshal)

	query = &url.Values{}
	query.Add("connection_id", blobberDeleteConnectionRequest.ConnectionID)
	query.Add("write_marker", WriteMarkerString)

	url = blobberDeleteConnectionRequest.Url + "/v1/file/commit/" + allocationID
	req, _ := http.NewRequest(http.MethodPost, url, nil)
	req.Header.Set("X-App-Client-Id", blobberDeleteConnectionRequest.ClientID)
	req.Header.Set("X-App-Client-Key", blobberDeleteConnectionRequest.ClientKey)
	req.Header.Set("X-App-Client-Signature", blobberDeleteConnectionRequest.ClientSignature)
	resp, err := client.Do(req)
	require.Nil(t, err)
	require.NotNil(t, resp)

}
