package cli_tests

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	climodel "github.com/0chain/system_test/internal/cli/model"
	cliutils "github.com/0chain/system_test/internal/cli/util"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBlobberConfigUpdate(t *testing.T) {
	t.Parallel()

	// Create a folder to keep all the generated files to be uploaded
	err := os.MkdirAll("tmp", os.ModePerm)
	require.Nil(t, err)

	// Success Scenarios

	t.Run("update blobber capacity should work", func(t *testing.T) {
		t.Parallel()

		output, err := registerWallet(t, configPath)
		require.Nil(t, err, "Failed to register wallet", strings.Join(output, "\n"))

		output, err = registerWalletForName(t, configPath, blobberOwnerWallet)
		require.Nil(t, err, "registering wallet failed", err, strings.Join(output, "\n"))

		output, err = listBlobbers(t, configPath, createParams(map[string]interface{}{"json": ""}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var blobberList []climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&blobberList)
		require.Nil(t, err, strings.Join(output, "\n"))

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobberList[0].ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1, strings.Join(output, "\n"))

		var intialBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&intialBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		oldChallengeCompletionTime := intialBlobberInfo.Terms.Challenge_completion_time
		defer func() {
			output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberList[0].ID, "cct": oldChallengeCompletionTime}))
			require.Nil(t, err, strings.Join(output, "\n"))
			require.Len(t, output, 1)
		}()
		newChallengeCompletionTime := 110000 * time.Second

		// BUG: Wallet not able to update blobber details.
		output, err = updateBlobberInfo(t, configPath, createParams(map[string]interface{}{"blobber_id": blobberList[0].ID, "cct": newChallengeCompletionTime}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		output, err = getBlobberInfo(t, configPath, createParams(map[string]interface{}{"json": "", "blobber_id": blobberList[0].ID}))
		require.Nil(t, err, strings.Join(output, "\n"))
		require.Len(t, output, 1)

		var finalBlobberInfo climodel.BlobberDetails
		err = json.NewDecoder(strings.NewReader(output[0])).Decode(&finalBlobberInfo)
		require.Nil(t, err, strings.Join(output, "\n"))

		assert.Equal(t, finalBlobberInfo.Terms.Challenge_completion_time, newChallengeCompletionTime)
	})
}

func getBlobberInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Requesting blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-info %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, escapedTestName(t), cliConfigFilename), 3, time.Second*2)
}

func updateBlobberInfo(t *testing.T, cliConfigFilename, params string) ([]string, error) {
	t.Log("Updating blobber info...")
	return cliutils.RunCommand(t, fmt.Sprintf("./zbox bl-update %s --silent --wallet %s_wallet.json --configDir ./config --config %s", params, blobberOwnerWallet, cliConfigFilename), 3, time.Second*2)
}
