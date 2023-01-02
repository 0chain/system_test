package cli_tests

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/rand"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/0chain/gosdk/zboxcore/fileref"
	"github.com/0chain/gosdk/zboxcore/sdk"
	"github.com/0chain/gosdk/zboxcore/zboxutil"
	"github.com/0chain/system_test/internal/api/util/test"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/require"
)

func TestRepairCopy(t *testing.T) {
	// todo: copy a file by changing the upload mask
	// perform repair on that file
}

func TestRepairMove(t *testing.T) {
	// todo: move a file by changing the upload mask
	// perform repair on that file
}

func TestRepairRename(t *testing.T) {
	// todo: rename a file by changing the upload mask
	// perform repair on that file
}

func TestRepairReplaceBlobber(testSetup *testing.T) {
	t := test.NewSystemTest(testSetup)
	t.Run("repair with replace blobber Should Work", func(t *test.SystemTest) {
		allocationID := setupAllocationAndReadLock(t, configPath, map[string]interface{}{
			"lock":   "0.5",
			"size":   10 * MB,
			"data":   1,
			"parity": 2,
			"tokens": 1, // tokens to lock for read pool
		})

		remotepath := "/dir/"
		filesize := 2 * MB
		filename := generateRandomTestFileName(t)

		err := createFileWithSize(filename, int64(filesize))
		require.Nil(t, err)

		remotePath := remotepath + filepath.Base(filename)
		output, err := uploadFile(t, configPath, map[string]interface{}{
			"allocation": allocationID,
			"remotepath": remotePath,
			"localpath":  filename,
		}, true)
		require.Nil(t, err, "error uploading file", strings.Join(output, "\n"))

		alloc, err := sdk.GetAllocation(allocationID)
		if err != nil {
			log.Fatal("Error fetching the allocation")
		}

		blobbers, err := sdk.GetBlobbers(true)
		if err != nil {
			log.Fatal("Error fetching blobbers")
		}

		allocBlobberMap := map[string]bool{}
		for _, blobber := range alloc.Blobbers {
			allocBlobberMap[blobber.ID] = true
		}

		// pick a random blobber from an allocation to be removed from allocation
		rand.Seed(time.Now().Unix())
		removeBlobber := alloc.Blobbers[rand.Intn(len(alloc.Blobbers))] // nolint

		// pick a new blobber that is not part of the current allocation
		var newBlobber *sdk.Blobber
		for _, blobber := range blobbers {
			if _, ok := allocBlobberMap[string(blobber.ID)]; !ok {
				newBlobber = blobber
			}
		}

		// replace 1 blobber with an other
		params := createParams(map[string]interface{}{
			"allocation":     allocationID,
			"add_blobber":    newBlobber.ID,
			"remove_blobber": removeBlobber.ID,
		})
		output, err = replaceBlobber(t, params)
		require.Nil(t, err, strings.Join(output, "\n"))

		// perform repair
		params = createParams(map[string]interface{}{
			"allocation": allocationID,
			"repairpath": remotePath,
			"rootpath":   "/tmp",
		})
		output, err = repair(t, params)
		require.Nil(t, err, strings.Join(output, "\n"))

		// check if the new blobber has the file
		err = checkBlobberHasfile(t, newBlobber.BaseURL, remotePath, allocationID)
		require.Nil(t, err, "error occurred when check if the blobber has the file")
	})
}

func replaceBlobber(t *test.SystemTest, params string) ([]string, error) {
	t.Log("replacing blobbers in an allocation ...")
	cmd := fmt.Sprintf("./zbox updateallocation %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), configPath)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func repair(t *test.SystemTest, params string) ([]string, error) {
	t.Log("repair a file ...")
	cmd := fmt.Sprintf("./zbox start-repair %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), configPath)
	return cliutils.RunCommand(t, cmd, 3, time.Second*2)
}

func checkBlobberHasfile(t *test.SystemTest, blobberURL, file, allocationID string) error {
	var s strings.Builder
	authTokenBytes := make([]byte, 0)
	pathHash := ""
	httpreq, err := zboxutil.NewListRequest(blobberURL, allocationID, file, pathHash, string(authTokenBytes))
	if err != nil {
		return err
	}

	targetWallet, err := getWalletForName(t, configPath, escapedTestName(t))
	require.Nil(t, err, "Error occurred when retrieving curator wallet")

	httpreq.Header.Set("X-App-Client-ID", targetWallet.ClientID)
	httpreq.Header.Set("X-App-Client-Key", targetWallet.ClientPublicKey)

	ctx, cncl := context.WithTimeout(context.TODO(), (time.Second * 30))
	defer cncl()
	err = zboxutil.HttpDo(ctx, cncl, httpreq, func(resp *http.Response, err error) error {
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		resp_body, err := io.ReadAll(resp.Body)
		if err != nil {
			return errors.Wrap(err, "Error: Resp")
		}
		_, err = s.WriteString(string(resp_body))
		if err != nil {
			return err
		}
		if resp.StatusCode == http.StatusOK {
			listResult := &fileref.ListResult{} // todo: validate the list resp
			err = json.Unmarshal(resp_body, listResult)
			if err != nil {
				return errors.Wrap(err, "list entities response parse error:")
			}
			return nil
		}
		return fmt.Errorf("error from server list response: %s", s.String())
	})
	return err
}
